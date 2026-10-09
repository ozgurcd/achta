package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ozgurcd/achta/internal/claim"
)

func journalCommand(t *testing.T, input string, want int, args ...string) string {
	t.Helper()
	var out, diagnostics bytes.Buffer
	code := 0
	if binary := os.Getenv("ACHTA_JOURNAL_BINARY"); binary != "" {
		command := exec.Command(binary, args...)
		command.Stdin, command.Stdout, command.Stderr = strings.NewReader(input), &out, &diagnostics
		if err := command.Run(); err != nil {
			code = 2
			if failure, ok := err.(*exec.ExitError); ok {
				code = failure.ExitCode()
			}
		}
	} else {
		code = runWithInput(args, strings.NewReader(input), &out, &diagnostics, testVersion)
	}
	if code != want || (want == 0 && diagnostics.Len() != 0) {
		t.Fatalf("journal exit=%d want=%d; diagnostics=%s", code, want, diagnostics.String())
	}
	return out.String()
}

// RULE: JOURNAL-BRIEF-1
func TestJournalBriefLive(t *testing.T) {
	repo := claimFixture(t)
	claimRun(t, repo, 0, "take", "--slice", "JOURNAL")
	runGit(t, repo, "update-ref", "refs/remotes/origin/main", "HEAD")
	writeTestFile(t, filepath.Join(repo, "tracked.txt"), "changed\n")
	writeTestFile(t, filepath.Join(repo, "new.txt"), "new\n")
	out := journalCommand(t, "", 0, "journal", "brief", "--workspace", repo, "--json")
	var doc map[string]any
	if json.Unmarshal([]byte(out), &doc) != nil || doc["schema_version"] != "achta.journal-brief.v1" {
		t.Fatal("missing versioned brief")
	}
	for _, want := range []string{`"slice":"JOURNAL"`, `"dirty_files":2`, `"ahead":0`, `"behind":0`} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %s", want)
		}
	}
	text := journalCommand(t, "", 0, "journal", "brief", "--workspace", repo)
	if strings.Count(text, "\n") > 10 || !strings.Contains(text, "JOURNAL") {
		t.Fatal("brief line bound or claim missing")
	}
}

// RULE: JOURNAL-NOTE-1
func TestJournalNoteRetention(t *testing.T) {
	repo := claimFixture(t)
	claimRun(t, repo, 0, "take", "--slice", "JOURNAL")
	for i := 0; i < 202; i++ {
		journalCommand(t, "", 0, "journal", "note", "--workspace", repo, "--slice", "JOURNAL", fmt.Sprintf("owner answer %03d", i))
	}
	path := filepath.Join(repo, ".git", "achta", "journal", "JOURNAL.log")
	data, err := os.ReadFile(path)
	if err != nil || strings.Count(string(data), "\n") != 200 || strings.Contains(string(data), "answer 001") {
		t.Fatal("note retention failed")
	}
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0600 {
		t.Fatal("notes must be private")
	}
	out := journalCommand(t, "", 0, "journal", "brief", "--workspace", repo)
	if !strings.Contains(out, "answer 199") || !strings.Contains(out, "answer 201") || strings.Contains(out, "answer 198") {
		t.Fatal("brief must show last three notes")
	}
	journalCommand(t, "", 2, "journal", "note", "--workspace", repo, "--slice", "OTHER", "unowned")
	journalCommand(t, "", 2, "journal", "note", "--workspace", repo, "--slice", "../escape", "unsafe")
	journalCommand(t, "", 2, "journal", "note", "--workspace", repo, "--slice", "JOURNAL", "password=fixture")
}

// RULE: JOURNAL-HOOK-1
func TestJournalHookSilent(t *testing.T) {
	repo := claimFixture(t)
	claimRun(t, repo, 0, "take", "--slice", "JOURNAL")
	for _, source := range []string{"compact", "resume", "startup", "clear"} {
		payload, _ := json.Marshal(map[string]string{"source": source, "cwd": repo})
		out := journalCommand(t, string(payload), 0, "hook", "journal")
		if (source == "compact" || source == "resume") != strings.Contains(out, "JOURNAL") {
			t.Fatalf("hook source %s", source)
		}
	}
	for _, input := range []string{"invalid", `{}`, `{"source":"compact","cwd":"/not-present"}`, strings.Repeat("x", 65537)} {
		if out := journalCommand(t, input, 0, "hook", "journal"); out != "" {
			t.Fatal("hook error must be silent")
		}
	}
}

func TestJournalEvidenceAndWorktrees(t *testing.T) {
	repo := claimFixture(t)
	root := filepath.Dir(repo)
	wt := filepath.Join(root, "linked")
	runGit(t, repo, "worktree", "add", "-b", "linked", wt)
	now := time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)
	for _, path := range []string{repo, wt} {
		if _, err := claim.Run("take", claim.Options{Repo: path, Slice: "SHARED", Now: now}); err != nil {
			t.Fatal(err)
		}
	}
	head := strings.TrimSpace(runGit(t, repo, "rev-parse", "HEAD"))
	writeTestFile(t, filepath.Join(repo, "GATE-RUN.txt"), "schema: gate-run.v1\nrepo-head: "+head+"\nplan: test\ntarget: test exit=0\nresult: green\n")
	runGit(t, repo, "add", "GATE-RUN.txt")
	runGit(t, repo, "commit", "-m", "Witness: make verify green at "+head)
	witnessHead := strings.TrimSpace(runGit(t, repo, "rev-parse", "HEAD"))
	if err := os.MkdirAll(filepath.Join(root, "wiki", "platform"), 0700); err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, filepath.Join(root, "wiki", "platform", "decisions.md"), "## Platform\n### P-101 — Choice (2026-10-09)\nBody withheld.\n### P-100 — Older (2026-10-08)\n")
	repos, err := journalRepos(root, now)
	if err != nil || len(repos) != 2 {
		t.Fatalf("claimed worktrees=%d error=%v", len(repos), err)
	}
	for _, r := range repos {
		if err := appendJournalNote(r, "owner approved the revision", now); err != nil {
			t.Fatal(err)
		}
	}
	doc, err := collectJournal(root, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Decisions) != 1 || doc.Decisions[0] != "P-101" {
		t.Fatal("fixed-date decisions mismatch")
	}
	for _, r := range doc.Repositories {
		if len(r.Notes) != 1 {
			t.Fatal("missing note in a claimed worktree")
		}
		if r.Repository == repo {
			if len(r.Gates) != 1 || r.Gates[0].Result != "green" || r.Gates[0].Head != head || r.Gates[0].Witness != witnessHead {
				t.Fatalf("gate metadata mismatch: %+v", r.Gates)
			}
		}
	}
}

func TestJournalRefusesLinkedNotes(t *testing.T) {
	repo := claimFixture(t)
	claimRun(t, repo, 0, "take", "--slice", "JOURNAL")
	dir := filepath.Join(repo, ".git", "achta")
	if err := os.Symlink(t.TempDir(), dir); err != nil {
		t.Fatal(err)
	}
	journalCommand(t, "", 2, "journal", "note", "--workspace", repo, "--slice", "JOURNAL", "owner answer")
	journalCommand(t, "", 2, "journal", "brief", "--workspace", repo)
}

func TestJournalCheckWitness(t *testing.T) {
	repo := claimFixture(t)
	claimRun(t, repo, 0, "take", "--slice", "JOURNAL")
	head := strings.TrimSpace(runGit(t, repo, "rev-parse", "HEAD"))
	writeTestFile(t, filepath.Join(repo, "GATE-RUN.txt"), "schema: gate-run.v1\nrepo-head: "+head+"\nplan: test\ntarget: test exit=0\nresult: green\n")
	runGit(t, repo, "add", "GATE-RUN.txt")
	runGit(t, repo, "commit", "-m", "Witness: make check green at "+head)
	want := strings.TrimSpace(runGit(t, repo, "rev-parse", "HEAD"))
	out := journalCommand(t, "", 0, "journal", "brief", "--workspace", repo, "--json")
	if !strings.Contains(out, `"witness":"`+want+`"`) {
		t.Fatal("make check witness missing")
	}
}

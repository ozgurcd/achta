package claim

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

var fixedTime = time.Date(2026, 10, 6, 12, 0, 0, 0, time.UTC)

func git(t *testing.T, repo string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("fixture git failed: %v: %s", err, out)
	}
	return strings.TrimSpace(string(out))
}

func put(t *testing.T, path, data string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
}

func fixture(t *testing.T) (string, Options) {
	t.Helper()
	repo := t.TempDir()
	git(t, repo, "init", "-b", "main")
	git(t, repo, "config", "user.name", "Fixture")
	git(t, repo, "config", "user.email", "fixture@example.invalid")
	put(t, filepath.Join(repo, "one.txt"), "original\n")
	git(t, repo, "add", "one.txt")
	git(t, repo, "commit", "-m", "initial")
	git(t, repo, "branch", "baseline")
	git(t, repo, "branch", "--set-upstream-to=baseline", "main")
	return repo, Options{Repo: repo, Slice: "A", Now: fixedTime}
}

func run(t *testing.T, verb string, opts Options, status string) Result {
	t.Helper()
	r, err := Run(verb, opts)
	if err != nil || r.Status != status {
		t.Fatalf("%s: status=%s error=%v reason=%s", verb, r.Status, err, r.Reason)
	}
	return r
}

// RULE: CLAIM-OWNERSHIP-1
func TestOwnershipAtFixedClock(t *testing.T) {
	repo, a := fixture(t)
	a.Note = "migration work"
	before := git(t, repo, "status", "--porcelain=v1", "--untracked-files=all")
	first := run(t, "take", a, "pass")
	if first.Claim == nil || first.Claim.Time != fixedTime || first.Claim.Note != a.Note {
		t.Fatal("claim lost holder metadata")
	}
	a.Now = fixedTime.Add(90 * time.Second)
	if again := run(t, "take", a, "pass"); again.AgeSeconds != 90 || again.Claim.Time != fixedTime {
		t.Fatal("same-holder take renewed the claim")
	}
	a.Note = ""
	b := a
	b.Slice = "B"
	for _, verb := range []string{"take", "check", "release"} {
		r := run(t, verb, b, "fail")
		if r.Reason != "held by A; age 90s" {
			t.Fatalf("missing holder age: %s", r.Reason)
		}
	}
	status := a
	status.Slice = ""
	run(t, "status", status, "pass")
	b.Force, b.Reason = true, "owner handoff"
	r := run(t, "release", b, "pass")
	if r.Claim != nil || len(r.Releases) != 1 || r.Releases[0].Holder != "A" || r.Releases[0].By != "B" || r.Releases[0].Reason != b.Reason || !r.Releases[0].Forced {
		t.Fatal("force release lost audit evidence")
	}
	b.Force, b.Reason = false, ""
	run(t, "take", b, "pass")
	r = run(t, "release", b, "pass")
	if len(r.Releases) != 2 || r.Releases[1].Forced {
		t.Fatal("release history was overwritten")
	}
	if got := git(t, repo, "status", "--porcelain=v1", "--untracked-files=all"); got != before {
		t.Fatal("claim operation changed checkout files")
	}
	info, err := os.Stat(filepath.Join(repo, ".git", "achta-claim.json"))
	if err != nil || info.Mode().Perm() != 0600 {
		t.Fatal("claim state must be private Git metadata")
	}
}

func TestCheckoutMeasurements(t *testing.T) {
	repo, opts := fixture(t)
	run(t, "check", opts, "pass")
	put(t, filepath.Join(repo, "untracked.txt"), "not counted")
	run(t, "check", opts, "pass")
	put(t, filepath.Join(repo, "one.txt"), "changed")
	run(t, "check", opts, "fail")
	for _, claimed := range []bool{false, true} {
		if claimed {
			run(t, "take", opts, "pass")
			run(t, "check", opts, "pass")
		}
		status := opts
		status.Slice = ""
		r := run(t, "status", status, "pass")
		if r.TrackedChangeCount != 1 || !reflect.DeepEqual(r.TrackedChanges, []string{"one.txt"}) {
			t.Fatalf("bad tracked paths: %+v", r.Checkout)
		}
	}
	run(t, "release", opts, "pass")
	git(t, repo, "add", "one.txt")
	git(t, repo, "commit", "-m", "change")
	put(t, filepath.Join(repo, "one.txt"), "original\n")
	git(t, repo, "add", "one.txt")
	git(t, repo, "commit", "-m", "undo")
	r := run(t, "check", opts, "fail")
	if r.TrackedChangeCount != 0 || r.UnpushedCommitCount == nil || *r.UnpushedCommitCount != 2 || !reflect.DeepEqual(r.UnpushedFiles, []string{"one.txt"}) {
		t.Fatalf("reverted local commits lost: %+v", r.Checkout)
	}
	git(t, repo, "branch", "--unset-upstream")
	r = run(t, "check", opts, "fail")
	if r.UnpushedCommitCount != nil || r.UpstreamAvailable {
		t.Fatal("missing upstream was reported clean")
	}
	run(t, "take", opts, "pass")
	run(t, "check", opts, "pass")
}

func TestWorktreeIsolation(t *testing.T) {
	repo, a := fixture(t)
	other := filepath.Join(t.TempDir(), "worktree")
	git(t, repo, "worktree", "add", "-b", "other", other)
	b := a
	b.Repo, b.Slice = other, "B"
	run(t, "take", a, "pass")
	run(t, "take", b, "pass")
	run(t, "check", a, "pass")
	run(t, "check", b, "pass")
	dir := git(t, other, "rev-parse", "--absolute-git-dir")
	if _, err := os.Stat(filepath.Join(dir, "achta-claim.json")); err != nil {
		t.Fatal("worktree claim missing from its Git directory")
	}
	if git(t, other, "status", "--porcelain=v1", "--untracked-files=all") != "" {
		t.Fatal("worktree claim became checkout content")
	}
}

func TestConcurrentTakeHasOneWinner(t *testing.T) {
	_, opts := fixture(t)
	start := make(chan struct{})
	winners := make(chan string, 8)
	var group sync.WaitGroup
	for _, name := range []string{"A", "B", "C", "D", "E", "F", "G", "H"} {
		group.Add(1)
		go func(name string) {
			defer group.Done()
			<-start
			local := opts
			local.Slice = name
			r, err := Run("take", local)
			if err == nil && r.Status == "pass" {
				winners <- name
			}
		}(name)
	}
	close(start)
	group.Wait()
	close(winners)
	if len(winners) != 1 {
		t.Fatalf("successful concurrent holders=%d want=1", len(winners))
	}
	opts.Slice = <-winners
	run(t, "check", opts, "pass")
}

func TestRefusalsPreserveState(t *testing.T) {
	repo, opts := fixture(t)
	run(t, "take", opts, "pass")
	path := filepath.Join(repo, ".git", "achta-claim.json")
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for name, change := range map[string]func(*Options){
		"no slice":             func(o *Options) { o.Slice = "" },
		"relative":             func(o *Options) { o.Repo = "." },
		"force without reason": func(o *Options) { o.Force = true },
		"reason without force": func(o *Options) { o.Reason = "handoff" },
		"backward clock":       func(o *Options) { o.Now = fixedTime.Add(-time.Second) },
		"secret":               func(o *Options) { o.Reason = "password=fixture-only"; o.Force = true },
		"control":              func(o *Options) { o.Slice = "B\nC" },
		"oversized":            func(o *Options) { o.Slice = strings.Repeat("a", 129) },
	} {
		t.Run(name, func(t *testing.T) {
			changed := opts
			change(&changed)
			if _, err := Run("release", changed); err == nil {
				t.Fatal("unsafe input accepted")
			}
		})
	}
	after, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("refusal changed state")
	}
	for _, data := range []string{"{}", string(before) + "{}", strings.Replace(string(before), StateSchema, "unknown", 1)} {
		put(t, path, data)
		if _, err := Run("take", opts); err == nil {
			t.Fatal("malformed state accepted")
		}
	}
}

func TestLinkedStateAndLockRefused(t *testing.T) {
	for _, name := range []string{"achta-claim.json", "achta-claim.lock"} {
		t.Run(name, func(t *testing.T) {
			repo, opts := fixture(t)
			external := filepath.Join(t.TempDir(), "untouched")
			put(t, external, "sentinel")
			if err := os.Symlink(external, filepath.Join(repo, ".git", name)); err != nil {
				t.Fatal(err)
			}
			if _, err := Run("take", opts); err == nil {
				t.Fatal("linked metadata accepted")
			}
			data, err := os.ReadFile(external)
			if err != nil || string(data) != "sentinel" {
				t.Fatal("linked destination changed")
			}
		})
	}
}

func TestFixedJSON(t *testing.T) {
	_, opts := fixture(t)
	r := run(t, "take", opts, "pass")
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"schema_version":"achta.claim.v1","operation":"take","status":"pass","claim":{"slice":"A","time":"2026-10-06T12:00:00Z","note":""},"age_seconds":0,"releases":[]}`
	if string(data) != want {
		t.Fatalf("claim machine contract changed: %s", data)
	}
}

func TestGitDirectorySymlinkRefused(t *testing.T) {
	repo, opts := fixture(t)
	elsewhere := filepath.Join(t.TempDir(), "metadata")
	if err := os.Rename(filepath.Join(repo, ".git"), elsewhere); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(elsewhere, filepath.Join(repo, ".git")); err != nil {
		t.Fatal(err)
	}
	if _, err := Run("take", opts); err == nil {
		t.Fatal("linked Git directory accepted for claim writes")
	}
}

func TestUnpushedOddFilenames(t *testing.T) {
	repo, opts := fixture(t)
	name := "\nleading newline.txt"
	put(t, filepath.Join(repo, name), "never printed")
	git(t, repo, "add", "--", name)
	git(t, repo, "commit", "-m", "odd filename")
	opts.Slice = ""
	r := run(t, "status", opts, "pass")
	if !reflect.DeepEqual(r.UnpushedFiles, []string{name}) {
		t.Fatalf("filename changed: %q", r.UnpushedFiles)
	}
}

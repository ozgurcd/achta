package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func gateInvocation(args []string, out, diagnostics *bytes.Buffer) int {
	if binary := os.Getenv("ACHTA_GATE_BINARY"); binary != "" {
		command := exec.Command(binary, args...)
		command.Stdout, command.Stderr = out, diagnostics
		if err := command.Run(); err != nil {
			if failure, ok := err.(*exec.ExitError); ok {
				return failure.ExitCode()
			}
			return 2
		}
		return 0
	}
	return Run(args, out, diagnostics, testVersion)
}

// RULE: GATE-RUN-LOG-1
func TestGateRunFullLogAndExit(t *testing.T) {
	repo := claimFixture(t)
	var out, diagnostics bytes.Buffer
	code := gateInvocation([]string{"gate", "run", "--repo", repo, "--json", "--", "sh", "-c", "pwd; printf 'first\\n'; printf 'error: broken\\n' >&2; exit 7"}, &out, &diagnostics)
	if code != 7 {
		t.Fatalf("exit=%d want=7", code)
	}
	var result struct {
		Schema string `json:"schema_version"`
		Log    string `json:"log_path"`
		Exit   int    `json:"exit_code"`
	}
	if json.Unmarshal(out.Bytes(), &result) != nil || result.Schema != "achta.gate-run.v1" || result.Exit != 7 || diagnostics.Len() != 0 {
		t.Fatal("invalid gate JSON contract")
	}
	canonical, err := filepath.EvalSymlinks(repo)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(result.Log, filepath.Join(canonical, ".git", "achta", "logs")+string(os.PathSeparator)) {
		t.Fatal("log outside checkout metadata")
	}
	log, err := os.ReadFile(result.Log)
	if err != nil || string(log) != repo+"\nfirst\nerror: broken\n" {
		t.Fatal("full combined log or working directory differs")
	}
	info, err := os.Stat(result.Log)
	if err != nil || info.Mode().Perm() != 0600 {
		t.Fatal("log must be private")
	}
}

func TestGateRunSummaryAndRetention(t *testing.T) {
	repo := claimFixture(t)
	credential := "postgres://fixture:" + "fixture-sensitive-value@localhost/test"
	t.Setenv("ACHTA_GATE_FIXTURE", credential)
	for n := 0; n < 22; n++ {
		var out, diagnostics bytes.Buffer
		code := gateInvocation([]string{"gate", "run", "--repo", repo, "--", "sh", "-c", "i=0; while [ $i -lt 100 ]; do printf 'error: fixture %s\\n' $i; i=$((i+1)); done; printf 'error: %s\\n' \"$ACHTA_GATE_FIXTURE\"; exit 3"}, &out, &diagnostics)
		if code != 3 {
			t.Fatalf("exit=%d want=3", code)
		}
		if bytes.Contains(out.Bytes(), []byte(credential)) || bytes.Contains(diagnostics.Bytes(), []byte(credential)) {
			t.Fatal("summary disclosed fixture credential")
		}
		if len(strings.Split(strings.TrimSpace(out.String()+diagnostics.String()), "\n")) > 25 {
			t.Fatal("summary exceeded 25 lines")
		}
		if !strings.Contains(out.String(), "error: fixture 99") || strings.Contains(out.String(), "error: fixture 0\n") || !strings.Contains(out.String(), "details withheld") {
			t.Fatal("last errors or redaction missing")
		}
	}
	logs, err := filepath.Glob(filepath.Join(repo, ".git", "achta", "logs", "*.log"))
	if err != nil || len(logs) != 20 {
		t.Fatalf("retained log count=%d want=20", len(logs))
	}
}

func TestGateRunChildFlagsAndWorktree(t *testing.T) {
	repo := claimFixture(t)
	worktree := filepath.Join(t.TempDir(), "checkout")
	runGit(t, repo, "worktree", "add", "-b", "gate-fixture", worktree)
	var out, diagnostics bytes.Buffer
	code := gateInvocation([]string{"gate", "run", "--repo", worktree, "--", "printf", "%s\\n", "--json", "--help"}, &out, &diagnostics)
	if code != 0 || json.Valid(out.Bytes()) || !strings.Contains(out.String(), "exit code: 0") {
		t.Fatal("child flags changed runner mode")
	}
	if info, err := os.Stat(filepath.Join(worktree, ".git")); err != nil || !info.Mode().IsRegular() {
		t.Fatal("worktree metadata changed")
	}
}

func TestGateRunHelpAndCapabilities(t *testing.T) {
	for _, args := range [][]string{{"gate", "run", "--help"}, {"capabilities", "--json"}} {
		var out, diagnostics bytes.Buffer
		if gateInvocation(args, &out, &diagnostics) != 0 || !strings.Contains(out.String(), "gate run") {
			t.Fatal("gate run missing from help or capabilities")
		}
	}
}

func TestGateRunFreshEvidenceAndStaleFallback(t *testing.T) {
	repo := claimFixture(t)
	for _, fresh := range []bool{true, false} {
		var out, diagnostics bytes.Buffer
		command := "printf 'error: fallback\\n'; exit 9"
		if fresh {
			command = "printf 'evidence: [check] FAIL record detail\\ntarget: check exit=9\\n' > GATE-RUN.txt; " + command
		}
		if gateInvocation([]string{"gate", "run", "--repo", repo, "--", "sh", "-c", command}, &out, &diagnostics) != 9 {
			t.Fatal("child exit changed")
		}
		if fresh && (!strings.Contains(out.String(), "FAIL record detail") || strings.Contains(out.String(), "error: fallback")) {
			t.Fatal("fresh failed-target evidence did not win")
		}
		if !fresh && (!strings.Contains(out.String(), "error: fallback") || strings.Contains(out.String(), "FAIL record detail")) {
			t.Fatal("stale record reused")
		}
	}
}

func TestGateRunRefusesLinkedLogDirectory(t *testing.T) {
	repo := claimFixture(t)
	outside := t.TempDir()
	if os.Symlink(outside, filepath.Join(repo, ".git", "achta")) != nil {
		t.Fatal("symlink fixture")
	}
	var out, diagnostics bytes.Buffer
	if gateInvocation([]string{"gate", "run", "--repo", repo, "--json", "--", "true"}, &out, &diagnostics) != 2 {
		t.Fatal("linked metadata accepted")
	}
	entries, err := os.ReadDir(outside)
	if err != nil || len(entries) != 0 {
		t.Fatal("write escaped Git metadata")
	}
}

func TestGateReleaseHookClaimChain(t *testing.T) {
	for _, shell := range []string{"cd /abs/repo && achta claim take --repo /abs/repo --slice X", "cd /abs/repo\nachta claim take --repo /abs/repo --slice X"} {
		payload, err := json.Marshal(map[string]any{"tool_input": map[string]string{"command": shell}})
		if err != nil {
			t.Fatal(err)
		}
		var out, diagnostics bytes.Buffer
		if binary := os.Getenv("ACHTA_GATE_BINARY"); binary != "" {
			command := exec.Command(binary, "hook", "cd")
			command.Stdin, command.Stdout, command.Stderr = bytes.NewReader(payload), &out, &diagnostics
			if command.Run() != nil {
				t.Fatal("installed hook refused absolute claim chain")
			}
		} else if runWithInput([]string{"hook", "cd"}, bytes.NewReader(payload), &out, &diagnostics, testVersion) != 0 {
			t.Fatal("hook refused absolute claim chain")
		}
		if out.Len()+diagnostics.Len() != 0 {
			t.Fatal("allowed hook emitted diagnostics")
		}
	}
}

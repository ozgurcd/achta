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

func claimFixture(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	runGit(t, repo, "init", "-b", "main")
	runGit(t, repo, "config", "user.name", "Fixture")
	runGit(t, repo, "config", "user.email", "fixture@example.invalid")
	writeTestFile(t, filepath.Join(repo, "tracked.txt"), "initial\n")
	runGit(t, repo, "add", "tracked.txt")
	runGit(t, repo, "commit", "-m", "initial")
	runGit(t, repo, "branch", "baseline")
	runGit(t, repo, "branch", "--set-upstream-to=baseline", "main")
	return repo
}

func claimRun(t *testing.T, repo string, want int, verb string, args ...string) string {
	t.Helper()
	var out, diagnostics bytes.Buffer
	argv := append([]string{"claim", verb, "--repo", repo}, args...)
	code := claimTestCommand(argv, &out, &diagnostics)
	if code != want {
		t.Fatalf("claim %s exit=%d want=%d: %s%s", verb, code, want, out.String(), diagnostics.String())
	}
	if strings.Contains(strings.Join(args, " "), "--json") {
		if !json.Valid(out.Bytes()) || diagnostics.Len() != 0 {
			t.Fatal("claim JSON must be one document with empty stderr")
		}
	}
	return out.String() + diagnostics.String()
}

// The release proof selects the installed executable; normal tests remain
// in-process. Both paths run the identical lifecycle in temporary checkouts.
func claimTestCommand(args []string, stdout, stderr *bytes.Buffer) int {
	if binary := os.Getenv("ACHTA_CLAIM_BINARY"); binary != "" {
		cmd := exec.Command(binary, args...)
		cmd.Stdout, cmd.Stderr = stdout, stderr
		if err := cmd.Run(); err != nil {
			if failure, ok := err.(*exec.ExitError); ok {
				return failure.ExitCode()
			}
			return 2
		}
		return 0
	}
	return Run(args, stdout, stderr, testVersion)
}

func TestClaimLifecycle(t *testing.T) {
	repo := claimFixture(t)
	claimRun(t, repo, 0, "check", "--slice", "A", "--json")
	claimRun(t, repo, 0, "take", "--slice", "A", "--note", "first change", "--json")
	if out := claimRun(t, repo, 1, "take", "--slice", "B"); !strings.Contains(out, "A") || !strings.Contains(out, "age") {
		t.Fatal("take refusal must identify holder and age")
	}
	writeTestFile(t, filepath.Join(repo, "tracked.txt"), "changed\n")
	claimRun(t, repo, 0, "check", "--slice", "A", "--json")
	claimRun(t, repo, 1, "check", "--slice", "B", "--json")
	if out := claimRun(t, repo, 0, "status", "--json"); !strings.Contains(out, "tracked.txt") || !strings.Contains(out, `"tracked_change_count":1`) {
		t.Fatal("status must enumerate tracked changes with a claim")
	}
	claimRun(t, repo, 1, "release", "--slice", "B", "--json")
	claimRun(t, repo, 0, "release", "--slice", "B", "--force", "--reason", "owner handoff", "--json")
	if out := claimRun(t, repo, 0, "status", "--json"); !strings.Contains(out, "owner handoff") || !strings.Contains(out, `"by":"B"`) {
		t.Fatal("forced release must retain actor and reason")
	}
	claimRun(t, repo, 1, "check", "--slice", "B", "--json")
	runGit(t, repo, "add", "tracked.txt")
	runGit(t, repo, "commit", "-m", "local change")
	if out := claimRun(t, repo, 0, "status", "--json"); !strings.Contains(out, `"unpushed_commit_count":1`) || !strings.Contains(out, "tracked.txt") {
		t.Fatal("status must enumerate unpushed commit paths without a claim")
	}
	claimRun(t, repo, 1, "check", "--slice", "B", "--json")
	claimRun(t, repo, 0, "take", "--slice", "B")
	claimRun(t, repo, 0, "release", "--slice", "B")
}

func TestClaimHelpAndCapabilities(t *testing.T) {
	for _, verb := range []string{"take", "status", "check", "release"} {
		var out, diagnostics bytes.Buffer
		if code := Run([]string{"claim", verb, "--help"}, &out, &diagnostics, testVersion); code != 0 || !strings.Contains(out.String(), "--repo ABS") {
			t.Fatalf("claim %s help lacks explicit checkout selector", verb)
		}
		out.Reset()
		Run([]string{"capabilities", "--json"}, &out, &diagnostics, testVersion)
		if !strings.Contains(out.String(), `"name":"claim `+verb+`"`) {
			t.Fatalf("claim %s missing from capabilities", verb)
		}
	}
}

func TestClaimSecretInputsNeverEcho(t *testing.T) {
	repo := claimFixture(t)
	for _, field := range []string{"--slice", "--note", "--reason", "--repo"} {
		t.Run(field, func(t *testing.T) {
			value := "password=" + strings.Repeat("fixture", 3)
			args := []string{"claim", "take", "--repo", repo, "--slice", "A", field, value, "--json"}
			var out, diagnostics bytes.Buffer
			if code := claimTestCommand(args, &out, &diagnostics); code != 2 {
				t.Fatalf("secret-shaped input exit=%d want=2", code)
			}
			if bytes.Contains(out.Bytes(), []byte(value)) || bytes.Contains(diagnostics.Bytes(), []byte(value)) {
				t.Fatal("refusal echoed input")
			}
			if !json.Valid(out.Bytes()) || diagnostics.Len() != 0 {
				t.Fatal("invalid refusal JSON contract")
			}
		})
	}
}

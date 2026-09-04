package witness

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseGreenAndTiming(t *testing.T) {
	record, err := Parse([]byte(`schema: gate-run.v1
gate: sample
repo-head: aaaaaaa
started: 2026-09-04T10:00:00Z
plan: beta alpha
elapsed: beta 1s
target: beta exit=0
elapsed: alpha 2s
target: alpha exit=0
finished: 2026-09-04T10:00:04Z
tree: commit=aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
result: green
`))
	if err != nil {
		t.Fatal(err)
	}
	if len(record.Targets) != 2 || record.ElapsedMS["alpha"] != 2000 {
		t.Fatalf("record = %+v", record)
	}
}

func TestParseRejectsDuplicateAndImpossibleTime(t *testing.T) {
	base := `schema: gate-run.v1
started: 2026-09-04T10:00:04Z
plan: one
target: one exit=0
finished: 2026-09-04T10:00:00Z
`
	if _, err := Parse([]byte(base)); err == nil || !strings.Contains(err.Error(), "precedes") {
		t.Fatalf("err = %v", err)
	}
	if _, err := Parse([]byte(strings.Replace(base, "plan: one", "plan: one one", 1))); err == nil {
		t.Fatal("duplicate plan accepted")
	}
}

func TestParseIncompleteIsRepresentable(t *testing.T) {
	record, err := Parse([]byte("schema: gate-run.v1\nstarted: 2026-09-04T10:00:00Z\nplan: one two\ntarget: one exit=0\n"))
	if err != nil {
		t.Fatal(err)
	}
	if len(record.Plan) != 2 || len(record.Targets) != 1 {
		t.Fatalf("record = %+v", record)
	}
}

func TestParseRefusesUnsupportedSchema(t *testing.T) {
	if _, err := Parse([]byte("schema: gate-run.v2\nplan: one\n")); err == nil {
		t.Fatal("unsupported witness schema accepted")
	}
}

func TestSummarizeMissingTimeOrderingAndBounds(t *testing.T) {
	repo := t.TempDir()
	runWitnessGit(t, repo, "init", "-b", "main")
	runWitnessGit(t, repo, "config", "user.name", "Achta Test")
	runWitnessGit(t, repo, "config", "user.email", "achta@example.invalid")
	if err := os.WriteFile(filepath.Join(repo, "tracked.txt"), []byte("tracked\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	runWitnessGit(t, repo, "add", "tracked.txt")
	runWitnessGit(t, repo, "commit", "-m", "base")
	head := runWitnessGit(t, repo, "rev-parse", "HEAD")
	record := Record{
		RepoHead:  head[:7],
		Plan:      []string{"alpha", "beta", "gamma"},
		Targets:   map[string]int{"alpha": 0, "beta": 0, "gamma": 0},
		ElapsedMS: map[string]int64{"gamma": 5, "beta": 10, "alpha": 10},
		TreeKind:  "commit",
		TreeValue: head,
		Result:    "green",
	}
	summary, err := Summarize(record, repo, filepath.Join(repo, "GATE-RUN.txt"), "", 2)
	if err != nil {
		t.Fatal(err)
	}
	if summary.Completeness != "incomplete" || summary.WallElapsedMS != nil || summary.RecordedTargetElapsedMS == nil || *summary.RecordedTargetElapsedMS != 25 {
		t.Fatalf("summary timing = %+v", summary)
	}
	if len(summary.SlowTargets) != 2 || summary.SlowTargets[0].Name != "alpha" || summary.SlowTargets[1].Name != "beta" {
		t.Fatalf("slow targets = %+v", summary.SlowTargets)
	}
	for _, limit := range []int{-1, 101} {
		if _, err := Summarize(record, repo, filepath.Join(repo, "GATE-RUN.txt"), "", limit); err == nil {
			t.Fatalf("slowest limit %d accepted", limit)
		}
	}
}

func runWitnessGit(t *testing.T, repo string, args ...string) string {
	t.Helper()
	command := exec.Command("git", append([]string{"-C", repo}, args...)...)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v: %s", args, err, output)
	}
	return strings.TrimSpace(string(output))
}

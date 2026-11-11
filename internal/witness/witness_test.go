package witness

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
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

// RULE: WITNESS-FAILED-NAMES-1
func TestSummarizeFailedTargetNames(t *testing.T) {
	repo := t.TempDir()
	runWitnessGit(t, repo, "init", "-b", "main")
	runWitnessGit(t, repo, "config", "user.name", "Achta Test")
	runWitnessGit(t, repo, "config", "user.email", "achta@example.invalid")
	if err := os.WriteFile(filepath.Join(repo, "tracked.txt"), []byte("tracked\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	runWitnessGit(t, repo, "add", "tracked.txt")
	runWitnessGit(t, repo, "commit", "-m", "base")
	recordPath := filepath.Join(repo, "GATE-RUN.txt")
	for _, tc := range []struct {
		name   string
		record string
		want   []string
		status string
		raw    string
	}{
		{
			name:   "plan order, negative exit, missing not listed",
			record: "schema: gate-run.v1\nplan: gamma alpha beta delta\ntarget: beta exit=-2\ntarget: alpha exit=0\ntarget: gamma exit=1\n",
			want:   []string{"gamma", "beta"},
			status: "red",
			raw:    `"failed_targets":2,"missing_targets":1,"slow_targets":[],"failed_target_names":["gamma","beta"]`,
		},
		{
			name:   "nothing failed",
			record: "schema: gate-run.v1\nplan: gamma alpha\ntarget: alpha exit=0\ntarget: gamma exit=0\n",
			want:   []string{},
			status: "incomplete",
			raw:    `"failed_targets":0,"missing_targets":0,"slow_targets":[],"failed_target_names":[]`,
		},
		{
			name:   "nothing recorded",
			record: "schema: gate-run.v1\nplan: gamma alpha\n",
			want:   []string{},
			status: "incomplete",
			raw:    `"failed_targets":0,"missing_targets":2,"slow_targets":[],"failed_target_names":[]`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			record, err := Parse([]byte(tc.record))
			if err != nil {
				t.Fatal(err)
			}
			summary, err := Summarize(record, repo, recordPath, "", 0)
			if err != nil {
				t.Fatal(err)
			}
			if summary.FailedTargetNames == nil || !slices.Equal(summary.FailedTargetNames, tc.want) {
				t.Fatalf("failed target names = %#v, want %#v", summary.FailedTargetNames, tc.want)
			}
			if summary.FailedTargets != len(summary.FailedTargetNames) {
				t.Fatalf("failed_targets = %d, names = %d", summary.FailedTargets, len(summary.FailedTargetNames))
			}
			if summary.Status != tc.status {
				t.Fatalf("status = %q, want %q", summary.Status, tc.status)
			}
			encoded, err := json.Marshal(summary)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(encoded), tc.raw) {
				t.Fatalf("document = %s, want it to contain %s", encoded, tc.raw)
			}
		})
	}
}

// RULE: WITNESS-MISSING-NAMES-1
func TestSummarizeMissingTargetNames(t *testing.T) {
	repo := t.TempDir()
	runWitnessGit(t, repo, "init", "-b", "main")
	runWitnessGit(t, repo, "config", "user.name", "Achta Test")
	runWitnessGit(t, repo, "config", "user.email", "achta@example.invalid")
	if err := os.WriteFile(filepath.Join(repo, "tracked.txt"), []byte("tracked\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	runWitnessGit(t, repo, "add", "tracked.txt")
	runWitnessGit(t, repo, "commit", "-m", "base")
	recordPath := filepath.Join(repo, "GATE-RUN.txt")
	for _, tc := range []struct {
		name   string
		record string
		want   []string
		raw    string
	}{
		{
			name:   "plan order, failed not listed",
			record: "schema: gate-run.v1\nplan: delta gamma alpha beta\ntarget: gamma exit=1\ntarget: alpha exit=0\n",
			want:   []string{"delta", "beta"},
			raw:    `"failed_targets":1,"missing_targets":2,"slow_targets":[],"failed_target_names":["gamma"],"missing_target_names":["delta","beta"]}`,
		},
		{
			name:   "nothing missing",
			record: "schema: gate-run.v1\nplan: gamma alpha\ntarget: alpha exit=0\ntarget: gamma exit=3\n",
			want:   []string{},
			raw:    `"failed_targets":1,"missing_targets":0,"slow_targets":[],"failed_target_names":["gamma"],"missing_target_names":[]}`,
		},
		{
			name:   "nothing recorded",
			record: "schema: gate-run.v1\nplan: gamma alpha\n",
			want:   []string{"gamma", "alpha"},
			raw:    `"failed_targets":0,"missing_targets":2,"slow_targets":[],"failed_target_names":[],"missing_target_names":["gamma","alpha"]}`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			record, err := Parse([]byte(tc.record))
			if err != nil {
				t.Fatal(err)
			}
			summary, err := Summarize(record, repo, recordPath, "", 0)
			if err != nil {
				t.Fatal(err)
			}
			if summary.MissingTargetNames == nil || !slices.Equal(summary.MissingTargetNames, tc.want) {
				t.Fatalf("missing target names = %#v, want %#v", summary.MissingTargetNames, tc.want)
			}
			if summary.MissingTargets != len(summary.MissingTargetNames) {
				t.Fatalf("missing_targets = %d, names = %d", summary.MissingTargets, len(summary.MissingTargetNames))
			}
			if summary.PassedTargets+summary.FailedTargets+summary.MissingTargets != summary.PlannedTargets {
				t.Fatalf("counts do not sum to the plan: %+v", summary)
			}
			encoded, err := json.Marshal(summary)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.HasSuffix(string(encoded), tc.raw) {
				t.Fatalf("document = %s, want it to end with %s", encoded, tc.raw)
			}
		})
	}
}

func TestParseRefusesRepeatedAndUnplannedTargets(t *testing.T) {
	for name, record := range map[string]string{
		"target recorded twice":     "schema: gate-run.v1\nplan: one two\ntarget: one exit=1\ntarget: one exit=1\n",
		"target not in the plan":    "schema: gate-run.v1\nplan: one\ntarget: two exit=1\n",
		"target planned twice":      "schema: gate-run.v1\nplan: one one\ntarget: one exit=1\n",
		"target name outside limit": "schema: gate-run.v1\nplan: one\ntarget: o,ne exit=1\n",
	} {
		if _, err := Parse([]byte(record)); err == nil {
			t.Fatalf("%s: record accepted", name)
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

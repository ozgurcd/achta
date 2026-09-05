package cli

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

const floorFixtureCensus = "# census\n\n## Totals (sum = 2)\n\n- COVERED: 1\n- OOS: 1\n\n## Full list\n\n```\nCOVERED  1  A  internal/a.go:1  RULE-1\nOOS      1  B  internal/b.go:2  P\n```\n"

func TestFloorCensusCommandReadsCoversAndRefusesArmedState(t *testing.T) {
	root, _, _ := testWorkspaceRepo(t, "sample")
	writeTestFile(t, filepath.Join(root, "wiki", "census.md"), floorFixtureCensus)
	writeTestFile(t, filepath.Join(root, "covers.json"), `{"schema_version":"rulefloor.covers.v1","rules":{"RULE-1":["internal/a.go:A"]}}`+"\n")

	var stdout, stderr bytes.Buffer
	base := []string{"--json", "--workspace", root, "floor", "census", "--file", "wiki/census.md", "--covers", "covers.json", "--bucket", "COVERED,OOS", "--covered", "COVERED", "--fence-heading", "## Full list", "--count-sum", `^## Totals \(sum = (\d+)`}
	if code := Run(base, &stdout, &stderr, "v0.1.0"); code != 0 {
		t.Fatalf("clean census code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), `"schema_version":"achta.floor-census.v1"`) || !strings.Contains(stdout.String(), `"status":"pass"`) || !strings.Contains(stdout.String(), `"refused":["whether a cited rule is ARMED`) {
		t.Fatalf("unexpected floor census JSON: %s", stdout.String())
	}

	// A stated count the table contradicts fails at the command level.
	writeTestFile(t, filepath.Join(root, "wiki", "census.md"), strings.Replace(floorFixtureCensus, "- COVERED: 1", "- COVERED: 4", 1))
	stdout.Reset()
	if code := Run(base, &stdout, &stderr, "v0.1.0"); code != 1 || !strings.Contains(stdout.String(), `"class":6`) {
		t.Fatalf("stated count mismatch must fail: code=%d stdout=%q", code, stdout.String())
	}

	// Vocabulary is the caller's: no --bucket is invalid input, not a default.
	stdout.Reset()
	if code := Run([]string{"--json", "--workspace", root, "floor", "census", "--file", "wiki/census.md", "--covers", "covers.json", "--covered", "COVERED", "--fence-heading", "## Full list"}, &stdout, &stderr, "v0.1.0"); code != 2 {
		t.Fatalf("missing --bucket must be exit 2: code=%d stdout=%q", code, stdout.String())
	}

	// Without --covers the verb executes rulefloor; a missing executable is cannot_evaluate naming it.
	stdout.Reset()
	if code := Run([]string{"--json", "--workspace", root, "floor", "census", "--file", "wiki/census.md", "--repo", "sample", "--rulefloor", filepath.Join(root, "no-such-rulefloor"), "--bucket", "COVERED,OOS", "--covered", "COVERED", "--fence-heading", "## Full list"}, &stdout, &stderr, "v0.1.0"); code != 2 || !strings.Contains(stdout.String(), "no-such-rulefloor") {
		t.Fatalf("missing rulefloor must be exit 2 naming the executable: code=%d stdout=%q", code, stdout.String())
	}
}

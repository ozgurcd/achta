// SPDX-License-Identifier: MIT
package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// RULE: CLAIM-COUNT-RELATIONSHIPS-1
func TestCountCheckFiresBreakdownAndCitationRulesAndPassesCleanFixture(t *testing.T) {
	root, _, _ := testWorkspaceRepo(t, "sample")
	base := []string{
		"--json", "--workspace", root, "count", "check",
		"--total-pattern", `TOTAL[[:space:]]+(?P<count>[0-9]+)[[:space:]]*[|]`,
		"--part-pattern", `(VACUOUS|HAS-CONTROL|SOUND)[[:space:]]+(?P<count>[0-9]+)`,
		"--claim-pattern", `claims[[:space:]]+(?P<count>[0-9]+)[[:space:]]+fixed`,
		"--citation-pattern", `[[:alnum:]_./-]+[.](go|md):[0-9]+`,
	}
	for _, test := range []struct {
		fixture string
		code    int
		rule    string
	}{
		{fixture: "clean.md", code: 0},
		{fixture: "breakdown-fail.md", code: 1, rule: "breakdown-sum"},
		{fixture: "citation-fail.md", code: 1, rule: "claim-citation-count"},
	} {
		t.Run(test.fixture, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join("..", "..", "testdata", "countcheck", test.fixture))
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(root, "wiki", test.fixture)
			writeTestFile(t, path, string(data))
			args := append(append([]string{}, base...), "--file", filepath.ToSlash(filepath.Join("wiki", test.fixture)))
			var stdout, stderr bytes.Buffer
			if code := Run(args, &stdout, &stderr, "v0.5.0"); code != test.code {
				t.Fatalf("code=%d want=%d stdout=%s stderr=%s", code, test.code, stdout.String(), stderr.String())
			}
			if !strings.Contains(stdout.String(), `"schema_version":"achta.count-check.v1"`) {
				t.Fatalf("missing schema: %s", stdout.String())
			}
			if test.rule != "" && !strings.Contains(stdout.String(), `"rule":"`+test.rule+`"`) {
				t.Fatalf("missing rule %q: %s", test.rule, stdout.String())
			}
		})
	}

	cleanData, err := os.ReadFile(filepath.Join("..", "..", "testdata", "countcheck", "clean.md"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "wiki", "log"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, filepath.Join(root, "wiki", "log", "entry.MD"), string(cleanData))
	var directoryStdout, directoryStderr bytes.Buffer
	directoryArgs := append(append([]string{}, base...), "--dir", "wiki/log")
	if code := Run(directoryArgs, &directoryStdout, &directoryStderr, "v0.5.0"); code != 0 || !strings.Contains(directoryStdout.String(), `"files":1`) {
		t.Fatalf("directory code=%d stdout=%s stderr=%s", code, directoryStdout.String(), directoryStderr.String())
	}

	var stdout, stderr bytes.Buffer
	missing := append(append([]string{}, base...), "--file", "wiki/absent.md")
	if code := Run(missing, &stdout, &stderr, "v0.5.0"); code != 2 || !strings.Contains(stdout.String(), `"status":"cannot_evaluate"`) {
		t.Fatalf("absent code=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	incomplete := []string{"--json", "--workspace", root, "count", "check", "--file", "wiki/clean.md", "--total-pattern", `(?P<count>[0-9]+)`}
	if code := Run(incomplete, &stdout, &stderr, "v0.5.0"); code != 2 || !strings.Contains(stdout.String(), `"status":"cannot_evaluate"`) {
		t.Fatalf("incomplete code=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
}

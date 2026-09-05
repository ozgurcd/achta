package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// RULE: LEDGER-ROWS-1
func TestLedgerRowsCommandFiresEachRuleAndPassesCleanFixture(t *testing.T) {
	root, _, _ := testWorkspaceRepo(t, "sample")
	baseArgs := []string{
		"--json", "--workspace", root, "ledger", "rows",
		"--id-cell", "1", "--prose-cell", "2",
		"--open-marker", "**", "--closed-marker", "DONE **", "--identity-end-marker", "**",
		"--completion-marker", "DONE ", "--completion-marker", "**SATISFIED",
		"--exempt-marker", "RULED-OPEN", "--exempt-marker", "STAYS OPEN",
		"--condition-marker", "Close condition.", "--condition-marker", "Close condition:",
		"--quote-marker", "CLOSE-CONDITION-MET:",
	}
	for _, tc := range []struct {
		fixture string
		code    int
		rule    string
	}{
		{fixture: "clean.md", code: 0},
		{fixture: "completion-open.md", code: 1, rule: "open-completion-claim"},
		{fixture: "condition-unquoted.md", code: 1, rule: "close-condition-quote"},
		{fixture: "condition-paraphrase.md", code: 1, rule: "close-condition-quote"},
	} {
		t.Run(tc.fixture, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join("..", "..", "testdata", "ledgerrows", tc.fixture))
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(root, "wiki", tc.fixture)
			writeTestFile(t, path, string(data))
			args := append(append([]string{}, baseArgs...), "--file", filepath.ToSlash(filepath.Join("wiki", tc.fixture)))
			var stdout, stderr bytes.Buffer
			if code := Run(args, &stdout, &stderr, "v0.4.5"); code != tc.code {
				t.Fatalf("code=%d want %d stdout=%q stderr=%q", code, tc.code, stdout.String(), stderr.String())
			}
			if !strings.Contains(stdout.String(), `"schema_version":"achta.ledger-rows.v1"`) {
				t.Fatalf("missing schema: %s", stdout.String())
			}
			if tc.rule != "" && !strings.Contains(stdout.String(), `"rule":"`+tc.rule+`"`) {
				t.Fatalf("missing rule %q: %s", tc.rule, stdout.String())
			}
		})
	}
}

func TestLedgerRowsCommandCannotEvaluateMissingVocabularyOrRows(t *testing.T) {
	root, _, _ := testWorkspaceRepo(t, "sample")
	writeTestFile(t, filepath.Join(root, "wiki", "ledger.md"), "| ID | Item |\n|---|---|\n")
	full := []string{
		"--json", "--workspace", root, "ledger", "rows", "--file", "wiki/ledger.md",
		"--id-cell", "1", "--prose-cell", "2",
		"--open-marker", "**", "--closed-marker", "DONE **", "--identity-end-marker", "**",
		"--completion-marker", "DONE ", "--condition-marker", "Close condition.",
		"--quote-marker", "CLOSE-CONDITION-MET:",
	}
	var stdout, stderr bytes.Buffer
	if code := Run(full, &stdout, &stderr, "v0.4.5"); code != 2 || !strings.Contains(stdout.String(), `"status":"cannot_evaluate"`) {
		t.Fatalf("no rows code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	missingMarkers := []string{"--json", "--workspace", root, "ledger", "rows", "--file", "wiki/ledger.md", "--id-cell", "1", "--prose-cell", "2"}
	if code := Run(missingMarkers, &stdout, &stderr, "v0.4.5"); code != 2 {
		t.Fatalf("missing vocabulary code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
}

// SPDX-License-Identifier: MIT
package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A passing check lists what it deliberately does not infer. Human output must
// not word those statements as refusals, which read like failures when Lictor
// relays them; JSON keeps them in the refused list (TOOL-FAULTS-1103 ACH-03).
func TestPassingChecksWordDesignLimitsAsNotInferred(t *testing.T) {
	root, _, _ := testWorkspaceRepo(t, "sample")
	copyFixture := func(t *testing.T, dir, fixture string) string {
		t.Helper()
		data, err := os.ReadFile(filepath.Join("..", "..", "testdata", dir, fixture))
		if err != nil {
			t.Fatal(err)
		}
		writeTestFile(t, filepath.Join(root, "wiki", dir+"-"+fixture), string(data))
		return filepath.ToSlash(filepath.Join("wiki", dir+"-"+fixture))
	}
	tests := []struct {
		name, prefix string
		args         func(t *testing.T) []string
	}{
		{"declared-route check", "declared-route check", func(t *testing.T) []string {
			return []string{
				"--workspace", filepath.Join("..", ".."), "declared-route", "check",
				"--route-pattern", `rulefloor/archive/refs/tags/\$\{RULEFLOOR_VERSION\}`,
				"--required-route-pattern", `rulefloor/archive/refs/tags/\$\{RULEFLOOR_VERSION\}`,
				"--ban-pattern", `go install[^#]*rulefloor`,
				"--required-key", "RULEFLOOR_VERSION", "--required-scope", "/env",
				"--file", filepath.ToSlash(filepath.Join("testdata", "declaredroute", "clean.yml")),
			}
		}},
		{"count check", "count check", func(t *testing.T) []string {
			return []string{
				"--workspace", root, "count", "check",
				"--total-pattern", `TOTAL[[:space:]]+(?P<count>[0-9]+)[[:space:]]*[|]`,
				"--part-pattern", `(VACUOUS|HAS-CONTROL|SOUND)[[:space:]]+(?P<count>[0-9]+)`,
				"--claim-pattern", `claims[[:space:]]+(?P<count>[0-9]+)[[:space:]]+fixed`,
				"--citation-pattern", `[[:alnum:]_./-]+[.](go|md):[0-9]+`,
				"--file", copyFixture(t, "countcheck", "clean.md"),
			}
		}},
		{"ledger rows", "ledger rows", func(t *testing.T) []string {
			return []string{
				"--workspace", root, "ledger", "rows",
				"--id-cell", "1", "--prose-cell", "2",
				"--open-marker", "**", "--closed-marker", "DONE **", "--identity-end-marker", "**",
				"--completion-marker", "DONE ", "--completion-marker", "**SATISFIED",
				"--exempt-marker", "RULED-OPEN", "--exempt-marker", "STAYS OPEN",
				"--condition-marker", "Close condition.", "--condition-marker", "Close condition:",
				"--quote-marker", "CLOSE-CONDITION-MET:",
				"--file", copyFixture(t, "ledgerrows", "clean.md"),
			}
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			args := test.args(t)
			var stdout, stderr bytes.Buffer
			if code := Run(args, &stdout, &stderr, "v0.5.0"); code != 0 {
				t.Fatalf("code = %d, want a passing run; stdout=%q stderr=%q", code, stdout.String(), stderr.String())
			}
			human := stdout.String()
			if strings.Contains(human, "refused") || !strings.Contains(human, test.prefix+": not inferred (by design) — ") {
				t.Fatalf("passing human output must word design limits as not inferred:\n%s", human)
			}

			stdout.Reset()
			stderr.Reset()
			if code := Run(append([]string{"--json"}, args...), &stdout, &stderr, "v0.5.0"); code != 0 {
				t.Fatalf("JSON code = %d; stderr=%q", code, stderr.String())
			}
			if !strings.Contains(stdout.String(), `"refused":[`) {
				t.Fatalf("JSON must keep the refused list unchanged: %s", stdout.String())
			}
		})
	}
}

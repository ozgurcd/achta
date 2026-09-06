package cli

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

// RULE: DECLARED-ROUTE-YAML-SCOPE-1
func TestDeclaredRouteCheckUsesYAMLScopeAndExitContract(t *testing.T) {
	workspace := filepath.Join("..", "..")
	base := []string{
		"--workspace", workspace,
		"declared-route", "check",
		"--route-pattern", `rulefloor/archive/refs/tags/\$\{RULEFLOOR_VERSION\}`,
		"--route-pattern", `approved-rulefloor-action@`,
		"--required-route-pattern", `rulefloor/archive/refs/tags/\$\{RULEFLOOR_VERSION\}`,
		"--ban-pattern", `brew install[^#]*rulefloor`,
		"--ban-pattern", `go install[^#]*rulefloor`,
		"--ban-pattern", `rulefloor/archive/refs/tags/v[0-9]`,
		"--ban-pattern", `-X main\.version=v[0-9]`,
		"--required-key", "RULEFLOOR_VERSION",
		"--required-scope", "/env",
		"--json",
	}
	tests := []struct {
		file string
		code int
		want string
	}{
		{file: "clean.yml", code: 0, want: `"status":"pass"`},
		{file: "two-routes.yml", code: 1, want: `"rule":"route-count"`},
		{file: "banned.yml", code: 1, want: `"rule":"banned-pattern"`},
		{file: "job-env.yml", code: 1, want: `"rule":"required-key-scope"`},
		{file: "action.yml", code: 0, want: `"required":false`},
	}
	for _, test := range tests {
		t.Run(test.file, func(t *testing.T) {
			args := append(append([]string{}, base...), "--file", filepath.ToSlash(filepath.Join("testdata", "declaredroute", test.file)))
			var stdout, stderr bytes.Buffer
			if code := Run(args, &stdout, &stderr, "v0.5.0"); code != test.code {
				t.Fatalf("code = %d, want %d; stdout=%q stderr=%q", code, test.code, stdout.String(), stderr.String())
			}
			if !strings.Contains(stdout.String(), test.want) {
				t.Fatalf("stdout = %q, want %q", stdout.String(), test.want)
			}
		})
	}
}

func TestDeclaredRouteCheckCannotEvaluateMissingVocabulary(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"--workspace", filepath.Join("..", ".."), "declared-route", "check", "--file", "testdata/declaredroute/clean.yml", "--json"}, &stdout, &stderr, "v0.5.0")
	if code != 2 || !strings.Contains(stdout.String(), `"status":"cannot_evaluate"`) {
		t.Fatalf("code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
}

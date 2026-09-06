package declaredroute

import (
	"os"
	"path/filepath"
	"testing"
)

var testOptions = Options{
	RoutePatterns:        []string{`rulefloor/archive/refs/tags/\$\{RULEFLOOR_VERSION\}`, `approved-rulefloor-action@`},
	BanPatterns:          []string{`brew install[^#]*rulefloor`, `go install[^#]*rulefloor`, `rulefloor/archive/refs/tags/v[0-9]`, `-X main\.version=v[0-9]`},
	RequiredRoutePattern: `rulefloor/archive/refs/tags/\$\{RULEFLOOR_VERSION\}`,
	RequiredKey:          "RULEFLOOR_VERSION",
	RequiredScope:        "/env",
}

func TestCheckUsesYAMLStructureForRoutesBansAndWorkflowScope(t *testing.T) {
	tests := []struct {
		name    string
		fixture string
		status  string
		rule    string
	}{
		{name: "clean", fixture: "clean.yml", status: "pass"},
		{name: "two routes", fixture: "two-routes.yml", status: "fail", rule: RuleRouteCount},
		{name: "banned install", fixture: "banned.yml", status: "fail", rule: RuleBanned},
		{name: "job pin", fixture: "job-env.yml", status: "fail", rule: RuleRequiredKey},
		{name: "route without pin requirement", fixture: "action.yml", status: "pass"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join("..", "..", "testdata", "declaredroute", test.fixture))
			if err != nil {
				t.Fatal(err)
			}
			result, err := Check([]Document{{Path: test.fixture, Data: data}}, testOptions)
			if err != nil {
				t.Fatal(err)
			}
			if result.Status != test.status {
				t.Fatalf("status = %q, want %q: %#v", result.Status, test.status, result)
			}
			if test.rule != "" && !hasRule(result.Violations, test.rule) {
				t.Fatalf("violations = %#v, want rule %q", result.Violations, test.rule)
			}
		})
	}
}

func TestCheckCannotEvaluateUnsafeOrAmbiguousInputs(t *testing.T) {
	valid := []byte("env:\n  RULEFLOOR_VERSION: v1\njobs:\n  check:\n    steps:\n      - run: curl rulefloor/archive/refs/tags/${RULEFLOOR_VERSION}\n")
	tests := []struct {
		name string
		docs []Document
		opts Options
	}{
		{name: "no document", docs: nil, opts: testOptions},
		{name: "malformed yaml", docs: []Document{{Path: "ci.yml", Data: []byte("env: [\n")}}, opts: testOptions},
		{name: "multiple documents", docs: []Document{{Path: "ci.yml", Data: append(valid, []byte("---\nenv: {}\n")...)}}, opts: testOptions},
		{name: "no routes", docs: []Document{{Path: "ci.yml", Data: valid}}, opts: Options{BanPatterns: testOptions.BanPatterns, RequiredRoutePattern: testOptions.RequiredRoutePattern, RequiredKey: testOptions.RequiredKey, RequiredScope: testOptions.RequiredScope}},
		{name: "no bans", docs: []Document{{Path: "ci.yml", Data: valid}}, opts: Options{RoutePatterns: testOptions.RoutePatterns, RequiredRoutePattern: testOptions.RequiredRoutePattern, RequiredKey: testOptions.RequiredKey, RequiredScope: testOptions.RequiredScope}},
		{name: "no required route", docs: []Document{{Path: "ci.yml", Data: valid}}, opts: Options{RoutePatterns: testOptions.RoutePatterns, BanPatterns: testOptions.BanPatterns, RequiredKey: testOptions.RequiredKey, RequiredScope: testOptions.RequiredScope}},
		{name: "bad scope", docs: []Document{{Path: "ci.yml", Data: valid}}, opts: Options{RoutePatterns: testOptions.RoutePatterns, BanPatterns: testOptions.BanPatterns, RequiredRoutePattern: testOptions.RequiredRoutePattern, RequiredKey: testOptions.RequiredKey, RequiredScope: "env"}},
		{name: "duplicate scope", docs: []Document{{Path: "ci.yml", Data: []byte("env: {}\nenv: {}\njobs:\n  check:\n    steps:\n      - run: curl rulefloor/archive/refs/tags/${RULEFLOOR_VERSION}\n")}}, opts: testOptions},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := Check(test.docs, test.opts); err == nil {
				t.Fatal("expected cannot-evaluate error")
			}
		})
	}
}

func hasRule(violations []Violation, rule string) bool {
	for _, violation := range violations {
		if violation.Rule == rule {
			return true
		}
	}
	return false
}

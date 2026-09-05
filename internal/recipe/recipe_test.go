package recipe

import (
	"strings"
	"testing"
)

const fixture = "SHELL := /usr/bin/env bash\n\nhelp:\n\t@echo help\n\ncheck:\n\t# a comment line does nothing and is skipped\n\t@bash tools/gate.sh run RECORD \"label\" \\\n\t\t'a=bash tools/a.sh' \\\n\t\t'b=achta --wiki-dir . wiki check'\n\nderive:\n\t@achta wiki derive\n"

func names(result Result) string {
	var rules []string
	for _, v := range result.Violations {
		rules = append(rules, v.Rule)
	}
	return strings.Join(rules, ",")
}

// RULE: RECIPE-CHECK-1
func TestCheckRefusesNeutralizersAndRequiresByteEqualLines(t *testing.T) {
	clean, err := Check([]byte(fixture), Options{Target: "check", ExpectLines: []string{`@bash tools/gate.sh run RECORD "label" \`}})
	if err != nil || clean.Status != "pass" || clean.Lines != 3 || len(clean.Violations) != 0 || len(clean.MissingExpected) != 0 {
		t.Fatalf("clean recipe: err=%v status=%q lines=%d violations=%v missing=%v", err, clean.Status, clean.Lines, clean.Violations, clean.MissingExpected)
	}

	// EQUALITY, not substring: anything appended to the expected line must fail.
	appended := strings.Replace(fixture, `'b=achta --wiki-dir . wiki check'`, `'b=achta --wiki-dir . wiki check' | cat`, 1)
	got, err := Check([]byte(appended), Options{Target: "check", ExpectLines: []string{`	'b=achta --wiki-dir . wiki check'`}})
	if err != nil || got.Status != "fail" || len(got.MissingExpected) != 1 || !strings.Contains(names(got), "pipe") {
		t.Fatalf("appended pipe: err=%v status=%q rules=%q missing=%v", err, got.Status, names(got), got.MissingExpected)
	}

	cases := []struct {
		name, from, to, rule string
	}{
		{"dash prefix", "\t@bash tools/gate.sh", "\t-@bash tools/gate.sh", "ignored-exit"},
		{"dash after @", "\t@bash tools/gate.sh", "\t@-bash tools/gate.sh", "ignored-exit"},
		{"pipe", `'b=achta --wiki-dir . wiki check'`, `'b=achta --wiki-dir . wiki check' | cat`, "pipe"},
		{"background", `'b=achta --wiki-dir . wiki check'`, `'b=achta --wiki-dir . wiki check' &`, "background"},
		{"or-true", `'b=achta --wiki-dir . wiki check'`, `'b=achta --wiki-dir . wiki check' || true`, "swallowed-exit"},
		{"or-colon", `'b=achta --wiki-dir . wiki check'`, `'b=achta --wiki-dir . wiki check' || :`, "swallowed-exit"},
		{"semicolon true", `'b=achta --wiki-dir . wiki check'`, `'b=achta --wiki-dir . wiki check'; true`, "swallowed-exit"},
		{"exit 0", `'b=achta --wiki-dir . wiki check'`, `'b=achta --wiki-dir . wiki check' || exit 0`, "swallowed-exit"},
	}
	for _, c := range cases {
		mutated := strings.Replace(fixture, c.from, c.to, 1)
		got, err := Check([]byte(mutated), Options{Target: "check"})
		if err != nil || got.Status != "fail" || !strings.Contains(names(got), c.rule) {
			t.Fatalf("%s: err=%v status=%q rules=%q", c.name, err, got.Status, names(got))
		}
	}

	// A `||` is not a pipe.
	orLine := strings.Replace(fixture, `'b=achta --wiki-dir . wiki check'`, `'b=test -f x || achta --wiki-dir . wiki check'`, 1)
	if got, err := Check([]byte(orLine), Options{Target: "check"}); err != nil || got.Status != "pass" {
		t.Fatalf("|| treated as a pipe: err=%v status=%q rules=%q", err, got.Status, names(got))
	}

	// --forbid-noop refuses commands that cannot fail; without it they pass the static rules.
	noop := strings.Replace(fixture, "\t@bash tools/gate.sh run RECORD \"label\" \\\n", "\t@true\n\t@echo gate\n\t@printf ok\n\t:\n", 1)
	got, err = Check([]byte(noop), Options{Target: "check", ForbidNoop: true})
	if err != nil || got.Status != "fail" || strings.Count(names(got), "noop") != 4 {
		t.Fatalf("forbid-noop: err=%v status=%q rules=%q", err, got.Status, names(got))
	}
	if got, err := Check([]byte(noop), Options{Target: "check"}); err != nil || got.Status != "pass" {
		t.Fatalf("noop without --forbid-noop should pass the static rules: err=%v status=%q", err, got.Status)
	}
}

func TestCheckCannotEvaluate(t *testing.T) {
	if _, err := Check([]byte(fixture), Options{Target: "nosuch"}); err == nil {
		t.Fatal("missing target must be an error, not a pass")
	}
	if _, err := Check([]byte("empty:\n\nnext:\n\t@true\n"), Options{Target: "empty"}); err == nil {
		t.Fatal("a target with no recipe lines must be an error")
	}
	if _, err := Check([]byte(fixture), Options{Target: "bad name"}); err == nil {
		t.Fatal("an invalid target name must be an error")
	}
	if _, err := Check([]byte(fixture+"\ncheck:\n\t@true\n"), Options{Target: "check"}); err == nil {
		t.Fatal("a target defined twice must be an error")
	}
}

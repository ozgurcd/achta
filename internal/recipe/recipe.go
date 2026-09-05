// Package recipe reads one Makefile target's recipe as TEXT and refuses
// neutralizers — edits that keep a gate looking present while make ignores its
// exit. Measured cause (identuum, 2026-09-05): macOS make 3.81 silently ignores
// `.SHELLFLAGS := -o pipefail -c`, so a static rule is the only guard, and a
// substring match on the expected command let anything appended pass.
package recipe

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// Schema is the stable machine interface for recipe checks.
const Schema = "achta.recipe-check.v1"

// MaxMakefile bounds the makefile size this package will read.
const MaxMakefile = 4 << 20

// Options select the target and the expectations.
type Options struct {
	Target      string
	ExpectLines []string
	ForbidNoop  bool
}

// Violation names one refused recipe line.
type Violation struct {
	Line int    `json:"line"`
	Rule string `json:"rule"`
	Text string `json:"text"`
}

// Result is the achta.recipe-check.v1 document.
type Result struct {
	SchemaVersion   string      `json:"schema_version"`
	Status          string      `json:"status"`
	Target          string      `json:"target"`
	Lines           int         `json:"lines"`
	Violations      []Violation `json:"violations"`
	MissingExpected []string    `json:"missing_expected"`
}

var (
	dashPrefix   = regexp.MustCompile(`^[[:space:]]*[@+]*-`)
	pipe         = regexp.MustCompile(`(^|[^|])\|([^|]|$)`)
	background   = regexp.MustCompile(`[^&]&[[:space:]]*$`)
	swallowed    = regexp.MustCompile(`(\|\||;)[[:space:]]*(true|:|exit 0)([[:space:]]|;|$)`)
	commandStart = regexp.MustCompile(`^[[:space:]]*[@+-]*[[:space:]]*`)
	noopWords    = map[string]bool{"true": true, ":": true, "echo": true, "printf": true}
	targetName   = regexp.MustCompile(`^[A-Za-z0-9_./-]+$`)
)

// Check evaluates the named target's recipe inside data. An error is a
// cannot-evaluate condition (bad target name, target absent, no recipe lines);
// violations and missing expectations are a normal fail.
func Check(data []byte, opts Options) (Result, error) {
	result := Result{SchemaVersion: Schema, Status: "pass", Target: opts.Target, Violations: []Violation{}, MissingExpected: []string{}}
	if !targetName.MatchString(opts.Target) {
		return result, errors.New("invalid target name")
	}
	if len(data) > MaxMakefile {
		return result, errors.New("makefile exceeds the size limit")
	}
	lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
	start := -1
	prefix := opts.Target + ":"
	for i, l := range lines {
		if strings.HasPrefix(l, prefix) && !strings.HasPrefix(l, opts.Target+":=") {
			if start >= 0 {
				return result, fmt.Errorf("target %q is defined more than once", opts.Target)
			}
			start = i
		}
	}
	if start < 0 {
		return result, fmt.Errorf("target %q not found", opts.Target)
	}
	type physical struct {
		number int
		text   string // without the leading tab
	}
	var recipe []physical
	for i := start + 1; i < len(lines); i++ {
		l := lines[i]
		if l == "" {
			continue
		}
		if !strings.HasPrefix(l, "\t") {
			break
		}
		text := l[1:]
		if strings.HasPrefix(strings.TrimSpace(text), "#") {
			continue // a shell comment line does nothing; equality on --expect-line still requires the real line
		}
		recipe = append(recipe, physical{number: i + 1, text: text})
	}
	if len(recipe) == 0 {
		return result, fmt.Errorf("target %q has no recipe lines", opts.Target)
	}
	result.Lines = len(recipe)
	for _, p := range recipe {
		if dashPrefix.MatchString(p.text) {
			result.Violations = append(result.Violations, Violation{Line: p.number, Rule: "ignored-exit", Text: p.text})
		}
		if pipe.MatchString(p.text) {
			result.Violations = append(result.Violations, Violation{Line: p.number, Rule: "pipe", Text: p.text})
		}
		if background.MatchString(p.text) {
			result.Violations = append(result.Violations, Violation{Line: p.number, Rule: "background", Text: p.text})
		}
		if swallowed.MatchString(p.text) {
			result.Violations = append(result.Violations, Violation{Line: p.number, Rule: "swallowed-exit", Text: p.text})
		}
		if opts.ForbidNoop {
			command := commandStart.ReplaceAllString(p.text, "")
			word := strings.Fields(command)
			if len(word) > 0 && noopWords[word[0]] {
				result.Violations = append(result.Violations, Violation{Line: p.number, Rule: "noop", Text: p.text})
			}
		}
	}
	for _, want := range opts.ExpectLines {
		found := false
		for _, p := range recipe {
			if p.text == want { // EQUALITY, byte for byte: appended text must fail
				found = true
				break
			}
		}
		if !found {
			result.MissingExpected = append(result.MissingExpected, want)
		}
	}
	if len(result.Violations) > 0 || len(result.MissingExpected) > 0 {
		result.Status = "fail"
	}
	return result, nil
}

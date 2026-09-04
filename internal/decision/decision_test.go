package decision

import (
	"bytes"
	"strings"
	"testing"
)

// RULE: DECISION-INSERTION-1
func TestAddUsesMaximumIDAndSectionBoundary(t *testing.T) {
	source := []byte("---\ntitle: Decisions\n---\n\n## Platform\n\n### P-002 — second\nbody\n\n### P-001 — first\nbody\n\n## UI\n\n### U-001 — ui\nbody\n")
	result, err := Add(source, "P", "measured choice (2026-09-04)", []byte("The owner chose this.\n"))
	if err != nil {
		t.Fatal(err)
	}
	if result.ID != "P-003" {
		t.Fatalf("ID = %q", result.ID)
	}
	want := "### P-003 — measured choice (2026-09-04)\nThe owner chose this.\n\n## UI"
	if !strings.Contains(string(result.Data), want) {
		t.Fatalf("result missing insertion before boundary:\n%s", result.Data)
	}
	if !bytes.Equal(source[:bytes.Index(source, []byte("## Platform"))], result.Data[:bytes.Index(result.Data, []byte("## Platform"))]) {
		t.Fatal("bytes before owned section changed")
	}
	malformed := []byte("## Platform\n### P-nope — malformed\nbody\n### P-001 — valid\nbody\n")
	if _, err := Add(malformed, "P", "next", []byte("body\n")); err == nil {
		t.Fatal("malformed decision heading was ignored")
	}
}

func TestAddPreservesCRLF(t *testing.T) {
	source := []byte("## Platform\r\n\r\n### P-001 — first\r\nbody\r\n\r\n## UI\r\n")
	result, err := Add(source, "P", "second", []byte("body\n"))
	if err != nil {
		t.Fatal(err)
	}
	withoutCRLF := bytes.ReplaceAll(result.Data, []byte("\r\n"), nil)
	if bytes.Contains(withoutCRLF, []byte("\n")) {
		t.Fatal("rendered mixed line endings")
	}
}

func TestAddPreservesFinalNewlineBehavior(t *testing.T) {
	for _, source := range []string{
		"## Platform\n### P-001 — first\nbody",
		"## Platform\n### P-001 — first\nbody\n",
		"## Platform\r\n### P-001 — first\r\nbody",
		"## Platform\r\n### P-001 — first\r\nbody\r\n",
	} {
		result, err := Add([]byte(source), "P", "second", []byte("body\n"))
		if err != nil {
			t.Fatal(err)
		}
		wantFinal := strings.HasSuffix(source, "\n")
		if gotFinal := bytes.HasSuffix(result.Data, []byte("\n")); gotFinal != wantFinal {
			t.Fatalf("final newline = %t, want %t for %q", gotFinal, wantFinal, source)
		}
	}
}

func TestAddRefusesAmbiguityAndInvalidInput(t *testing.T) {
	tests := []struct {
		name   string
		source string
		prefix string
		title  string
		body   string
	}{
		{"missing prefix", "## Platform\n", "P", "title", "body"},
		{"duplicate", "## Platform\n### P-001 — a\nx\n### P-001 — b\ny\n", "P", "title", "body"},
		{"duplicate normalized ID", "## Platform\n### P-001 — a\nx\n### P-0001 — b\ny\n", "P", "title", "body"},
		{"malformed ID", "## Platform\n### P-nope — a\nx\n", "P", "title", "body"},
		{"malformed heading", "## Platform\n### P-001 missing separator\nx\n", "P", "title", "body"},
		{"malformed other prefix", "## Platform\n### U-nope — a\nx\n### P-001 — p\ny\n", "P", "title", "body"},
		{"heading outside section", "### P-001 — a\nx\n", "P", "title", "body"},
		{"multiple sections", "## A\n### P-001 — a\nx\n## B\n### P-002 — b\ny\n", "P", "title", "body"},
		{"bad prefix", "## A\n### P-001 — a\nx\n", "p", "title", "body"},
		{"bad title", "## A\n### P-001 — a\nx\n", "P", "bad\ntitle", "body"},
		{"empty body", "## A\n### P-001 — a\nx\n", "P", "title", ""},
		{"oversized body", "## A\n### P-001 — a\nx\n", "P", "title", strings.Repeat("x", MaxBody+1)},
		{"body carriage return", "## A\n### P-001 — a\nx\n", "P", "title", "bad\rbody"},
		{"body nul", "## A\n### P-001 — a\nx\n", "P", "title", "bad\x00body"},
		{"whitespace body", "## A\n### P-001 — a\nx\n", "P", "title", " \t\n"},
		{"body heading", "## A\n### P-001 — a\nx\n", "P", "title", "body\n### U-001 — hidden"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := Add([]byte(tc.source), tc.prefix, tc.title, []byte(tc.body)); err == nil {
				t.Fatal("expected refusal")
			}
		})
	}
	base := "## A\n### P-001 — a\n"
	nearLimit := base + strings.Repeat("x", MaxRegister-len(base))
	if _, err := Add([]byte(nearLimit), "P", "title", []byte("body")); err == nil {
		t.Fatal("rendered register exceeding the size limit was accepted")
	}
}

func FuzzAddDecisionHeading(f *testing.F) {
	f.Add([]byte("## Platform\n### P-001 — first\nbody\n"), "P", "next", []byte("body\n"))
	f.Add([]byte("## A\n### P-001 — first\nbody\n## B\n### P-002 — second\nbody\n"), "P", "next", []byte("body\n"))
	f.Fuzz(func(t *testing.T, register []byte, prefix, title string, body []byte) {
		result, err := Add(register, prefix, title, body)
		if err != nil {
			return
		}
		if result.ID == "" || bytes.Equal(result.Data, register) {
			t.Fatal("successful decision insertion returned an incomplete result")
		}
	})
}

package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ozgurcd/achta/internal/decision"
)

// RULE: DECISION-STDIN-1
func TestDecisionStdin(t *testing.T) {
	const register = "# Decisions\n\n## Platform\n\n### P-001 — existing platform choice\nOld platform body.\n\n## Identity\n\n### D-034 — existing identity choice\nOld identity body.\n"
	for _, tc := range []struct {
		name, body, prefix string
		check              bool
		code               int
	}{
		{"identity", "New body.\n", "D", false, 0},
		{"platform", "New body.\n", "P", false, 0},
		{"check", "New body.\n", "D", true, 1},
		{"empty", "", "D", false, 2},
		{"whitespace", " \n\t", "D", false, 2},
		{"nul", "bad\x00body", "D", false, 2},
		{"carriage-return", "bad\rbody", "D", false, 2},
		{"section", "## Injected\n", "D", false, 2},
		{"decision-heading", "### D-999 — injected\n", "D", false, 2},
		{"maximum", strings.Repeat("a", decision.MaxBody), "D", false, 0},
		{"oversize", strings.Repeat("a", decision.MaxBody+1), "D", false, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root, _, _ := testWorkspaceRepo(t, "sample")
			path := filepath.Join(root, "wiki/platform/decisions.md")
			writeTestFile(t, path, register)
			args := []string{"--workspace", root, "decision", "add", "--title", "new choice", "--prefix", tc.prefix, "--body-file", "-", "--json"}
			if tc.check {
				args = append(args, "--check")
			}
			var out, errOut bytes.Buffer
			code := runWithInput(args, strings.NewReader(tc.body), &out, &errOut, testVersion)
			if code != tc.code {
				t.Fatalf("exit = %d, want %d; output = %s; stderr = %s", code, tc.code, &out, &errOut)
			}
			var doc map[string]any
			decoder := json.NewDecoder(&out)
			if err := decoder.Decode(&doc); err != nil {
				t.Fatal(err)
			}
			if err := decoder.Decode(new(any)); err != io.EOF || errOut.Len() != 0 {
				t.Fatal("expected one JSON document and no stderr")
			}
			got := readTestFile(t, path)
			if tc.code != 0 {
				if got != register {
					t.Fatal("check or refused input changed register")
				}
			} else {
				want, err := decision.Add([]byte(register), tc.prefix, "new choice", []byte(tc.body))
				if err != nil || got != string(want.Data) || doc["decision_id"] != want.ID {
					t.Fatal("stdin differs from existing decision validation and insertion")
				}
			}
		})
	}
}

type decisionCountingReader struct{ read int }

func (r *decisionCountingReader) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = 'a'
	}
	r.read += len(p)
	return len(p), nil
}

type decisionBrokenReader struct{}

func (decisionBrokenReader) Read([]byte) (int, error) {
	return 0, errors.New("fixture read failure")
}

func TestDecisionStdinBoundAndConfinement(t *testing.T) {
	root, _, _ := testWorkspaceRepo(t, "sample")
	path := filepath.Join(root, "wiki/platform/decisions.md")
	const register = "# Decisions\n\n## Platform\n\n### P-001 — existing\nBody.\n"
	writeTestFile(t, path, register)
	outside := filepath.Join(t.TempDir(), "body.md")
	writeTestFile(t, outside, "Body.\n")
	counting := &decisionCountingReader{}
	for _, tc := range []struct {
		name, bodyFile, message string
		input                   io.Reader
	}{
		{"bounded", "-", "1-65536 bytes", counting},
		{"read-error", "-", "read decision body from stdin", decisionBrokenReader{}},
		{"outside", outside, "use --body-file -", strings.NewReader("Body.\n")},
		{"secret-path", "password.md", "secret-like", strings.NewReader("Body.\n")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var out, errOut bytes.Buffer
			code := runWithInput([]string{"--workspace", root, "decision", "add", "--title", "new", "--body-file", tc.bodyFile, "--json"}, tc.input, &out, &errOut, testVersion)
			if code != 2 || !strings.Contains(out.String(), tc.message) {
				t.Fatalf("exit = %d; expected %q in %s", code, tc.message, &out)
			}
			if readTestFile(t, path) != register {
				t.Fatal("refused input changed register")
			}
		})
	}
	if counting.read != decision.MaxBody+1 {
		t.Fatalf("read %d bytes, want bounded sentinel %d", counting.read, decision.MaxBody+1)
	}
}

func TestDecisionStdinHelpAndCapabilities(t *testing.T) {
	var out, errOut bytes.Buffer
	if code := Run([]string{"decision", "add", "--help"}, &out, &errOut, testVersion); code != 0 {
		t.Fatalf("help exit = %d", code)
	}
	for _, text := range []string{"--body-file -", "stdin", "--prefix", "ID series", "Identity", "Platform", "--prefix D", "--prefix P"} {
		if !strings.Contains(out.String(), text) {
			t.Errorf("help missing %q", text)
		}
	}
	if strings.Contains(out.String(), "explicit platform decision") {
		t.Error("help still claims platform-only decisions")
	}
	out.Reset()
	if code := Run([]string{"capabilities", "--json"}, &out, &errOut, testVersion); code != 0 {
		t.Fatalf("capabilities exit = %d", code)
	}
	var doc struct {
		Commands []struct {
			Name       string   `json:"name"`
			InputForms []string `json:"input_forms"`
		} `json:"commands"`
	}
	if err := json.Unmarshal(out.Bytes(), &doc); err != nil {
		t.Fatal(err)
	}
	for _, command := range doc.Commands {
		if command.Name == "decision add" {
			for _, form := range command.InputForms {
				if form == "--body-file -" {
					return
				}
			}
		}
	}
	t.Fatal("decision add capabilities omit stdin form")
}

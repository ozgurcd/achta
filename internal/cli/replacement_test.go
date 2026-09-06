package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReplacementCheckV051ClaimsFail(t *testing.T) {
	root, _, _ := testWorkspaceRepo(t, "sample")
	fixture, err := os.ReadFile(filepath.Join("..", "..", "testdata", "replacement", "v0.5.1-uncited.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "claims.json"), fixture, 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"--workspace", root, "replacement", "check", "--file", "claims.json", "--claim-status", "replaces", "--claim-status", "retired"}, &stdout, &stderr, "v0.5.3")
	if code != 1 {
		t.Fatalf("code = %d, want 1; stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	for _, verb := range []string{"count check", "declared-route check", "mirror check --digest"} {
		if !strings.Contains(stdout.String(), `verb="`+verb+`"`) {
			t.Fatalf("stdout missing %q: %s", verb, stdout.String())
		}
	}
}

// RULE: REPLACEMENT-REPLAY-EVIDENCE-1
func TestReplacementCheckRejectsContradictedCitedReplay(t *testing.T) {
	root, _, _ := testWorkspaceRepo(t, "sample")
	manifest, err := os.ReadFile(filepath.Join("..", "..", "testdata", "replacement", "v0.5.4-contradicted-citations.json"))
	if err != nil {
		t.Fatal(err)
	}
	replay, err := os.ReadFile(filepath.Join("..", "..", "evidence", "replacement-replay-2026-09-06.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "claims.json"), manifest, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "evidence"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "evidence", "replacement-replay-2026-09-06.json"), replay, 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"--workspace", root, "replacement", "check", "--file", "claims.json", "--claim-status", "replaces", "--claim-status", "retired"}, &stdout, &stderr, "v0.5.5")
	if code != 1 {
		t.Fatalf("code = %d, want 1; stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	want := []string{
		"fixture=g replay-row-mismatch — manifest exits 1/1 contradict cited replay exits 1/0",
		"fixture=02 replay-row-mismatch — manifest exits 0/0 contradict cited replay exits 0/2",
		"fixture=03 replay-row-mismatch — manifest exits 0/0 contradict cited replay exits 0/2",
		"fixture=04 replay-row-mismatch — manifest exits 0/0 contradict cited replay exits 0/2",
		"fixture=07 replay-row-mismatch — manifest exits 1/1 contradict cited replay exits 1/0",
		"fixture=10 replay-row-mismatch — manifest exits 0/0 contradict cited replay exits 0/2",
		"replacement check: fail; 2 claim(s), 6 violation(s)",
	}
	for _, fragment := range want {
		if !strings.Contains(stdout.String(), fragment) {
			t.Fatalf("stdout missing %q:\n%s", fragment, stdout.String())
		}
	}

	tampered := append(append([]byte(nil), replay...), '\n')
	if err := os.WriteFile(filepath.Join(root, "evidence", "replacement-replay-2026-09-06.json"), tampered, 0o600); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	code = Run([]string{"--workspace", root, "replacement", "check", "--file", "claims.json", "--claim-status", "replaces"}, &stdout, &stderr, "v0.5.5")
	if code != 1 || !strings.Contains(stdout.String(), "dual-replay-digest") {
		t.Fatalf("tampered evidence code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
}

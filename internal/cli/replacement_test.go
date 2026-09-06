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

package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// RULE: MIRROR-BYTE-IDENTITY-1
func TestMirrorCheckSupportsWorkspaceRootMasterAndExitContract(t *testing.T) {
	root := t.TempDir()
	wiki := filepath.Join(root, "wiki")
	if err := os.MkdirAll(filepath.Join(wiki, "repos"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(wiki, "platform"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(wiki, "contracts"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(wiki, "platform", "decisions.md"), []byte("# Decisions\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	master := filepath.Join(root, "AGENTS.md")
	mirrorPath := filepath.Join(wiki, "contracts", "AGENTS.md")
	if err := os.WriteFile(master, []byte("master\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(mirrorPath, []byte("master\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	args := []string{"--wiki-dir", wiki, "mirror", "check", "--master", "AGENTS.md", "--mirror", "wiki/contracts/AGENTS.md", "--json"}
	var stdout, stderr bytes.Buffer
	if code := Run(args, &stdout, &stderr, "v0.4.5"); code != 0 {
		t.Fatalf("identical code=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), `"status":"match"`) {
		t.Fatalf("identical output = %s", stdout.String())
	}

	if err := os.WriteFile(mirrorPath, []byte("mastfr\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	if code := Run(args, &stdout, &stderr, "v0.4.5"); code != 1 {
		t.Fatalf("changed code=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), `"status":"differs"`) || !strings.Contains(stdout.String(), `"master_sha256"`) || !strings.Contains(stdout.String(), `"mirror_sha256"`) {
		t.Fatalf("changed output = %s", stdout.String())
	}

	if err := os.Remove(master); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	if code := Run(args, &stdout, &stderr, "v0.4.5"); code != 2 {
		t.Fatalf("absent master code=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), `"status":"cannot_evaluate"`) || stderr.Len() != 0 {
		t.Fatalf("absent master stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
}

func TestMirrorCheckReportsAbsentMirrorAsMismatch(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "wiki", "repos"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "wiki", "platform"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "wiki", "platform", "decisions.md"), []byte("# Decisions\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "master.md"), []byte("master\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	args := []string{"--workspace", root, "mirror", "check", "--master", "master.md", "--mirror", "absent.md", "--json"}
	if code := Run(args, &stdout, &stderr, "v0.4.5"); code != 1 {
		t.Fatalf("code=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), `"status":"absent"`) {
		t.Fatalf("output = %s", stdout.String())
	}
}

package cli

import (
	"bytes"
	"crypto/sha256"
	"fmt"
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

// RULE: MIRROR-RECORDED-DIGEST-1
func TestMirrorCheckDigestExitContract(t *testing.T) {
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
	masterPath := filepath.Join(root, "master.md")
	digestPath := filepath.Join(root, "master.sha256")
	master := []byte("master\n")
	if err := os.WriteFile(masterPath, master, 0o644); err != nil {
		t.Fatal(err)
	}
	masterSHA := fmt.Sprintf("%x", sha256.Sum256(master))
	if err := os.WriteFile(digestPath, []byte(masterSHA+"  master.md\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	args := []string{"--workspace", root, "mirror", "check", "--master", "master.md", "--digest", "master.sha256", "--json"}
	var stdout, stderr bytes.Buffer
	if code := Run(args, &stdout, &stderr, "v0.5.0"); code != 0 {
		t.Fatalf("matching code=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), `"file":"master.sha256"`) || !strings.Contains(stdout.String(), `"line":1`) || !strings.Contains(stdout.String(), `"name":"master.md"`) || !strings.Contains(stdout.String(), `"recorded_sha256":"`+masterSHA+`"`) || !strings.Contains(stdout.String(), `"master_sha256":"`+masterSHA+`"`) {
		t.Fatalf("matching output = %s", stdout.String())
	}

	changed := []byte("mastfr\n")
	changedSHA := fmt.Sprintf("%x", sha256.Sum256(changed))
	if err := os.WriteFile(masterPath, changed, 0o644); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	if code := Run(args, &stdout, &stderr, "v0.5.0"); code != 1 {
		t.Fatalf("changed code=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), masterSHA) || !strings.Contains(stdout.String(), changedSHA) || !strings.Contains(stdout.String(), `"status":"differs"`) {
		t.Fatalf("changed output does not name both digests: %s", stdout.String())
	}

	if err := os.WriteFile(masterPath, master, 0o644); err != nil {
		t.Fatal(err)
	}
	tamperedSHA := strings.Repeat("0", 64)
	if err := os.WriteFile(digestPath, []byte(tamperedSHA+"  master.md\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	if code := Run(args, &stdout, &stderr, "v0.5.0"); code != 1 {
		t.Fatalf("tampered code=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), tamperedSHA) || !strings.Contains(stdout.String(), masterSHA) {
		t.Fatalf("tampered output does not name both digests: %s", stdout.String())
	}

	if err := os.Remove(digestPath); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	if code := Run(args, &stdout, &stderr, "v0.5.0"); code != 2 {
		t.Fatalf("absent code=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), `"status":"cannot_evaluate"`) || stderr.Len() != 0 {
		t.Fatalf("absent stdout=%s stderr=%s", stdout.String(), stderr.String())
	}

	for _, test := range []struct {
		name string
		data string
	}{
		{name: "different basename", data: masterSHA + "  other.md\n"},
		{name: "duplicate basename", data: masterSHA + "  master.md\n" + masterSHA + "  master.md\n"},
		{name: "non-canonical spacing", data: masterSHA + " master.md\n"},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := os.WriteFile(digestPath, []byte(test.data), 0o644); err != nil {
				t.Fatal(err)
			}
			stdout.Reset()
			stderr.Reset()
			if code := Run(args, &stdout, &stderr, "v0.5.0"); code != 2 {
				t.Fatalf("code=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
			}
			if !strings.Contains(stdout.String(), `"status":"cannot_evaluate"`) || stderr.Len() != 0 {
				t.Fatalf("stdout=%s stderr=%s", stdout.String(), stderr.String())
			}
		})
	}
}

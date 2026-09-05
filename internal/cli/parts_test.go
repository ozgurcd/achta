package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// RULE: PARTS-LOCK-1
func TestPartsLockAndVerifyExitContract(t *testing.T) {
	root := partsWorkspace(t)
	partDir := filepath.Join(root, "prompt", "parts")
	if err := os.MkdirAll(partDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writePart := func(name, body string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(partDir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	writePart("alpha.txt", "alpha\n")
	writePart("beta.txt", "beta\n")
	lockArgs := []string{"--workspace", root, "parts", "lock", "--dir", "prompt/parts", "--lock", "prompt/parts.lock", "--json"}
	verifyArgs := []string{"--workspace", root, "parts", "verify", "--dir", "prompt/parts", "--lock", "prompt/parts.lock", "--json"}

	var stdout, stderr bytes.Buffer
	if code := Run(lockArgs, &stdout, &stderr, testVersion); code != 0 {
		t.Fatalf("lock code=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), `"version":"v1"`) || !strings.Contains(stdout.String(), `"status":"updated"`) {
		t.Fatalf("lock output = %s", stdout.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := Run(verifyArgs, &stdout, &stderr, testVersion); code != 0 {
		t.Fatalf("verify code=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), `"status":"pass"`) {
		t.Fatalf("verify output = %s", stdout.String())
	}

	writePart("beta.txt", "betA\n")
	stdout.Reset()
	stderr.Reset()
	if code := Run(verifyArgs, &stdout, &stderr, testVersion); code != 1 {
		t.Fatalf("edited code=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), `"name":"beta.txt","status":"differs"`) {
		t.Fatalf("edited output = %s", stdout.String())
	}

	writePart("beta.txt", "beta\n")
	writePart("extra.txt", "extra\n")
	stdout.Reset()
	stderr.Reset()
	if code := Run(verifyArgs, &stdout, &stderr, testVersion); code != 1 {
		t.Fatalf("unlocked code=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), `"name":"extra.txt","status":"unlocked"`) {
		t.Fatalf("unlocked output = %s", stdout.String())
	}
}

func TestPartsLockBumpAndCannotEvaluate(t *testing.T) {
	root := partsWorkspace(t)
	partDir := filepath.Join(root, "parts")
	if err := os.MkdirAll(partDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(partDir, "one.txt"), []byte("one\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	args := []string{"--workspace", root, "parts", "lock", "--dir", "parts", "--lock", "parts.lock", "--bump", "--json"}
	var stdout, stderr bytes.Buffer
	if code := Run(args, &stdout, &stderr, testVersion); code != 2 {
		t.Fatalf("bump absent code=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), `"status":"cannot_evaluate"`) {
		t.Fatalf("bump absent output = %s", stdout.String())
	}

	args = []string{"--workspace", root, "parts", "lock", "--dir", "parts", "--lock", "parts.lock", "--json"}
	stdout.Reset()
	stderr.Reset()
	if code := Run(args, &stdout, &stderr, testVersion); code != 0 {
		t.Fatalf("initial code=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	args = append(args[:len(args)-1], "--bump", "--json")
	stdout.Reset()
	stderr.Reset()
	if code := Run(args, &stdout, &stderr, testVersion); code != 0 {
		t.Fatalf("bump code=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), `"version":"v2"`) {
		t.Fatalf("bump output = %s", stdout.String())
	}
}

func partsWorkspace(t *testing.T) string {
	t.Helper()
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
	return root
}

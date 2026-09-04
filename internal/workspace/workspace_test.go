package workspace

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveAndConfine(t *testing.T) {
	root := fixtureWorkspace(t)
	nested := filepath.Join(root, "repo", "nested")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := Resolve(root, nested)
	if err != nil {
		t.Fatal(err)
	}
	if got.Root != root {
		t.Fatalf("root = %q, want %q", got.Root, root)
	}
	if _, err := got.Confine("../escape"); err == nil {
		t.Fatal("Confine accepted traversal")
	}
}

// RULE: WORKSPACE-AMBIGUITY-1
func TestResolveRefusesAmbiguousAncestors(t *testing.T) {
	root := fixtureWorkspace(t)
	nested := filepath.Join(root, "nested")
	if err := os.MkdirAll(filepath.Join(nested, "wiki", "repos"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(nested, "wiki", "platform"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nested, "wiki", "platform", "decisions.md"), []byte("# Decisions\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Resolve("", nested); err == nil {
		t.Fatal("Resolve accepted nested candidate when its outer workspace is also a candidate")
	}
}

func TestResolveExplicitPrecedence(t *testing.T) {
	root := fixtureWorkspace(t)
	other := fixtureWorkspace(t)
	got, err := Resolve(root, filepath.Join(other, "wiki"))
	if err != nil {
		t.Fatal(err)
	}
	if got.Root != root {
		t.Fatalf("root = %q, want explicit %q", got.Root, root)
	}
}

func fixtureWorkspace(t *testing.T) string {
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
	real, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	return real
}

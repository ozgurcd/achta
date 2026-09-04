package safefile

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadReplacePreservesMode(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "page.md")
	if err := os.WriteFile(path, []byte("before\n"), 0o640); err != nil {
		t.Fatal(err)
	}
	snapshot, err := Read(root, path, 1024)
	if err != nil {
		t.Fatal(err)
	}
	if err := snapshot.Replace([]byte("after\n"), 1024); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "after\n" {
		t.Fatalf("content = %q", got)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o640 {
		t.Fatalf("mode = %o", info.Mode().Perm())
	}
}

func TestReadRejectsSymlinkAndSecretLike(t *testing.T) {
	root := t.TempDir()
	real := filepath.Join(root, "real.md")
	if err := os.WriteFile(real, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(real, filepath.Join(root, "linked.md")); err != nil {
		t.Fatal(err)
	}
	if _, err := Read(root, "linked.md", 1024); err == nil {
		t.Fatal("Read accepted symlink")
	}
	if err := os.WriteFile(filepath.Join(root, ".env"), []byte("do not read"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Read(root, ".env", 1024); err == nil {
		t.Fatal("Read accepted secret-like path")
	}
}

func TestReplaceRefusesConcurrentChange(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "page.md")
	if err := os.WriteFile(path, []byte("before\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	snapshot, err := Read(root, path, 1024)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("changed\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := snapshot.Replace([]byte("after\n"), 1024); err == nil {
		t.Fatal("Replace accepted concurrent change")
	}
}

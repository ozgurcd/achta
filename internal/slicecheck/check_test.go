package slicecheck

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSecretPathClassification(t *testing.T) {
	if !secretPath([]string{"config/.env.local"}) {
		t.Fatal("secret-like path passed")
	}
	if secretPath([]string{"dev.env.example", "docs/key.pem.example"}) {
		t.Fatal("example path failed")
	}
}

func TestLogHeadings(t *testing.T) {
	got := logHeadings([]byte("# Log\r\n## [2026-01-01] one\r\nbody\r\n## unrelated\r\n## [2026-01-02] two\r\n"))
	if len(got) != 2 || got[0] != "## [2026-01-01] one" || got[1] != "## [2026-01-02] two" {
		t.Fatalf("headings = %#v", got)
	}
}

// RULE: SLICE-OWNED-WIKI-LOG-1
func TestRepositoryLogPrefersOwnedWiki(t *testing.T) {
	repo := t.TempDir()
	if err := os.WriteFile(filepath.Join(repo, "log.md"), []byte("root\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(repo, "wiki"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "wiki", "log.md"), []byte("owned\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := repositoryLog(repo)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join("wiki", "log.md")
	if got != want {
		t.Fatalf("repositoryLog = %q, want %q", got, want)
	}

	if err := os.Remove(filepath.Join(repo, "wiki", "log.md")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(repo, "log.md"), filepath.Join(repo, "wiki", "log.md")); err != nil {
		t.Fatal(err)
	}
	if _, err := repositoryLog(repo); err == nil {
		t.Fatal("repositoryLog accepted a linked owned wiki log")
	}
}

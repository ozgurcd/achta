package slicecheck

import (
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
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

// RULE: SLICE-LOG-DIRECTORY-1
func TestLogAppendDirectoryIsOrderedAndAppendOnly(t *testing.T) {
	t.Run("frozen file and later directory file both count", func(t *testing.T) {
		repo := newLogRepository(t, map[string]string{
			"log.md":              "# Log\n## [2026-09-04] frozen\n",
			"log/001-baseline.md": "## [2026-09-04] directory baseline\n",
		})
		base := strings.TrimSpace(runLogGit(t, repo, "rev-parse", "HEAD"))
		writeLogTestFile(t, repo, "log.md", "# Log\n## [2026-09-04] frozen\n## [2026-09-05] file append\n")
		writeLogTestFile(t, repo, "log/002-slice.md", "## [2026-09-05] directory append\n")
		runLogGit(t, repo, "add", "log.md", "log/002-slice.md")
		runLogGit(t, repo, "commit", "-m", "slice")

		if status, detail := logAppend(repo, base, "log.md", "log", 2); status != "pass" {
			t.Fatalf("status=%s detail=%s", status, detail)
		}
		if status, detail := logAppend(repo, base, "log.md", "log", 3); status != "fail" {
			t.Fatalf("missing entry status=%s detail=%s", status, detail)
		}
	})

	t.Run("lexically earlier file cannot reorder the directory", func(t *testing.T) {
		repo := newLogRepository(t, map[string]string{
			"log/200-baseline.md": "## [2026-09-04] frozen\n",
		})
		base := strings.TrimSpace(runLogGit(t, repo, "rev-parse", "HEAD"))
		writeLogTestFile(t, repo, "log/100-inserted.md", "## [2026-09-04] frozen\n## [2026-09-05] inserted\n")
		runLogGit(t, repo, "add", "log/100-inserted.md")
		runLogGit(t, repo, "commit", "-m", "reordered")

		if status, detail := logAppend(repo, base, "", "log", 2); status != "fail" {
			t.Fatalf("status=%s detail=%s", status, detail)
		}
	})

	t.Run("heading inserted inside its file is not an append", func(t *testing.T) {
		repo := newLogRepository(t, map[string]string{
			"log/100-entry.md": "## [2026-09-03] first\n## [2026-09-04] second\n",
		})
		base := strings.TrimSpace(runLogGit(t, repo, "rev-parse", "HEAD"))
		writeLogTestFile(t, repo, "log/100-entry.md", "## [2026-09-03] first\n## [2026-09-05] inserted\n## [2026-09-04] second\n")
		runLogGit(t, repo, "add", "log/100-entry.md")
		runLogGit(t, repo, "commit", "-m", "inserted")

		if status, detail := logAppend(repo, base, "", "log", 1); status != "fail" {
			t.Fatalf("status=%s detail=%s", status, detail)
		}
	})
}

func newLogRepository(t *testing.T, files map[string]string) string {
	t.Helper()
	repo := t.TempDir()
	runLogGit(t, repo, "init", "-b", "main")
	runLogGit(t, repo, "config", "user.name", "Achta Test")
	runLogGit(t, repo, "config", "user.email", "achta-test@example.invalid")
	for path, content := range files {
		writeLogTestFile(t, repo, path, content)
	}
	paths := make([]string, 0, len(files))
	for path := range files {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	runLogGit(t, repo, append([]string{"add"}, paths...)...)
	runLogGit(t, repo, "commit", "-m", "baseline")
	return repo
}

func writeLogTestFile(t *testing.T, repo, relative, content string) {
	t.Helper()
	path := filepath.Join(repo, relative)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func runLogGit(t *testing.T, repo string, args ...string) string {
	t.Helper()
	command := exec.Command("git", append([]string{"--no-optional-locks", "-C", repo}, args...)...)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v: %s", args, err, output)
	}
	return string(output)
}

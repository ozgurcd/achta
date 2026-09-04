package wiki

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// RULE: WIKI-FRESHNESS-1
func TestFreshnessCurrentBehindAndFilter(t *testing.T) {
	root := t.TempDir()
	repo := filepath.Join(root, "sample")
	mustMkdir(t, filepath.Join(root, "wiki", "repos"))
	mustMkdir(t, repo)
	runGitStatusTest(t, repo, "init", "-b", "main")
	runGitStatusTest(t, repo, "config", "user.name", "Achta Test")
	runGitStatusTest(t, repo, "config", "user.email", "achta@example.invalid")
	mustWrite(t, filepath.Join(repo, "x"), "one\n")
	runGitStatusTest(t, repo, "add", "x")
	runGitStatusTest(t, repo, "commit", "-m", "one")
	first := runGitStatusTest(t, repo, "rev-parse", "HEAD")
	mustWrite(t, filepath.Join(root, "wiki", "repos", "sample.md"), "---\ntitle: sample\ncategory: repo\nverified: 2026-09-04\nverified_against: sample @ "+first+" (main)\n---\n")

	result, err := Freshness(root, "sample")
	if err != nil || result.Status != "pass" || result.Fresh != 1 {
		t.Fatalf("fresh result = %+v, err=%v", result, err)
	}
	mustWrite(t, filepath.Join(repo, "x"), "two\n")
	runGitStatusTest(t, repo, "add", "x")
	runGitStatusTest(t, repo, "commit", "-m", "two")
	result, err = Freshness(root, "sample")
	if err != nil || result.Status != "drift" || result.Behind != 1 || result.Pages[0].CommitsBehind != 1 {
		t.Fatalf("behind result = %+v, err=%v", result, err)
	}
	if _, err := Freshness(root, "absent"); err == nil {
		t.Fatal("missing filter passed")
	}
}

func TestFreshnessDoesNotTakeSHAFromSuffix(t *testing.T) {
	root := t.TempDir()
	repo := filepath.Join(root, "sample")
	mustMkdir(t, filepath.Join(root, "wiki", "repos"))
	mustMkdir(t, repo)
	runGitStatusTest(t, repo, "init", "-b", "main")
	runGitStatusTest(t, repo, "config", "user.name", "Achta Test")
	runGitStatusTest(t, repo, "config", "user.email", "achta@example.invalid")
	mustWrite(t, filepath.Join(repo, "x"), "one\n")
	runGitStatusTest(t, repo, "add", "x")
	runGitStatusTest(t, repo, "commit", "-m", "one")
	old := runGitStatusTest(t, repo, "rev-parse", "HEAD")
	mustWrite(t, filepath.Join(repo, "x"), "two\n")
	runGitStatusTest(t, repo, "add", "x")
	runGitStatusTest(t, repo, "commit", "-m", "two")
	head := runGitStatusTest(t, repo, "rev-parse", "HEAD")
	mustWrite(t, filepath.Join(root, "wiki", "repos", "sample.md"), "---\ntitle: sample\ncategory: repo\nverified: 2026-09-04\nverified_against: sample @ "+head+" (mentions sample@"+old+")\n---\n")
	result, err := Freshness(root, "sample")
	if err != nil || result.Status != "pass" {
		t.Fatalf("suffix repointed pin: %+v, err=%v", result, err)
	}
}

func TestWikiOperationsRejectLinkedRepository(t *testing.T) {
	root := t.TempDir()
	mustMkdir(t, filepath.Join(root, "wiki", "repos"))
	external := t.TempDir()
	if err := os.Symlink(external, filepath.Join(root, "sample")); err != nil {
		t.Fatal(err)
	}
	page := "---\ntitle: sample\ncategory: repo\nverified: 2026-09-04\nverified_against: sample @ 0123456789abcdef0123456789abcdef01234567\n---\n"
	mustWrite(t, filepath.Join(root, "wiki", "repos", "sample.md"), page)
	result, err := Freshness(root, "sample")
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "cannot_evaluate" || len(result.Pages) != 1 || result.Pages[0].Status != "no_repo" {
		t.Fatalf("freshness = %+v", result)
	}
	if _, err := DerivedBlock(root, "sample"); err == nil {
		t.Fatal("DerivedBlock accepted a linked repository")
	}
}

func TestFreshnessFrontmatterRefusesAmbiguity(t *testing.T) {
	for name, source := range map[string]string{
		"duplicate":      "---\ntitle: sample\ncategory: repo\ncategory: repo\n---\n",
		"malformed":      "---\ntitle sample\n---\n",
		"missing closer": "---\ntitle: sample\n",
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := frontmatter([]byte(source)); err == nil {
				t.Fatal("ambiguous frontmatter accepted")
			}
		})
	}
}

func mustMkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func runGitStatusTest(t *testing.T, repo string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v: %s", args, err, out)
	}
	return string(bytesTrimSpace(out))
}

func bytesTrimSpace(value []byte) []byte {
	start, end := 0, len(value)
	for start < end && (value[start] == ' ' || value[start] == '\n' || value[start] == '\r' || value[start] == '\t') {
		start++
	}
	for end > start && (value[end-1] == ' ' || value[end-1] == '\n' || value[end-1] == '\r' || value[end-1] == '\t') {
		end--
	}
	return value[start:end]
}

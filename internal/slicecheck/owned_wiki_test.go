package slicecheck

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSliceOwnedWiki(t *testing.T) {
	for _, directory := range []string{"wiki", "llm-wiki"} {
		t.Run(directory, func(t *testing.T) {
			repo := newLogRepository(t, map[string]string{"file": "baseline\n", directory + "/log.md": "# Log\n"})
			name := filepath.Base(repo)
			page := directory + "/repos/" + name + ".md"
			log := directory + "/log.md"
			writeLogTestFile(t, repo, page, "---\ncategory: repo\nco_versioned: true\nverified: 2026-10-09\n---\n")
			writeLogTestFile(t, repo, log, "# Log\n\n## [2026-10-09] maintenance | record\n")
			runLogGit(t, repo, "add", page, log)
			runLogGit(t, repo, "commit", "-m", "record")
			result := Run(Options{Workspace: t.TempDir(), Repository: repo, Commits: 1, ExpectedEntries: 1})
			for _, check := range result.Checks {
				if check.Name == "wiki-pin" || check.Name == "log-append" {
					if check.Status != "pass" {
						t.Errorf("%s: %s: %s", check.Name, check.Status, check.Detail)
					}
				}
			}
			for _, invalid := range []string{
				"---\ncategory: repo\nco_versioned: true\n---\n",
				"---\ncategory: repo\nco_versioned: false\nverified: 2026-10-09\n---\n",
				"---\ncategory: repo\nco_versioned: true\nverified: 2026-10-09\nverified_against: other @ deadbee\n---\n",
			} {
				writeLogTestFile(t, repo, page, invalid)
				assertOwnedPinRefused(t, repo)
			}
			if err := os.Remove(filepath.Join(repo, page)); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(filepath.Join(repo, "file"), filepath.Join(repo, page)); err != nil {
				t.Fatal(err)
			}
			assertOwnedPinRefused(t, repo)
		})
	}
}

func assertOwnedPinRefused(t *testing.T, repo string) {
	t.Helper()
	result := Run(Options{Workspace: t.TempDir(), Repository: repo, Commits: 1, ExpectedEntries: 1})
	for _, check := range result.Checks {
		if check.Name == "wiki-pin" {
			if check.Status == "pass" || check.Status == "skip" {
				t.Fatalf("invalid owned page accepted: %+v", check)
			}
			return
		}
	}
	t.Fatal("wiki-pin check absent")
}

package slicecheck

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// newMainWithOrigin returns a repository whose main branch tracks a local bare
// origin, so the upstream is origin/main with no network.
func newMainWithOrigin(t *testing.T) string {
	t.Helper()
	repo := newLogRepository(t, map[string]string{"file": "baseline\n"})
	writeLogTestFile(t, repo, "file", "second\n")
	runLogGit(t, repo, "commit", "-am", "second")
	origin := filepath.Join(t.TempDir(), "origin.git")
	runLogGit(t, repo, "init", "--bare", "-b", "main", origin)
	runLogGit(t, repo, "remote", "add", "origin", origin)
	runLogGit(t, repo, "push", "-u", "origin", "main")
	return repo
}

// addPrivateWorktree adds a linked worktree on a new branch with no upstream
// and one commit, in a directory whose basename matches no wiki page.
func addPrivateWorktree(t *testing.T, repo string) string {
	t.Helper()
	worktree := filepath.Join(t.TempDir(), "job-private-1104")
	runLogGit(t, repo, "worktree", "add", "--no-track", "-b", "job/private", worktree, "main")
	writeLogTestFile(t, worktree, "file", "changed\n")
	runLogGit(t, worktree, "commit", "-am", "slice")
	return worktree
}

func checkOf(t *testing.T, result Result, name string) Check {
	t.Helper()
	for _, check := range result.Checks {
		if check.Name == name {
			return check
		}
	}
	t.Fatalf("check %s absent", name)
	return Check{}
}

func writeRepoPage(t *testing.T, workspace, name, sha string) {
	t.Helper()
	page := fmt.Sprintf("---\ncategory: repo\nverified: 2026-11-04\nverified_against: %s @ %s\n---\n", name, sha)
	writeLogTestFile(t, workspace, filepath.Join("wiki", "repos", name+".md"), page)
}

func headOf(t *testing.T, repo string) string {
	t.Helper()
	return strings.TrimSpace(runLogGit(t, repo, "rev-parse", "HEAD"))
}

func TestWorktreeUpstreamResolvesToMainRepository(t *testing.T) {
	repo := newMainWithOrigin(t)
	worktree := addPrivateWorktree(t, repo)
	one := 1
	result := Run(Options{Workspace: t.TempDir(), Repository: worktree, Commits: 1, ExpectedEntries: 0, ExactAhead: &one})
	if check := checkOf(t, result, "upstream"); check.Status != "pass" || !strings.Contains(check.Detail, "origin/main") {
		t.Fatalf("upstream: %+v", check)
	}
	if check := checkOf(t, result, "ahead"); check.Status != "pass" {
		t.Fatalf("ahead: %+v", check)
	}
	if check := checkOf(t, result, "ancestry"); check.Status != "pass" {
		t.Fatalf("ancestry: %+v", check)
	}
}

func TestWorktreeOwnUpstreamWinsOverMainRepository(t *testing.T) {
	repo := newMainWithOrigin(t)
	worktree := addPrivateWorktree(t, repo)
	runLogGit(t, worktree, "branch", "--set-upstream-to=main")
	result := Run(Options{Workspace: t.TempDir(), Repository: worktree, Commits: 1})
	if check := checkOf(t, result, "upstream"); check.Status != "pass" || !strings.Contains(check.Detail, "upstream is main") {
		t.Fatalf("upstream: %+v", check)
	}
}

func TestWorktreeWithoutAnyUpstreamStaysUnevaluable(t *testing.T) {
	repo := newLogRepository(t, map[string]string{"file": "baseline\n"})
	worktree := addPrivateWorktree(t, repo)
	for name, dir := range map[string]string{"main repo": repo, "worktree": worktree} {
		result := Run(Options{Workspace: t.TempDir(), Repository: dir, Commits: 1})
		if check := checkOf(t, result, "upstream"); check.Status != "cannot_evaluate" {
			t.Fatalf("%s: upstream: %+v", name, check)
		}
		if result.Status != "cannot_evaluate" {
			t.Fatalf("%s: status = %s", name, result.Status)
		}
	}
}

func TestExplicitUpstreamOverride(t *testing.T) {
	repo := newLogRepository(t, map[string]string{"file": "baseline\n"})
	worktree := addPrivateWorktree(t, repo)
	one := 1
	// An explicit commit that is neither HEAD nor a descendant of it is accepted.
	base := headOf(t, repo)
	result := Run(Options{Workspace: t.TempDir(), Repository: worktree, Commits: 1, ExactAhead: &one, Upstream: base})
	if check := checkOf(t, result, "upstream"); check.Status != "pass" || !strings.Contains(check.Detail, "explicit") || !strings.Contains(check.Detail, base) {
		t.Fatalf("upstream: %+v", check)
	}
	if check := checkOf(t, result, "ahead"); check.Status != "pass" {
		t.Fatalf("ahead: %+v", check)
	}
	for _, bad := range []string{"no-such-ref", "main\nother", "-x"} {
		result := Run(Options{Workspace: t.TempDir(), Repository: worktree, Commits: 1, Upstream: bad})
		if check := checkOf(t, result, "upstream"); check.Status != "cannot_evaluate" {
			t.Fatalf("upstream %q accepted: %+v", bad, check)
		}
	}
}

func TestExplicitUpstreamAcceptsRemoteTrackingRef(t *testing.T) {
	repo := newMainWithOrigin(t)
	worktree := addPrivateWorktree(t, repo)
	one := 1
	result := Run(Options{Workspace: t.TempDir(), Repository: worktree, Commits: 1, ExactAhead: &one, Upstream: "origin/main"})
	if check := checkOf(t, result, "upstream"); check.Status != "pass" || !strings.Contains(check.Detail, "origin/main") {
		t.Fatalf("upstream: %+v", check)
	}
	if check := checkOf(t, result, "ahead"); check.Status != "pass" {
		t.Fatalf("ahead: %+v", check)
	}
}

// An override that makes ahead and ancestry hold for any unpushed slice proves
// nothing, so it is refused and the dependent checks are skipped.
// RULE: SLICE-UPSTREAM-NONTRIVIAL-1
func TestExplicitUpstreamRefusesTrivialComparison(t *testing.T) {
	repo := newMainWithOrigin(t)
	worktree := addPrivateWorktree(t, repo)
	sliceHead := headOf(t, worktree)
	zero := 0
	// Reviewer R2: HEAD, HEAD~1, the slice's own branch, a local branch, the HEAD sha
	// itself and a descendant of HEAD (seen from the main checkout).
	cases := []struct {
		name, repo, upstream string
	}{
		{"HEAD", worktree, "HEAD"},
		{"HEAD~1", worktree, "HEAD~1"},
		{"own local branch", worktree, "job/private"},
		{"other local branch", worktree, "main"},
		{"HEAD sha", worktree, sliceHead},
		{"HEAD short sha", worktree, sliceHead[:10]},
		{"descendant sha", repo, sliceHead},
		{"descendant local branch", repo, "job/private"},
	}
	for _, tc := range cases {
		result := Run(Options{Workspace: t.TempDir(), Repository: tc.repo, Commits: 1, ExactAhead: &zero, Upstream: tc.upstream})
		upstream := checkOf(t, result, "upstream")
		if upstream.Status != "cannot_evaluate" {
			t.Fatalf("%s: trivial upstream accepted: %+v", tc.name, upstream)
		}
		for _, dependent := range []string{"ahead", "ancestry"} {
			if check := checkOf(t, result, dependent); check.Status != "skip" {
				t.Fatalf("%s: %s = %+v, want skip", tc.name, dependent, check)
			}
		}
		if result.Status != "cannot_evaluate" {
			t.Fatalf("%s: status = %s", tc.name, result.Status)
		}
	}
}

func TestExplicitUpstreamHeadWithAheadZeroDoesNotPassUnpushedSlice(t *testing.T) {
	repo := newMainWithOrigin(t)
	worktree := addPrivateWorktree(t, repo)
	zero := 0
	result := Run(Options{Workspace: t.TempDir(), Repository: worktree, Commits: 1, ExactAhead: &zero, Upstream: "HEAD"})
	if check := checkOf(t, result, "ahead"); check.Status == "pass" {
		t.Fatalf("unpushed slice passed ahead against HEAD: %+v", check)
	}
	// Against the real remote-tracking ref the same slice is correctly 1 ahead, not 0.
	result = Run(Options{Workspace: t.TempDir(), Repository: worktree, Commits: 1, ExactAhead: &zero, Upstream: "origin/main"})
	if check := checkOf(t, result, "ahead"); check.Status != "fail" {
		t.Fatalf("ahead against origin/main: %+v", check)
	}
}

// RULE: SLICE-WORKTREE-IDENTITY-1
func TestWorktreeWikiPageResolvesToMainRepositoryName(t *testing.T) {
	repo := newMainWithOrigin(t)
	worktree := addPrivateWorktree(t, repo)
	workspace := t.TempDir()
	name := filepath.Base(repo)
	writeRepoPage(t, workspace, name, headOf(t, worktree))
	result := Run(Options{Workspace: workspace, Repository: worktree, Commits: 1})
	if check := checkOf(t, result, "wiki-pin"); check.Status != "pass" {
		t.Fatalf("wiki-pin: %+v", check)
	}
	// The pin is judged against the checked worktree's HEAD, not the main checkout's.
	writeRepoPage(t, workspace, name, headOf(t, repo))
	result = Run(Options{Workspace: workspace, Repository: worktree, Commits: 1})
	if check := checkOf(t, result, "wiki-pin"); check.Status != "fail" {
		t.Fatalf("stale pin accepted: %+v", check)
	}
}

// RULE: SLICE-REPONAME-IDENTITY-1
func TestRepoNameOverrideMustMatchRepositoryIdentity(t *testing.T) {
	repo := newMainWithOrigin(t)
	worktree := addPrivateWorktree(t, repo)
	workspace := t.TempDir()
	name := filepath.Base(repo)
	writeRepoPage(t, workspace, name, headOf(t, worktree))
	// The name of the repository itself is accepted from its worktree.
	result := Run(Options{Workspace: workspace, Repository: worktree, Commits: 1, RepoName: name})
	if check := checkOf(t, result, "wiki-pin"); check.Status != "pass" {
		t.Fatalf("wiki-pin: %+v", check)
	}
	// Reviewer R1: another repository's page pinned to this slice's HEAD must not pass.
	writeRepoPage(t, workspace, "beta", headOf(t, worktree))
	for label, dir := range map[string]string{"main": repo, "worktree": worktree} {
		result := Run(Options{Workspace: workspace, Repository: dir, Commits: 1, RepoName: "beta"})
		check := checkOf(t, result, "wiki-pin")
		if check.Status != "cannot_evaluate" || !strings.Contains(check.Detail, `"beta"`) || !strings.Contains(check.Detail, name) {
			t.Fatalf("%s: mismatched repo name accepted or unnamed: %+v", label, check)
		}
	}
	for _, bad := range []string{"missing-page", "bad name", "../x", "a\nb"} {
		result := Run(Options{Workspace: workspace, Repository: repo, Commits: 1, RepoName: bad})
		if check := checkOf(t, result, "wiki-pin"); check.Status != "cannot_evaluate" {
			t.Fatalf("repo name %q accepted: %+v", bad, check)
		}
	}
}

func TestPlainRepositoryKeepsBasenamePage(t *testing.T) {
	repo := newMainWithOrigin(t)
	workspace := t.TempDir()
	if err := os.MkdirAll(filepath.Join(workspace, "wiki", "repos"), 0o755); err != nil {
		t.Fatal(err)
	}
	result := Run(Options{Workspace: workspace, Repository: repo, Commits: 1})
	if check := checkOf(t, result, "wiki-pin"); check.Status != "cannot_evaluate" {
		t.Fatalf("no page must stay unevaluable: %+v", check)
	}
}

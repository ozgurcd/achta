package gitstate

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// Checkout describes local metadata only; an unavailable upstream is explicit.
type Checkout struct {
	TrackedChanges      []string `json:"tracked_changes"`
	TrackedChangeCount  int      `json:"tracked_change_count"`
	UnpushedFiles       []string `json:"unpushed_files"`
	UnpushedCommitCount *int     `json:"unpushed_commit_count"`
	UpstreamAvailable   bool     `json:"upstream_available"`
}

// CheckoutGitDir resolves the per-worktree Git directory, never the common one.
func CheckoutGitDir(repo string) (string, error) {
	if !filepath.IsAbs(repo) {
		return "", errors.New("--repo must be an absolute checkout root")
	}
	for _, name := range []string{"GIT_DIR", "GIT_WORK_TREE", "GIT_COMMON_DIR", "GIT_INDEX_FILE"} {
		if os.Getenv(name) != "" {
			return "", errors.New("git checkout overrides are not supported")
		}
	}
	canonical, err := filepath.EvalSymlinks(repo)
	if err != nil {
		return "", errors.New("checkout root is unavailable")
	}
	metadata, err := os.Lstat(filepath.Join(canonical, ".git"))
	if err != nil || metadata.Mode()&os.ModeSymlink != 0 {
		return "", errors.New("linked or unavailable Git metadata is not supported")
	}
	top, err := run(repo, nil, "rev-parse", "--show-toplevel")
	if err != nil || filepath.Clean(strings.TrimSpace(top)) != canonical {
		return "", errors.New("--repo must select a Git checkout root")
	}
	dir, err := run(repo, nil, "rev-parse", "--absolute-git-dir")
	if err != nil {
		return "", errors.New("cannot resolve checkout Git directory")
	}
	dir = strings.TrimSpace(dir)
	if !filepath.IsAbs(dir) || strings.ContainsAny(dir, "\r\n\x00") {
		return "", errors.New("invalid checkout Git directory")
	}
	return dir, nil
}

// InspectCheckout counts changed tracked paths and commits absent from the local
// upstream ref. It does not fetch, read file contents, or run external diff tools.
func InspectCheckout(repo string) (Checkout, error) {
	c := Checkout{TrackedChanges: []string{}, UnpushedFiles: []string{}}
	out, err := run(repo, nil, "-c", "core.fsmonitor=false", "status", "--porcelain=v1", "-z", "--untracked-files=no", "--ignore-submodules=none")
	if err != nil {
		return c, errors.New("cannot measure tracked changes")
	}
	entries := strings.Split(strings.TrimSuffix(out, "\x00"), "\x00")
	paths := map[string]bool{}
	for i := 0; i < len(entries); i++ {
		entry := entries[i]
		if entry == "" {
			continue
		}
		if len(entry) < 4 || entry[2] != ' ' {
			return c, errors.New("invalid Git status record")
		}
		paths[entry[3:]] = true
		if strings.ContainsAny(entry[:2], "RC") {
			i++
			if i >= len(entries) || entries[i] == "" {
				return c, errors.New("incomplete Git rename record")
			}
			paths[entries[i]] = true
		}
	}
	for path := range paths {
		c.TrackedChanges = append(c.TrackedChanges, path)
	}
	sort.Strings(c.TrackedChanges)
	c.TrackedChangeCount = len(c.TrackedChanges)
	upstream, err := run(repo, nil, "rev-parse", "--verify", "@{upstream}^{commit}")
	if err != nil {
		return c, nil
	}
	upstream = strings.TrimSpace(upstream)
	if !validSHA(upstream) {
		return c, errors.New("invalid upstream commit")
	}
	out, err = run(repo, nil, "rev-list", "--count", upstream+"..HEAD", "--")
	if err != nil {
		return c, errors.New("cannot count local commits")
	}
	count, err := strconv.Atoi(strings.TrimSpace(out))
	if err != nil || count < 0 {
		return c, errors.New("invalid local commit count")
	}
	c.UpstreamAvailable, c.UnpushedCommitCount = true, &count
	// Include paths from every local commit, even changes later reverted.
	commits, err := run(repo, nil, "rev-list", upstream+"..HEAD", "--")
	if err != nil {
		return c, errors.New("cannot enumerate local commits")
	}
	for _, sha := range strings.Fields(commits) {
		if !validSHA(sha) {
			return c, errors.New("invalid local commit identity")
		}
	}
	out, err = run(repo, strings.NewReader(commits), "diff-tree", "--stdin", "--no-commit-id", "--root", "-m", "-r", "--name-only", "-z", "--no-renames", "--no-ext-diff", "--no-textconv")
	if err != nil {
		return c, errors.New("cannot enumerate local commit paths")
	}
	paths = map[string]bool{}
	for _, path := range strings.Split(out, "\x00") {
		if path != "" {
			paths[path] = true
		}
	}
	for path := range paths {
		c.UnpushedFiles = append(c.UnpushedFiles, path)
	}
	sort.Strings(c.UnpushedFiles)
	return c, nil
}

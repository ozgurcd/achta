package gitstate

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	commandTimeout = 10 * time.Second
	maxOutput      = 1 << 20
)

func Head(repo string) (string, error) {
	out, err := run(repo, nil, "rev-parse", "--verify", "HEAD^{commit}")
	if err != nil {
		return "", err
	}
	sha := strings.TrimSpace(out)
	if !validSHA(sha) {
		return "", errors.New("git returned malformed HEAD")
	}
	return sha, nil
}

func Branch(repo string) (string, error) {
	out, err := run(repo, nil, "symbolic-ref", "--quiet", "--short", "HEAD")
	if err != nil {
		var commandErr *CommandError
		if errors.As(err, &commandErr) && commandErr.ExitCode == 1 {
			return "detached", nil
		}
		return "", err
	}
	branch := strings.TrimSpace(out)
	if branch == "" || strings.ContainsAny(branch, "\r\n\x00") {
		return "", errors.New("git returned malformed branch")
	}
	return branch, nil
}

func ResolveCommit(repo, revision string) (string, error) {
	if strings.TrimSpace(revision) == "" || strings.ContainsAny(revision, "\r\n\x00") {
		return "", errors.New("invalid revision")
	}
	out, err := run(repo, nil, "rev-parse", "--verify", revision+"^{commit}")
	if err != nil {
		return "", err
	}
	sha := strings.TrimSpace(out)
	if !validSHA(sha) {
		return "", errors.New("git returned malformed commit")
	}
	return sha, nil
}

func IsAncestor(repo, base string) (bool, error) {
	if !validSHA(base) {
		return false, errors.New("base must be a full lowercase Git SHA")
	}
	_, err := run(repo, nil, "merge-base", "--is-ancestor", base, "HEAD")
	if err == nil {
		return true, nil
	}
	var commandErr *CommandError
	if errors.As(err, &commandErr) && commandErr.ExitCode == 1 {
		return false, nil
	}
	return false, err
}

func IsClean(repo string) (bool, error) {
	out, err := run(repo, nil, "-c", "core.excludesFile=/dev/null", "status", "--porcelain=v1", "--untracked-files=normal")
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(out) == "", nil
}

func IsCleanExcept(repo, exclude string) (bool, error) {
	clean := filepath.Clean(exclude)
	if filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return false, errors.New("excluded path must be repository-relative")
	}
	out, err := run(repo, nil, "-c", "core.excludesFile=/dev/null", "status", "--porcelain=v1", "--untracked-files=normal", "--", ".", ":(exclude)"+filepath.ToSlash(clean))
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(out) == "", nil
}

// Upstream reports the configured upstream reference without contacting a
// remote. A missing upstream is returned as an error so callers cannot mistake
// an unevaluable publication check for agreement.
func Upstream(repo string) (string, error) {
	out, err := run(repo, nil, "rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{u}")
	if err != nil {
		return "", fmt.Errorf("resolve upstream: %w", err)
	}
	upstream := strings.TrimSpace(out)
	if upstream == "" || strings.ContainsAny(upstream, "\r\n\x00") {
		return "", errors.New("git returned malformed upstream")
	}
	return upstream, nil
}

// AheadBehind compares HEAD with a local reference. It performs no fetch and
// therefore describes only the checked-out repository's remote-tracking state.
func AheadBehind(repo, reference string) (ahead, behind int, err error) {
	if strings.TrimSpace(reference) == "" || strings.ContainsAny(reference, "\r\n\x00") {
		return 0, 0, errors.New("invalid reference")
	}
	ahead, err = revisionCount(repo, reference+"..HEAD")
	if err != nil {
		return 0, 0, err
	}
	behind, err = revisionCount(repo, "HEAD.."+reference)
	if err != nil {
		return 0, 0, err
	}
	return ahead, behind, nil
}

func revisionCount(repo, revisionRange string) (int, error) {
	out, err := run(repo, nil, "rev-list", "--count", revisionRange)
	if err != nil {
		return 0, err
	}
	var count int
	if _, err := fmt.Sscanf(strings.TrimSpace(out), "%d", &count); err != nil || count < 0 {
		return 0, errors.New("git returned malformed revision count")
	}
	return count, nil
}

// TrackedPaths returns sorted repository-relative paths matching pathspec.
func TrackedPaths(repo, pathspec string) ([]string, error) {
	if pathspec == "" || strings.ContainsAny(pathspec, "\r\n\x00") {
		return nil, errors.New("invalid tracked-path pattern")
	}
	out, err := run(repo, nil, "ls-files", "-z", "--", pathspec)
	if err != nil {
		return nil, err
	}
	parts := strings.Split(out, "\x00")
	paths := make([]string, 0, len(parts))
	for _, path := range parts {
		if path != "" {
			paths = append(paths, path)
		}
	}
	sort.Strings(paths)
	return paths, nil
}

type CommitIdentity struct {
	SHA            string
	AuthorEmail    string
	CommitterEmail string
}

// ChangedPaths returns sorted paths changed between base and HEAD without
// reading their contents.
func ChangedPaths(repo, base string) ([]string, error) {
	if !validSHA(base) {
		return nil, errors.New("base must be a full lowercase Git SHA")
	}
	out, err := run(repo, nil, "diff", "--name-only", "-z", base, "HEAD")
	if err != nil {
		return nil, err
	}
	paths := strings.Split(out, "\x00")
	filtered := paths[:0]
	for _, path := range paths {
		if path != "" {
			filtered = append(filtered, path)
		}
	}
	sort.Strings(filtered)
	return filtered, nil
}

func CommitIdentities(repo string, count int) ([]CommitIdentity, error) {
	if count < 1 || count > 1000 {
		return nil, errors.New("commit count must be between 1 and 1000")
	}
	out, err := run(repo, nil, "log", fmt.Sprintf("-%d", count), "--format=%H%x1f%ae%x1f%ce")
	if err != nil {
		return nil, err
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	identities := make([]CommitIdentity, 0, len(lines))
	for _, line := range lines {
		parts := strings.Split(line, "\x1f")
		if len(parts) != 3 || !validSHA(parts[0]) {
			return nil, errors.New("git returned malformed commit identity")
		}
		identities = append(identities, CommitIdentity{SHA: parts[0], AuthorEmail: parts[1], CommitterEmail: parts[2]})
	}
	return identities, nil
}

func ConfiguredEmail(repo string) (string, error) {
	out, err := run(repo, nil, "config", "user.email")
	if err != nil {
		var commandErr *CommandError
		if errors.As(err, &commandErr) && commandErr.ExitCode == 1 {
			return "", nil
		}
		return "", err
	}
	value := strings.TrimSpace(out)
	if strings.ContainsAny(value, "\r\n\x00") {
		return "", errors.New("git returned malformed configured email")
	}
	return value, nil
}

func CommitAuthorEmail(repo, revision string) (string, error) {
	commit, err := ResolveCommit(repo, revision)
	if err != nil {
		return "", err
	}
	out, err := run(repo, nil, "show", "-s", "--format=%ae", commit)
	if err != nil {
		return "", err
	}
	value := strings.TrimSpace(out)
	if value == "" || strings.ContainsAny(value, "\r\n\x00") {
		return "", errors.New("git returned malformed author email")
	}
	return value, nil
}

func IsRefAncestor(repo, ancestor, descendant string) (bool, error) {
	if ancestor == "" || descendant == "" || strings.ContainsAny(ancestor+descendant, "\r\n\x00") {
		return false, errors.New("invalid ancestry reference")
	}
	_, err := run(repo, nil, "merge-base", "--is-ancestor", ancestor, descendant)
	if err == nil {
		return true, nil
	}
	var commandErr *CommandError
	if errors.As(err, &commandErr) && commandErr.ExitCode == 1 {
		return false, nil
	}
	return false, err
}

// FileAt reads one bounded, repository-relative non-secret file from a commit.
func FileAt(repo, revision, path string, limit int64) ([]byte, error) {
	if limit <= 0 || filepath.IsAbs(path) || path == ".." || strings.HasPrefix(path, ".."+string(filepath.Separator)) || strings.ContainsAny(path, "\r\n\x00") {
		return nil, errors.New("invalid repository file request")
	}
	base := strings.ToLower(filepath.Base(path))
	if strings.HasPrefix(base, ".env") || strings.HasSuffix(base, ".key") || strings.HasSuffix(base, ".pem") || strings.HasSuffix(base, ".p12") || strings.HasSuffix(base, ".pfx") || strings.HasSuffix(base, ".lic") {
		return nil, errors.New("refusing secret-like repository path")
	}
	commit, err := ResolveCommit(repo, revision)
	if err != nil {
		return nil, err
	}
	out, err := run(repo, nil, "show", commit+":"+filepath.ToSlash(path))
	if err != nil {
		return nil, err
	}
	if int64(len(out)) > limit {
		return nil, fmt.Errorf("repository file exceeds %d-byte limit", limit)
	}
	return []byte(out), nil
}

func NewestWitness(repo, witnessPath string) (string, error) {
	repoAbs, err := filepath.Abs(filepath.Clean(repo))
	if err != nil {
		return "", err
	}
	abs, err := filepath.Abs(filepath.Clean(witnessPath))
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(repoAbs, abs)
	if err != nil || rel == ".." || filepath.IsAbs(rel) || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", errors.New("witness log must be inside repository")
	}
	out, err := run(repo, nil, "log", "--format=%H%x1f%s", "HEAD", "--", filepath.ToSlash(rel))
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		parts := strings.SplitN(line, "\x1f", 2)
		if len(parts) != 2 || !validSHA(parts[0]) {
			return "", errors.New("git returned malformed witness history")
		}
		if strings.HasPrefix(parts[1], "Witness: make verify green at ") && strings.TrimPrefix(parts[1], "Witness: make verify green at ") != "" {
			return parts[0], nil
		}
	}
	return "", errors.New("no accepted witness commit found")
}

func TreeDigest(repo, exclude string) (string, error) {
	return treeDigest(repo, []string{exclude})
}

func TreeDigestAll(repo string) (string, error) {
	return treeDigest(repo, nil)
}

func treeDigest(repo string, excludes []string) (string, error) {
	args := []string{"ls-files", "-z", "--cached", "--others", "--exclude-standard", "--", "."}
	for _, exclude := range excludes {
		cleanExclude := filepath.Clean(exclude)
		if cleanExclude == "." || filepath.IsAbs(cleanExclude) || cleanExclude == ".." || strings.HasPrefix(cleanExclude, ".."+string(filepath.Separator)) {
			return "", errors.New("excluded record must be repository-relative")
		}
		args = append(args, ":(exclude)"+filepath.ToSlash(cleanExclude))
	}
	out, err := run(repo, nil, args...)
	if err != nil {
		return "", err
	}
	parts := bytes.Split([]byte(out), []byte{0})
	paths := make([]string, 0, len(parts))
	seen := make(map[string]struct{})
	for _, part := range parts {
		path := string(part)
		if path == "" {
			continue
		}
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		paths = append(paths, path)
	}
	sort.Strings(paths)
	if len(paths) == 0 {
		return "", errors.New("cannot digest an empty tree")
	}
	input := strings.Join(paths, "\n") + "\n"
	hashes, err := run(repo, strings.NewReader(input), "hash-object", "--stdin-paths")
	if err != nil {
		return "", err
	}
	hashLines := strings.Split(strings.TrimSuffix(hashes, "\n"), "\n")
	if len(hashLines) != len(paths) {
		return "", errors.New("git hash-object result count mismatch")
	}
	digest := sha256.New()
	for i, hash := range hashLines {
		if !validSHA(hash) {
			return "", errors.New("git hash-object returned malformed SHA")
		}
		fmt.Fprintf(digest, "%s %s\n", hash, paths[i])
	}
	return hex.EncodeToString(digest.Sum(nil)), nil
}

type CommandError struct {
	ExitCode int
	Message  string
}

func (e *CommandError) Error() string { return e.Message }

func run(repo string, stdin io.Reader, args ...string) (string, error) {
	gitPath, err := exec.LookPath("git")
	if err != nil {
		return "", &CommandError{ExitCode: 2, Message: fmt.Sprintf("discover Git executable %q: %v", "git", err)}
	}
	gitPath, err = filepath.Abs(gitPath)
	if err != nil {
		return "", &CommandError{ExitCode: 2, Message: fmt.Sprintf("canonicalize Git executable %q: %v", gitPath, err)}
	}
	gitPath = filepath.Clean(gitPath)
	ctx, cancel := context.WithTimeout(context.Background(), commandTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, gitPath, append([]string{"--no-optional-locks", "-C", repo}, args...)...)
	cmd.Stdin = stdin
	var stdout, stderr limitedBuffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()
	if ctx.Err() != nil {
		return "", fmt.Errorf("git executable %s timed out", gitPath)
	}
	if stdout.overflow || stderr.overflow {
		return "", fmt.Errorf("git executable %s output exceeded limit", gitPath)
	}
	if err != nil {
		code := 2
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			code = exitErr.ExitCode()
		}
		message := strings.TrimSpace(stderr.String())
		if message == "" {
			message = "git command failed"
		}
		if len(message) > 512 {
			message = message[:512] + "..."
		}
		return "", &CommandError{ExitCode: code, Message: fmt.Sprintf("git executable %s: %s", gitPath, message)}
	}
	return stdout.String(), nil
}

type limitedBuffer struct {
	buf      bytes.Buffer
	overflow bool
}

func (b *limitedBuffer) Write(p []byte) (int, error) {
	if b.buf.Len()+len(p) > maxOutput {
		remaining := maxOutput - b.buf.Len()
		if remaining > 0 {
			_, _ = b.buf.Write(p[:remaining])
		}
		b.overflow = true
		return len(p), nil
	}
	return b.buf.Write(p)
}

func (b *limitedBuffer) String() string { return b.buf.String() }

func validSHA(value string) bool {
	if len(value) != 40 {
		return false
	}
	for _, r := range value {
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f')) {
			return false
		}
	}
	return true
}

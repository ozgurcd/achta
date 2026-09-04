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
	out, err := run(repo, nil, "status", "--porcelain=v1", "--untracked-files=normal")
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(out) == "", nil
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
	out, err := run(repo, nil, "log", "--format=%H", "-1", "--fixed-strings", "--grep=Witness: make verify green at ", "HEAD", "--", filepath.ToSlash(rel))
	if err != nil {
		return "", err
	}
	sha := strings.TrimSpace(out)
	if !validSHA(sha) {
		return "", errors.New("no accepted witness commit found")
	}
	return sha, nil
}

func TreeDigest(repo, exclude string) (string, error) {
	cleanExclude := filepath.Clean(exclude)
	if filepath.IsAbs(cleanExclude) || cleanExclude == ".." || strings.HasPrefix(cleanExclude, ".."+string(filepath.Separator)) {
		return "", errors.New("excluded record must be repository-relative")
	}
	pathspec := ":(exclude)" + filepath.ToSlash(cleanExclude)
	out, err := run(repo, nil, "ls-files", "-z", "--cached", "--others", "--exclude-standard", "--", ".", pathspec)
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
	ctx, cancel := context.WithTimeout(context.Background(), commandTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", repo}, args...)...)
	cmd.Stdin = stdin
	var stdout, stderr limitedBuffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if ctx.Err() != nil {
		return "", errors.New("git timed out")
	}
	if stdout.overflow || stderr.overflow {
		return "", errors.New("git output exceeded limit")
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
		return "", &CommandError{ExitCode: code, Message: message}
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

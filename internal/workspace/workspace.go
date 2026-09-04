package workspace

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const maxWalkDepth = 128

// Workspace is a canonical workspace root with the expected Achta layout.
type Workspace struct {
	Root string
}

// Resolve validates an explicit workspace or discovers exactly one candidate
// while walking from start to the filesystem root.
func Resolve(explicit, start string) (Workspace, error) {
	if explicit != "" {
		root, err := canonicalDirectory(explicit)
		if err != nil {
			return Workspace{}, fmt.Errorf("workspace: %w", err)
		}
		if !hasLayout(root) {
			return Workspace{}, fmt.Errorf("workspace %q lacks wiki/repos and wiki/platform/decisions.md", root)
		}
		return Workspace{Root: root}, nil
	}

	root, err := canonicalDirectory(start)
	if err != nil {
		return Workspace{}, fmt.Errorf("starting directory: %w", err)
	}
	var candidates []string
	for depth := 0; depth < maxWalkDepth; depth++ {
		if hasLayout(root) {
			candidates = append(candidates, root)
		}
		parent := filepath.Dir(root)
		if parent == root {
			break
		}
		root = parent
	}
	if len(candidates) == 0 {
		return Workspace{}, errors.New("cannot discover workspace; pass --workspace")
	}
	if len(candidates) != 1 {
		return Workspace{}, fmt.Errorf("ambiguous workspace discovery (%d candidates); pass --workspace", len(candidates))
	}
	return Workspace{Root: candidates[0]}, nil
}

func canonicalDirectory(path string) (string, error) {
	abs, err := filepath.Abs(filepath.Clean(path))
	if err != nil {
		return "", err
	}
	real, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(real)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", fmt.Errorf("%q is not a directory", real)
	}
	return real, nil
}

func hasLayout(root string) bool {
	repos, err := os.Stat(filepath.Join(root, "wiki", "repos"))
	if err != nil || !repos.IsDir() {
		return false
	}
	decisions, err := os.Stat(filepath.Join(root, "wiki", "platform", "decisions.md"))
	return err == nil && decisions.Mode().IsRegular()
}

// Confine resolves a path beneath the workspace without requiring it to exist.
func (w Workspace) Confine(path string) (string, error) {
	if !filepath.IsAbs(path) {
		path = filepath.Join(w.Root, path)
	}
	clean, err := filepath.Abs(filepath.Clean(path))
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(w.Root, clean)
	if err != nil || rel == ".." || filepath.IsAbs(rel) || startsWithParent(rel) {
		return "", fmt.Errorf("path %q escapes workspace", path)
	}
	current := w.Root
	for _, component := range strings.Split(rel, string(filepath.Separator)) {
		if component == "." || component == "" {
			continue
		}
		current = filepath.Join(current, component)
		info, statErr := os.Lstat(current)
		if errors.Is(statErr, os.ErrNotExist) {
			break
		}
		if statErr != nil {
			return "", fmt.Errorf("inspect confined path component %q: %w", component, statErr)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("linked path component rejected: %q", component)
		}
	}
	return clean, nil
}

func startsWithParent(rel string) bool {
	return len(rel) > 3 && rel[:3] == ".."+string(filepath.Separator)
}

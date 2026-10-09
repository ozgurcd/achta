package gitstate

import (
	"errors"
	"path/filepath"
	"strings"
)

// DirtyFiles counts porcelain entries including untracked files, excluding
// ignored files. A rename is one changed entry, not two filenames.
func DirtyFiles(repo string) (int, error) {
	out, err := run(repo, nil, "-c", "core.fsmonitor=false", "status", "--porcelain=v1", "-z", "--untracked-files=all")
	if err != nil {
		return 0, errors.New("cannot count dirty files")
	}
	entries := strings.Split(out, "\x00")
	count := 0
	for i := 0; i < len(entries); i++ {
		if entries[i] == "" {
			continue
		}
		if len(entries[i]) < 4 {
			return 0, errors.New("invalid status entry")
		}
		count++
		if strings.ContainsAny(entries[i][:2], "RC") {
			i++
		}
	}
	return count, nil
}

// JournalWitness reports a recorded witness without changing gate acceptance.
func JournalWitness(repo, path string) (string, error) {
	rel, err := filepath.Rel(repo, path)
	if err != nil || filepath.IsAbs(rel) || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", errors.New("witness outside repository")
	}
	out, err := run(repo, nil, "log", "--format=%H%x1f%s", "HEAD", "--", filepath.ToSlash(rel))
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		parts := strings.SplitN(line, "\x1f", 2)
		if len(parts) != 2 || !validSHA(parts[0]) {
			continue
		}
		for _, prefix := range []string{"Witness: make verify green at ", "Witness: make check green at "} {
			if strings.HasPrefix(parts[1], prefix) && strings.TrimPrefix(parts[1], prefix) != "" {
				return parts[0], nil
			}
		}
	}
	return "", errors.New("no recorded witness found")
}

package wiki

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ozgurcd/achta/internal/workspace"
)

// Selection records explicit reasons for repository pages not being judged.
// A nil selection judges everything. Selection never changes a judgement.
type Selection map[string]string

func repositoryPageName(name string) string {
	if name == "identuum-ai" {
		return "identuum.ai"
	}
	return name
}

// SelectRepositories validates the complete scope before any checks execute.
func SelectRepositories(root string, include, exclude []string) (Selection, error) {
	if len(include) == 0 && len(exclude) == 0 {
		return nil, nil
	}
	directory, err := (workspace.Workspace{Root: root}).Confine("wiki/repos")
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		return nil, err
	}
	known := map[string]bool{}
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".md" {
			known[repositoryPageName(strings.TrimSuffix(e.Name(), ".md"))] = true
		}
	}
	selected, omitted := map[string]bool{}, Selection{}
	for _, value := range include {
		name := repositoryPageName(value)
		if !repoPattern.MatchString(name) || !known[name] || selected[name] {
			return nil, fmt.Errorf("invalid, unknown, or repeated --repo %q", value)
		}
		selected[name] = true
	}
	for _, value := range exclude {
		name, reason, ok := strings.Cut(value, "=")
		name = repositoryPageName(name)
		if !ok || strings.TrimSpace(reason) == "" || strings.ContainsAny(reason, "\r\n\x00") || !repoPattern.MatchString(name) || !known[name] || selected[name] || omitted[name] != "" {
			return nil, fmt.Errorf("--exclude requires a unique unselected repository and nonempty reason: NAME=REASON")
		}
		omitted[name] = reason
	}
	if len(selected) > 0 {
		for name := range known {
			if !selected[name] && omitted[name] == "" {
				omitted[name] = "not selected by --repo"
			}
		}
	}
	return omitted, nil
}

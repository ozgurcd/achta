package wiki

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/ozgurcd/achta/internal/gitstate"
	"github.com/ozgurcd/achta/internal/safefile"
	"github.com/ozgurcd/achta/internal/workspace"
)

const (
	FreshnessSchema = "achta.wiki-freshness.v1"
	UnpushedSchema  = "achta.wiki-unpushed.v1"
)

var verifiedAgainstPattern = regexp.MustCompile(`^([^[:space:]]+)[[:space:]]+@[[:space:]]+([0-9a-f]{7,40})(.*)$`)

type FreshnessPage struct {
	Repository    string `json:"repository"`
	Page          string `json:"page"`
	Status        string `json:"status"`
	Verified      string `json:"verified,omitempty"`
	DeclaredSHA   string `json:"declared_sha,omitempty"`
	HeadSHA       string `json:"head_sha,omitempty"`
	CommitsBehind int    `json:"commits_behind,omitempty"`
	Problem       string `json:"problem,omitempty"`
}

type FreshnessResult struct {
	SchemaVersion string          `json:"schema_version"`
	Status        string          `json:"status"`
	Fresh         int             `json:"fresh"`
	Behind        int             `json:"behind"`
	Unpinned      int             `json:"unpinned"`
	Unreadable    int             `json:"unreadable"`
	Skipped       int             `json:"skipped"`
	Pages         []FreshnessPage `json:"pages"`
}

type UnpushedResult struct {
	SchemaVersion string `json:"schema_version"`
	Status        string `json:"status"`
	Repository    string `json:"repository"`
	Branch        string `json:"branch,omitempty"`
	Upstream      string `json:"upstream,omitempty"`
	Ahead         int    `json:"ahead"`
	Behind        int    `json:"behind"`
	Limitation    string `json:"limitation"`
}

// Freshness checks repository-page pins against local repository HEADs. It
// performs no fetch; its result is a staleness signal, never a truth verdict.
func Freshness(root, filter string) (FreshnessResult, error) {
	result := FreshnessResult{SchemaVersion: FreshnessSchema, Status: "pass", Pages: []FreshnessPage{}}
	if filter != "" && !repoPattern.MatchString(filter) {
		return result, errors.New("invalid repository filter")
	}
	boundary := workspace.Workspace{Root: root}
	directory, err := boundary.Confine(filepath.Join("wiki", "repos"))
	if err != nil {
		return result, fmt.Errorf("confine wiki repository pages: %w", err)
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		return result, fmt.Errorf("read wiki repository pages: %w", err)
	}
	matched := false
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".md" {
			continue
		}
		name := strings.TrimSuffix(entry.Name(), ".md")
		repoName := name
		if name == "identuum-ai" {
			repoName = "identuum.ai"
		}
		if filter != "" && name != filter && repoName != filter {
			continue
		}
		matched = true
		page := FreshnessPage{Repository: repoName, Page: filepath.ToSlash(filepath.Join("wiki", "repos", entry.Name()))}
		snapshot, readErr := safefile.Read(root, filepath.Join(directory, entry.Name()), MaxPageSize)
		if readErr != nil {
			page.Status, page.Problem = "unreadable", boundedProblem(readErr)
			result.Unreadable++
			result.Pages = append(result.Pages, page)
			continue
		}
		fields, parseErr := frontmatter(snapshot.Data)
		if parseErr != nil {
			page.Status, page.Problem = "malformed", boundedProblem(parseErr)
			result.Unreadable++
			result.Pages = append(result.Pages, page)
			continue
		}
		if fields["category"] != "repo" {
			result.Skipped++
			continue
		}
		page.Verified = fields["verified"]
		declared := fields["verified_against"]
		match := verifiedAgainstPattern.FindStringSubmatch(declared)
		if page.Verified == "" || match == nil {
			page.Status = "unpinned"
			page.Problem = "category: repo requires verified and canonical verified_against fields"
			result.Unpinned++
			result.Pages = append(result.Pages, page)
			continue
		}
		if match[1] != repoName {
			page.Status, page.Problem = "malformed", "verified_against repository does not match page"
			result.Unreadable++
			result.Pages = append(result.Pages, page)
			continue
		}
		page.DeclaredSHA = match[2]
		repo, confineErr := boundary.Confine(repoName)
		if confineErr != nil {
			page.Status, page.Problem = "no_repo", boundedProblem(confineErr)
			result.Unreadable++
			result.Pages = append(result.Pages, page)
			continue
		}
		head, headErr := gitstate.Head(repo)
		if headErr != nil {
			page.Status, page.Problem = "no_repo", boundedProblem(headErr)
			result.Unreadable++
			result.Pages = append(result.Pages, page)
			continue
		}
		page.HeadSHA = head
		declaredFull, resolveErr := gitstate.ResolveCommit(repo, page.DeclaredSHA)
		if resolveErr != nil {
			page.Status, page.Problem = "gone", "declared SHA is not a commit in the repository"
			result.Behind++
			result.Pages = append(result.Pages, page)
			continue
		}
		if declaredFull == head {
			page.Status = "fresh"
			result.Fresh++
		} else {
			page.Status = "behind"
			page.CommitsBehind, _, _ = gitstate.AheadBehind(repo, declaredFull)
			result.Behind++
		}
		result.Pages = append(result.Pages, page)
	}
	if filter != "" && !matched {
		return result, fmt.Errorf("repository filter %q matches no wiki page", filter)
	}
	if result.Unreadable > 0 {
		result.Status = "cannot_evaluate"
	} else if result.Behind > 0 || result.Unpinned > 0 {
		result.Status = "drift"
	}
	return result, nil
}

func Unpushed(repo string) (UnpushedResult, error) {
	result := UnpushedResult{
		SchemaVersion: UnpushedSchema,
		Repository:    filepath.Base(repo),
		Limitation:    "uses the local upstream tracking reference and performs no fetch",
	}
	if _, err := gitstate.Head(repo); err != nil {
		return result, fmt.Errorf("repository HEAD: %w", err)
	}
	branch, err := gitstate.Branch(repo)
	if err != nil {
		return result, fmt.Errorf("repository branch: %w", err)
	}
	result.Branch = branch
	upstream, err := gitstate.Upstream(repo)
	if err != nil {
		return result, err
	}
	result.Upstream = upstream
	result.Ahead, result.Behind, err = gitstate.AheadBehind(repo, upstream)
	if err != nil {
		return result, err
	}
	switch {
	case result.Ahead > 0 && result.Behind > 0:
		result.Status = "diverged"
	case result.Ahead > 0:
		result.Status = "unpushed"
	case result.Behind > 0:
		result.Status = "behind"
	default:
		result.Status = "published"
	}
	return result, nil
}

func frontmatter(data []byte) (map[string]string, error) {
	data = bytes.ReplaceAll(data, []byte("\r\n"), []byte("\n"))
	lines := strings.Split(string(data), "\n")
	if len(lines) < 3 || lines[0] != "---" {
		return nil, errors.New("missing frontmatter opener")
	}
	fields := make(map[string]string)
	for i := 1; i < len(lines); i++ {
		if lines[i] == "---" {
			return fields, nil
		}
		key, value, ok := strings.Cut(lines[i], ":")
		if !ok || key == "" {
			return nil, fmt.Errorf("malformed frontmatter line %d", i+1)
		}
		if _, exists := fields[key]; exists {
			return nil, fmt.Errorf("duplicate frontmatter field %q", key)
		}
		fields[key] = strings.TrimSpace(value)
	}
	return nil, errors.New("missing frontmatter closer")
}

func boundedProblem(err error) string {
	message := strings.ReplaceAll(err.Error(), "\n", " ")
	if len(message) > 256 {
		message = message[:256] + "..."
	}
	return message
}

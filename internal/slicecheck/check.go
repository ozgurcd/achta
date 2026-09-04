package slicecheck

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
	achtawiki "github.com/ozgurcd/achta/internal/wiki"
)

const Schema = "achta.slice-check.v1"

var secretPathPattern = regexp.MustCompile(`(?i)(^|/)\.env($|\.)|(^|/)[^/]*\.env$|\.(lic|key|pem|p12|pfx)$|(^|/)id_(rsa|ed25519)$`)

type Options struct {
	Workspace       string
	Repository      string
	Commits         int
	ExpectedEntries int
	ExactAhead      *int
}

type Check struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Detail string `json:"detail"`
}

type Result struct {
	SchemaVersion string  `json:"schema_version"`
	Status        string  `json:"status"`
	Repository    string  `json:"repository"`
	Commits       int     `json:"commits"`
	Passed        int     `json:"passed"`
	Failed        int     `json:"failed"`
	Skipped       int     `json:"skipped"`
	Checks        []Check `json:"checks"`
}

func Run(options Options) Result {
	result := Result{SchemaVersion: Schema, Status: "pass", Repository: filepath.Base(options.Repository), Commits: options.Commits, Checks: []Check{}}
	add := func(name, status, detail string) {
		result.Checks = append(result.Checks, Check{Name: name, Status: status, Detail: bounded(detail)})
		switch status {
		case "pass":
			result.Passed++
		case "skip":
			result.Skipped++
		case "fail":
			result.Failed++
			if result.Status == "pass" {
				result.Status = "fail"
			}
		default:
			result.Failed++
			result.Status = "cannot_evaluate"
		}
	}
	if options.Commits < 1 || options.Commits > 1000 {
		add("input", "cannot_evaluate", "--commits must be between 1 and 1000")
		return result
	}
	if options.ExpectedEntries < 0 || options.ExpectedEntries > 1000 {
		add("input", "cannot_evaluate", "--entries must be between 0 and 1000")
		return result
	}

	clean, err := gitstate.IsClean(options.Repository)
	if err != nil {
		add("clean-tree", "cannot_evaluate", err.Error())
	} else if clean {
		add("clean-tree", "pass", "working tree is clean")
	} else {
		add("clean-tree", "fail", "working tree is not clean")
	}

	upstream, upstreamErr := gitstate.Upstream(options.Repository)
	if upstreamErr != nil {
		add("upstream", "cannot_evaluate", upstreamErr.Error())
		add("ahead", "skip", "no upstream")
		add("ancestry", "skip", "no upstream")
	} else {
		add("upstream", "pass", "upstream is "+upstream)
		ahead, behind, compareErr := gitstate.AheadBehind(options.Repository, upstream)
		if compareErr != nil {
			add("ahead", "cannot_evaluate", compareErr.Error())
			add("ancestry", "cannot_evaluate", compareErr.Error())
		} else {
			if options.ExactAhead != nil {
				if ahead == *options.ExactAhead {
					add("ahead", "pass", fmt.Sprintf("exactly %d commit(s) ahead", ahead))
				} else {
					add("ahead", "fail", fmt.Sprintf("expected exactly %d commit(s) ahead, found %d", *options.ExactAhead, ahead))
				}
			} else if ahead >= options.Commits {
				add("ahead", "pass", fmt.Sprintf("slice commits remain local; %d ahead in total", ahead))
			} else {
				add("ahead", "fail", fmt.Sprintf("expected at least %d commit(s) ahead, found %d", options.Commits, ahead))
			}
			ancestor, ancestorErr := gitstate.IsRefAncestor(options.Repository, upstream, "HEAD")
			if ancestorErr != nil {
				add("ancestry", "cannot_evaluate", ancestorErr.Error())
			} else if behind == 0 && ancestor {
				add("ancestry", "pass", "upstream is an ancestor of HEAD")
			} else {
				add("ancestry", "fail", fmt.Sprintf("upstream is not cleanly behind HEAD; behind=%d", behind))
			}
		}
	}

	base, baseErr := gitstate.ResolveCommit(options.Repository, fmt.Sprintf("HEAD~%d", options.Commits))
	if baseErr != nil {
		add("slice-base", "cannot_evaluate", baseErr.Error())
		return result
	}
	identities, identityErr := gitstate.CommitIdentities(options.Repository, options.Commits)
	if identityErr != nil {
		add("commit-identity", "cannot_evaluate", identityErr.Error())
		add("agent-identity", "cannot_evaluate", identityErr.Error())
	} else {
		wanted, configErr := gitstate.ConfiguredEmail(options.Repository)
		if configErr == nil && wanted == "" {
			wanted, configErr = gitstate.CommitAuthorEmail(options.Repository, base)
		}
		if configErr != nil || wanted == "" {
			add("commit-identity", "cannot_evaluate", "cannot determine owner commit identity")
		} else {
			mismatch := false
			for _, identity := range identities {
				if identity.AuthorEmail != wanted || identity.CommitterEmail != wanted {
					mismatch = true
				}
			}
			if mismatch {
				add("commit-identity", "fail", "slice author or committer differs from configured owner identity")
			} else {
				add("commit-identity", "pass", "slice author and committer match the owner identity")
			}
		}
		agent := false
		for _, identity := range identities {
			value := strings.ToLower(identity.AuthorEmail + " " + identity.CommitterEmail)
			if strings.Contains(value, "anthropic") || strings.Contains(value, "claude") || strings.Contains(value, "openai") || strings.Contains(value, "chatgpt") || strings.Contains(value, "[bot]") {
				agent = true
			}
		}
		if agent {
			add("agent-identity", "fail", "a slice commit carries an agent identity")
		} else {
			add("agent-identity", "pass", "no slice commit carries an agent identity")
		}
	}

	paths, pathsErr := gitstate.ChangedPaths(options.Repository, base)
	if pathsErr != nil {
		add("secret-paths", "cannot_evaluate", pathsErr.Error())
		add("module-boundary", "cannot_evaluate", pathsErr.Error())
	} else {
		if secretPath(paths) {
			add("secret-paths", "fail", "slice contains a secret-like path")
		} else {
			add("secret-paths", "pass", "slice contains no secret-like path")
		}
		if problem, err := moduleBoundary(options.Repository, paths); err != nil {
			add("module-boundary", "cannot_evaluate", err.Error())
		} else if problem != "" {
			add("module-boundary", "fail", problem)
		} else {
			add("module-boundary", "pass", "no go.work or replace directive introduced")
		}
	}

	logRelative, logErr := repositoryLog(options.Repository)
	if errors.Is(logErr, os.ErrNotExist) {
		add("log-append", "skip", "repository has no log.md")
	} else if logErr != nil {
		add("log-append", "cannot_evaluate", logErr.Error())
	} else if status, detail := logAppend(options.Repository, base, logRelative, options.ExpectedEntries); status != "pass" {
		add("log-append", status, detail)
	} else {
		add("log-append", "pass", detail)
	}

	freshness, freshnessErr := achtawiki.Freshness(options.Workspace, result.Repository)
	if freshnessErr != nil {
		add("wiki-pin", "cannot_evaluate", freshnessErr.Error())
	} else if freshness.Status == "pass" && freshness.Fresh == 1 {
		add("wiki-pin", "pass", "repository wiki page is pinned to HEAD")
	} else {
		add("wiki-pin", "fail", "repository wiki page is not pinned to HEAD")
	}
	return result
}

func secretPath(paths []string) bool {
	for _, path := range paths {
		if secretPathPattern.MatchString(filepath.ToSlash(path)) && !strings.Contains(path, ".example") {
			return true
		}
	}
	return false
}

func moduleBoundary(repo string, paths []string) (string, error) {
	for _, path := range paths {
		base := filepath.Base(path)
		if base == "go.work" || base == "go.work.sum" {
			return "slice introduces go.work or go.work.sum", nil
		}
		if base != "go.mod" {
			continue
		}
		snapshot, err := safefile.Read(repo, filepath.Join(repo, path), 4<<20)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return "", err
		}
		for _, line := range strings.Split(string(snapshot.Data), "\n") {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "replace ") || trimmed == "replace (" || strings.Contains(trimmed, " => ") {
				return "changed go.mod contains a replace directive", nil
			}
		}
	}
	return "", nil
}

func repositoryLog(repo string) (string, error) {
	for _, relative := range []string{filepath.Join("wiki", "log.md"), "log.md"} {
		path := filepath.Join(repo, relative)
		info, err := os.Lstat(path)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return "", err
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			return "", fmt.Errorf("repository log must be a regular non-linked file: %s", relative)
		}
		return relative, nil
	}
	return "", os.ErrNotExist
}

func logAppend(repo, base, relative string, expected int) (string, string) {
	current, err := safefile.Read(repo, filepath.Join(repo, relative), 8<<20)
	if err != nil {
		return "cannot_evaluate", err.Error()
	}
	previous, err := gitstate.FileAt(repo, base, relative, 8<<20)
	if err != nil {
		return "cannot_evaluate", err.Error()
	}
	before := logHeadings(previous)
	after := logHeadings(current.Data)
	if len(after) != len(before)+expected || !equalStrings(before, after[:min(len(before), len(after))]) {
		return "fail", fmt.Sprintf("expected %d appended log heading(s), found %d", expected, len(after)-len(before))
	}
	return "pass", fmt.Sprintf("exactly %d log heading(s) appended at the end", expected)
}

func logHeadings(data []byte) []string {
	var headings []string
	for _, line := range bytes.Split(bytes.ReplaceAll(data, []byte("\r\n"), []byte("\n")), []byte("\n")) {
		if bytes.HasPrefix(line, []byte("## [")) {
			headings = append(headings, string(line))
		}
	}
	return headings
}

func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

func bounded(value string) string {
	value = strings.ReplaceAll(value, "\n", " ")
	if len(value) > 512 {
		return value[:512] + "..."
	}
	return value
}

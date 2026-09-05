package slicecheck

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/ozgurcd/achta/internal/gitstate"
	"github.com/ozgurcd/achta/internal/safefile"
	achtawiki "github.com/ozgurcd/achta/internal/wiki"
)

const Schema = "achta.slice-check.v1"

const maxLogFiles = 10_000

var secretPathPattern = regexp.MustCompile(`(?i)(^|/)\.env($|\.)|(^|/)[^/]*\.env$|\.(lic|key|pem|p12|pfx)$|(^|/)id_(rsa|ed25519)$`)

type Options struct {
	Workspace       string
	Repository      string
	Commits         int
	ExpectedEntries int
	ExactAhead      *int
	LogDirectory    string
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
	logDirectory, directoryErr := repositoryLogDirectory(options.Repository, options.LogDirectory)
	if errors.Is(logErr, os.ErrNotExist) && options.LogDirectory == "" {
		add("log-append", "skip", "repository has no log.md and no --log-dir")
	} else if logErr != nil && !errors.Is(logErr, os.ErrNotExist) {
		add("log-append", "cannot_evaluate", logErr.Error())
	} else if options.LogDirectory != "" && directoryErr != nil {
		add("log-append", "cannot_evaluate", directoryErr.Error())
	} else {
		if errors.Is(logErr, os.ErrNotExist) {
			logRelative = ""
		}
		if options.LogDirectory == "" {
			logDirectory = ""
		}
		status, detail := logAppend(options.Repository, base, logRelative, logDirectory, options.ExpectedEntries)
		add("log-append", status, detail)
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

func repositoryLogDirectory(repo, relative string) (string, error) {
	if relative == "" {
		return "", os.ErrNotExist
	}
	clean := filepath.Clean(relative)
	if clean == "." || filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) || strings.ContainsAny(relative, "\r\n\x00") {
		return "", errors.New("--log-dir must be a repository-relative directory")
	}
	current := repo
	for _, component := range strings.Split(clean, string(filepath.Separator)) {
		current = filepath.Join(current, component)
		info, err := os.Lstat(current)
		if err != nil {
			return "", fmt.Errorf("inspect --log-dir %s: %w", clean, err)
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return "", fmt.Errorf("--log-dir must be a non-linked directory: %s", clean)
		}
	}
	return clean, nil
}

type logDocument struct {
	path     string
	headings []string
}

func logAppend(repo, base, logFile, logDirectory string, expected int) (string, string) {
	before, err := historicalLogDocuments(repo, base, logFile, logDirectory)
	if err != nil {
		return "cannot_evaluate", err.Error()
	}
	after, err := currentLogDocuments(repo, logFile, logDirectory)
	if err != nil {
		return "cannot_evaluate", err.Error()
	}
	if len(after) < len(before) {
		return "fail", "log sources were removed or reordered"
	}
	appended := 0
	for index, previous := range before {
		current := after[index]
		if current.path != previous.path {
			return "fail", "log directory filenames were inserted, removed, or reordered"
		}
		if len(current.headings) < len(previous.headings) || !equalStrings(previous.headings, current.headings[:min(len(previous.headings), len(current.headings))]) {
			return "fail", fmt.Sprintf("prior log headings changed or were not preserved at the start of %s", current.path)
		}
		appended += len(current.headings) - len(previous.headings)
	}
	for _, current := range after[len(before):] {
		appended += len(current.headings)
	}
	if appended != expected {
		return "fail", fmt.Sprintf("expected %d appended log heading(s), found %d", expected, appended)
	}
	return "pass", fmt.Sprintf("exactly %d log heading(s) appended at the end of ordered log sources", expected)
}

func historicalLogDocuments(repo, base, logFile, logDirectory string) ([]logDocument, error) {
	var paths []string
	if logFile != "" {
		paths = append(paths, logFile)
	}
	if logDirectory != "" {
		directoryPaths, err := gitstate.TreeFilesAt(repo, base, logDirectory)
		if err != nil {
			return nil, err
		}
		paths = append(paths, directoryPaths...)
	}
	if len(paths) > maxLogFiles {
		return nil, fmt.Errorf("log sources exceed %d-file limit", maxLogFiles)
	}
	documents := make([]logDocument, 0, len(paths))
	for index, path := range paths {
		if index > 0 && path == logFile {
			return nil, fmt.Errorf("--log-dir overlaps discovered log file: %s", logFile)
		}
		data, err := gitstate.FileAt(repo, base, path, 8<<20)
		if err != nil {
			return nil, err
		}
		documents = append(documents, logDocument{path: filepath.ToSlash(path), headings: logHeadings(data)})
	}
	return documents, nil
}

func currentLogDocuments(repo, logFile, logDirectory string) ([]logDocument, error) {
	var paths []string
	if logFile != "" {
		paths = append(paths, logFile)
	}
	if logDirectory != "" {
		var directoryPaths []string
		root := filepath.Join(repo, logDirectory)
		err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if path == root {
				return nil
			}
			if entry.Type()&os.ModeSymlink != 0 {
				return fmt.Errorf("repository log directory contains a linked entry: %s", path)
			}
			if entry.IsDir() {
				return nil
			}
			relative, err := filepath.Rel(repo, path)
			if err != nil {
				return err
			}
			directoryPaths = append(directoryPaths, relative)
			return nil
		})
		if err != nil {
			return nil, err
		}
		sort.Slice(directoryPaths, func(i, j int) bool {
			return filepath.ToSlash(directoryPaths[i]) < filepath.ToSlash(directoryPaths[j])
		})
		paths = append(paths, directoryPaths...)
	}
	if len(paths) > maxLogFiles {
		return nil, fmt.Errorf("log sources exceed %d-file limit", maxLogFiles)
	}
	documents := make([]logDocument, 0, len(paths))
	for index, path := range paths {
		if index > 0 && path == logFile {
			return nil, fmt.Errorf("--log-dir overlaps discovered log file: %s", logFile)
		}
		current, err := safefile.Read(repo, filepath.Join(repo, path), 8<<20)
		if err != nil {
			return nil, err
		}
		documents = append(documents, logDocument{path: filepath.ToSlash(path), headings: logHeadings(current.Data)})
	}
	return documents, nil
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

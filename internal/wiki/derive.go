package wiki

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
	"github.com/ozgurcd/achta/internal/workspace"
)

const DeriveSchema = "achta.wiki-derive.v1"

var (
	beginDerivedPattern   = regexp.MustCompile(`^<!-- BEGIN DERIVED: ([A-Za-z0-9._-]+) -->$`)
	canonicalCountPattern = regexp.MustCompile(`^[[:space:]]*(?:const[[:space:]]+)?(Canonical[A-Za-z]*Count)[[:space:]]*=[[:space:]]*(.+?)[[:space:]]*$`)
	portPattern           = regexp.MustCompile(`^[[:space:]]*-[[:space:]]*"?(?:127\.0\.0\.1:)?([0-9]{4,5}:[0-9]{4,5})"?[[:space:]]*$`)
)

type DerivePage struct {
	Repository string `json:"repository"`
	Page       string `json:"page"`
	Status     string `json:"status"`
}

type DeriveResult struct {
	SchemaVersion string       `json:"schema_version"`
	Status        string       `json:"status"`
	Scanned       int          `json:"scanned"`
	Changed       int          `json:"changed"`
	Pages         []DerivePage `json:"pages"`
}

// Derive rewrites or checks every opted-in generated block beneath wiki/.
// Each page is independently validated and atomically replaced.
func Derive(root string, check bool) (DeriveResult, error) {
	result := DeriveResult{SchemaVersion: DeriveSchema, Status: "unchanged", Pages: []DerivePage{}}
	wikiRoot, err := (workspace.Workspace{Root: root}).Confine("wiki")
	if err != nil {
		return result, fmt.Errorf("confine wiki: %w", err)
	}
	var pages []string
	err = filepath.WalkDir(wikiRoot, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Type()&os.ModeSymlink != 0 {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".md" {
			pages = append(pages, path)
		}
		return nil
	})
	if err != nil {
		return result, fmt.Errorf("scan wiki pages: %w", err)
	}
	sort.Strings(pages)
	for _, path := range pages {
		candidate, err := readOptionalRegular(path, MaxPageSize)
		if err != nil {
			return result, fmt.Errorf("read %s: %w", filepath.Base(path), err)
		}
		repository, marked, err := derivedRepository(candidate)
		if err != nil {
			return result, fmt.Errorf("parse %s: %w", filepath.Base(path), err)
		}
		if !marked {
			continue
		}
		snapshot, err := safefile.Read(root, path, MaxPageSize)
		if err != nil {
			return result, fmt.Errorf("read marked page %s: %w", filepath.Base(path), err)
		}
		block, err := DerivedBlock(root, repository)
		if err != nil {
			return result, fmt.Errorf("derive %s: %w", repository, err)
		}
		rendered, err := spliceDerived(snapshot.Data, repository, block)
		if err != nil {
			return result, fmt.Errorf("splice %s: %w", repository, err)
		}
		renderedRepository, marked, err := derivedRepository(rendered)
		if err != nil {
			return result, fmt.Errorf("validate rendered %s: %w", repository, err)
		}
		if !marked || renderedRepository != repository {
			return result, fmt.Errorf("validate rendered %s: repository=%q marked=%t", repository, renderedRepository, marked)
		}
		page := DerivePage{Repository: repository, Page: filepath.ToSlash(strings.TrimPrefix(path, root+string(filepath.Separator))), Status: "unchanged"}
		result.Scanned++
		if !bytes.Equal(snapshot.Data, rendered) {
			result.Changed++
			if check {
				page.Status = "stale"
				result.Status = "would_change"
			} else {
				if err := snapshot.Replace(rendered, MaxPageSize); err != nil {
					return result, fmt.Errorf("replace %s: %w", page.Page, err)
				}
				page.Status = "updated"
				result.Status = "updated"
			}
		}
		result.Pages = append(result.Pages, page)
	}
	return result, nil
}

// DerivedBlock returns the deterministic body between a repository's markers.
func DerivedBlock(root, repository string) ([]byte, error) {
	if !repoPattern.MatchString(repository) {
		return nil, errors.New("invalid repository name")
	}
	repo, err := (workspace.Workspace{Root: root}).Confine(repository)
	if err != nil {
		return nil, fmt.Errorf("confine repository: %w", err)
	}
	head, err := gitstate.Head(repo)
	if err != nil {
		return nil, fmt.Errorf("repository HEAD: %w", err)
	}
	branch, err := gitstate.Branch(repo)
	if err != nil {
		return nil, fmt.Errorf("repository branch: %w", err)
	}
	clean, err := gitstate.IsClean(repo)
	if err != nil {
		return nil, fmt.Errorf("repository status: %w", err)
	}
	goVersion, err := goVersionFact(root, repository)
	if err != nil {
		return nil, err
	}
	migrations, err := migrationsFact(root, repository)
	if err != nil {
		return nil, err
	}
	seams, err := seamsFact(root, repository)
	if err != nil {
		return nil, err
	}
	pins, err := pinsFact(root, repository)
	if err != nil {
		return nil, err
	}
	ports, err := portsFact(root, repository)
	if err != nil {
		return nil, err
	}
	worktree := "**clean** — HEAD and the tree agree"
	if !clean {
		worktree = "**DIRTY** — the tree does not match HEAD. Every row below is read off the working tree, so it may describe uncommitted work rather than the state at the HEAD above. The number of changed files is deliberately not pinned here: it moves with every edit and with each machine's global gitignore, so pinning it made this block stale on ordinary activity. Run `git status` for the count."
	}
	var output strings.Builder
	output.WriteString("Generated by `wiki/tools/wiki-derive.sh` — **do not edit inside this block by hand**; your edit will be overwritten on the next run. Every line below is read off the tree, so it states what the code says, not what anyone decided. Facts that were *decided* rather than derived (why a port was chosen, what a seam is for, which decision is deferred) belong in prose outside this block. If a line here looks wrong, the code moved — re-run the script; do not retype the value.\n\n")
	output.WriteString("| Derived fact | Value |\n| --- | --- |\n")
	fmt.Fprintf(&output, "| Repo HEAD | %s (%s) |\n", head[:7], branch)
	fmt.Fprintf(&output, "| Working tree vs HEAD | %s |\n", worktree)
	fmt.Fprintf(&output, "| Go version (`go.mod`) | %s |\n", goVersion)
	fmt.Fprintf(&output, "| SQL migrations | %s |\n", migrations)
	fmt.Fprintf(&output, "| Public `pkg/` seams | %s |\n", seams)
	fmt.Fprintf(&output, "| Pinned canonical counts | %s |\n", pins)
	output.WriteString("\nPublished host ports:\n\n")
	output.WriteString(ports)
	if int64(output.Len()) > MaxPageSize {
		return nil, errors.New("derived block exceeds page size limit")
	}
	return []byte(output.String()), nil
}

func derivedRepository(data []byte) (string, bool, error) {
	normalized := bytes.ReplaceAll(data, []byte("\r\n"), []byte("\n"))
	found := ""
	endCount := 0
	startIndex, endIndex := -1, -1
	for index, line := range strings.Split(string(normalized), "\n") {
		if match := beginDerivedPattern.FindStringSubmatch(line); match != nil {
			if found != "" {
				return "", false, errors.New("multiple derived block starts")
			}
			found = match[1]
			startIndex = index
		}
		if line == "<!-- END DERIVED -->" {
			endCount++
			if endIndex < 0 {
				endIndex = index
			}
		}
	}
	if found == "" {
		if endCount != 0 {
			return "", false, errors.New("derived block end has no start")
		}
		return "", false, nil
	}
	if endCount != 1 {
		return "", false, errors.New("derived block must have exactly one end marker")
	}
	if endIndex <= startIndex {
		return "", false, errors.New("derived block end must follow its start")
	}
	return found, true, nil
}

func spliceDerived(source []byte, repository string, block []byte) ([]byte, error) {
	newline := []byte("\n")
	if bytes.Contains(source, []byte("\r\n")) {
		newline = []byte("\r\n")
	}
	normalized := bytes.ReplaceAll(source, []byte("\r\n"), []byte("\n"))
	lines := strings.Split(string(normalized), "\n")
	begin := "<!-- BEGIN DERIVED: " + repository + " -->"
	start, end := -1, -1
	for i, line := range lines {
		if line == begin {
			start = i
		}
		if start >= 0 && line == "<!-- END DERIVED -->" {
			end = i
			break
		}
	}
	if start < 0 || end <= start {
		return nil, errors.New("missing or mismatched derived markers")
	}
	body := strings.Split(strings.TrimSuffix(string(block), "\n"), "\n")
	replacement := append(append(append([]string{}, lines[:start+1]...), body...), lines[end:]...)
	rendered := []byte(strings.Join(replacement, "\n"))
	if bytes.Equal(newline, []byte("\r\n")) {
		rendered = bytes.ReplaceAll(rendered, []byte("\n"), newline)
	}
	return rendered, nil
}

func goVersionFact(root, repository string) (string, error) {
	path := filepath.Join(root, repository, "go.mod")
	data, err := readOptionalRegular(path, 1<<20)
	if errors.Is(err, os.ErrNotExist) {
		return "n/a (no go.mod)", nil
	}
	if err != nil {
		return "", fmt.Errorf("read go.mod: %w", err)
	}
	for _, line := range strings.Split(string(data), "\n") {
		if fields := strings.Fields(line); len(fields) == 2 && fields[0] == "go" {
			return fields[1], nil
		}
	}
	return "unreadable", nil
}

func migrationsFact(root, repository string) (string, error) {
	repo := filepath.Join(root, repository)
	for _, relative := range []string{"migrations", filepath.Join("db", "migrations")} {
		directory := filepath.Join(repo, relative)
		entries, err := os.ReadDir(directory)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return "", err
		}
		var files []string
		for _, entry := range entries {
			if !entry.IsDir() && filepath.Ext(entry.Name()) == ".sql" && entry.Type().IsRegular() {
				files = append(files, entry.Name())
			}
		}
		if len(files) == 0 {
			continue
		}
		sort.Strings(files)
		tracked, err := gitstate.TrackedPaths(repo, filepath.ToSlash(filepath.Join(relative, "*.sql")))
		if err != nil {
			return "", err
		}
		if len(tracked) == len(files) {
			return fmt.Sprintf("head `%s` — %d files in `%s/`, all committed", files[len(files)-1], len(files), filepath.ToSlash(relative)), nil
		}
		return fmt.Sprintf("head `%s` — %d files in `%s/` but **only %d committed**. A fresh clone gets %d. The head above exists on this machine only.", files[len(files)-1], len(files), filepath.ToSlash(relative), len(tracked), len(tracked)), nil
	}
	return "n/a (no SQL migrations)", nil
}

func seamsFact(root, repository string) (string, error) {
	directory := filepath.Join(root, repository, "pkg")
	entries, err := os.ReadDir(directory)
	if errors.Is(err, os.ErrNotExist) {
		return "none — this repo publishes no `pkg/` seam", nil
	}
	if err != nil {
		return "", err
	}
	var names []string
	for _, entry := range entries {
		if entry.IsDir() && entry.Type()&os.ModeSymlink == 0 {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)
	if len(names) == 0 {
		return "none — `pkg/` exists but holds no packages", nil
	}
	quoted := make([]string, len(names))
	for i, name := range names {
		quoted[i] = "`" + name + "`"
	}
	return fmt.Sprintf("%d package(s): %s", len(names), strings.Join(quoted, " ")), nil
}

func pinsFact(root, repository string) (string, error) {
	repo := filepath.Join(root, repository)
	var pins []string
	err := filepath.WalkDir(repo, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if entry.Name() == ".git" || entry.Name() == ".gograph" || entry.Name() == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 || filepath.Ext(entry.Name()) != ".go" || len(pins) >= 10 {
			return nil
		}
		data, err := readOptionalRegular(path, 2<<20)
		if err != nil {
			return err
		}
		for number, line := range strings.Split(string(data), "\n") {
			if match := canonicalCountPattern.FindStringSubmatch(line); match != nil {
				relative, _ := filepath.Rel(repo, path)
				pins = append(pins, fmt.Sprintf("`%s:%d → %s = %s`", filepath.ToSlash(relative), number+1, match[1], match[2]))
				if len(pins) >= 10 {
					break
				}
			}
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	sort.Slice(pins, func(i, j int) bool {
		iEndpoint := strings.Contains(pins[i], "CanonicalEndpointCount")
		jEndpoint := strings.Contains(pins[j], "CanonicalEndpointCount")
		if iEndpoint != jEndpoint {
			return iEndpoint
		}
		return pins[i] < pins[j]
	})
	if len(pins) == 0 {
		return "none pinned", nil
	}
	return strings.Join(pins, "; "), nil
}

func portsFact(root, repository string) (string, error) {
	directory := filepath.Join(root, repository, "deployment")
	entries, err := os.ReadDir(directory)
	if errors.Is(err, os.ErrNotExist) {
		return "  - no `deployment/` directory\n", nil
	}
	if err != nil {
		return "", err
	}
	var names []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasPrefix(entry.Name(), "docker-compose") && strings.HasSuffix(entry.Name(), ".yml") && entry.Type().IsRegular() {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)
	if len(names) == 0 {
		return "  - no `docker-compose*.yml` under `deployment/`\n", nil
	}
	var output strings.Builder
	for _, name := range names {
		data, err := readOptionalRegular(filepath.Join(directory, name), 2<<20)
		if err != nil {
			return "", err
		}
		var ports []string
		for _, line := range strings.Split(string(data), "\n") {
			if match := portPattern.FindStringSubmatch(line); match != nil {
				ports = append(ports, match[1])
			}
		}
		if len(ports) == 0 {
			fmt.Fprintf(&output, "  - `%s` publishes **no host port**\n", name)
		} else {
			fmt.Fprintf(&output, "  - `%s` publishes %s\n", name, strings.Join(ports, " "))
		}
	}
	return output.String(), nil
}

func readOptionalRegular(path string, limit int64) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("%s is not a regular file", path)
	}
	if info.Size() > limit {
		return nil, fmt.Errorf("%s exceeds %d-byte limit", path, limit)
	}
	return os.ReadFile(path)
}

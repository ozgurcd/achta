package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"github.com/ozgurcd/achta/internal/declaredroute"
	"github.com/ozgurcd/achta/internal/safefile"
	"github.com/ozgurcd/achta/internal/workspace"
)

func runDeclaredRoute(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	rest, err := requireSubcommand(args, "check")
	if err != nil {
		return renderError(stdout, stderr, opts.json, declaredroute.Schema, err)
	}
	set := flagSet("declared-route check")
	var files, directories, routePatterns, banPatterns repeatedValue
	set.Var(&files, "file", "repeatable workflow YAML path (inside the workspace)")
	set.Var(&directories, "dir", "repeatable directory of direct workflow YAML files (inside the workspace)")
	set.Var(&routePatterns, "route-pattern", "repeatable caller regexp for an allowed route alternative")
	set.Var(&banPatterns, "ban-pattern", "repeatable caller regexp forbidden in live YAML scalar text")
	requiredRoutePattern := set.String("required-route-pattern", "", "configured route alternative that requires the scoped key")
	requiredKey := set.String("required-key", "", "caller-owned mapping key required exactly once")
	requiredScope := set.String("required-scope", "", "JSON Pointer to the YAML mapping containing the required key")
	routeCardinality := set.String("route-cardinality", declaredroute.CardinalityPerFileOne, "route cardinality: per-file-one or per-file-any")
	jsonMode := set.Bool("json", opts.json, "emit JSON")
	if err := parseFlags(set, rest); err != nil {
		opts.json = *jsonMode
		return renderError(stdout, stderr, opts.json, declaredroute.Schema, err)
	}
	opts.json = *jsonMode
	if len(files) > declaredroute.MaxDocuments {
		return renderError(stdout, stderr, opts.json, declaredroute.Schema, invalid("workflow count exceeds %d", declaredroute.MaxDocuments))
	}
	ws, err := resolveWorkspace(opts)
	if err != nil {
		return renderError(stdout, stderr, opts.json, declaredroute.Schema, err)
	}
	documents := make([]declaredroute.Document, 0, len(files))
	seen := map[string]bool{}
	for _, value := range files {
		path, pathErr := confinedPath(ws, value)
		if pathErr != nil {
			return renderError(stdout, stderr, opts.json, declaredroute.Schema, pathErr)
		}
		if seen[path] {
			return renderError(stdout, stderr, opts.json, declaredroute.Schema, invalid("workflow %q is repeated", value))
		}
		seen[path] = true
		snapshot, readErr := safefile.Read(ws.Root, path, declaredroute.MaxDocument)
		if readErr != nil {
			return renderError(stdout, stderr, opts.json, declaredroute.Schema, invalid("read workflow %q: %v", value, readErr))
		}
		documents = append(documents, declaredroute.Document{Path: filepath.ToSlash(value), Data: snapshot.Data})
	}
	for _, value := range directories {
		input, readErr := readDeclaredRouteDirectory(ws, value, seen)
		if readErr != nil {
			return renderError(stdout, stderr, opts.json, declaredroute.Schema, invalid("read workflow directory %q: %v", value, readErr))
		}
		documents = append(documents, input...)
		if len(documents) > declaredroute.MaxDocuments {
			return renderError(stdout, stderr, opts.json, declaredroute.Schema, invalid("workflow count exceeds %d", declaredroute.MaxDocuments))
		}
	}
	result, err := declaredroute.Check(documents, declaredroute.Options{
		RoutePatterns:        routePatterns,
		BanPatterns:          banPatterns,
		RequiredRoutePattern: *requiredRoutePattern,
		RequiredKey:          *requiredKey,
		RequiredScope:        *requiredScope,
		RouteCardinality:     *routeCardinality,
	})
	if err != nil {
		return renderError(stdout, stderr, opts.json, declaredroute.Schema, invalid("declared-route check: %v", err))
	}
	code := 0
	if result.Status != "pass" {
		code = 1
	}
	if opts.json {
		if writeJSON(stdout, stderr, result) != 0 {
			return 2
		}
		return code
	}
	for _, violation := range result.Violations {
		fmt.Fprintf(stdout, "declared-route check: %s:%d %s expected=%d observed=%d — %s\n", violation.File, violation.Line, violation.Rule, violation.Expected, violation.Observed, violation.Text)
	}
	fmt.Fprintf(stdout, "declared-route check: %s; %d workflow(s), %d violation(s)\n", result.Status, len(result.Documents), len(result.Violations))
	for _, refused := range result.Refused {
		fmt.Fprintf(stdout, "declared-route check: refused — %s\n", refused)
	}
	return code
}

func readDeclaredRouteDirectory(ws workspace.Workspace, value string, seen map[string]bool) ([]declaredroute.Document, error) {
	directory, err := confinedPath(ws, value)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(directory)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%q is not a directory", value)
	}
	entries, err := f.ReadDir(declaredroute.MaxDirectoryEntries + 1)
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, err
	}
	if len(entries) > declaredroute.MaxDirectoryEntries {
		return nil, fmt.Errorf("directory exceeds %d entries", declaredroute.MaxDirectoryEntries)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	documents := make([]declaredroute.Document, 0)
	for _, entry := range entries {
		extension := filepath.Ext(entry.Name())
		if extension != ".yml" && extension != ".yaml" {
			continue
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("workflow %q is linked", entry.Name())
		}
		entryInfo, infoErr := entry.Info()
		if infoErr != nil {
			return nil, fmt.Errorf("inspect workflow %q: %w", entry.Name(), infoErr)
		}
		if !entryInfo.Mode().IsRegular() {
			return nil, fmt.Errorf("workflow %q is not a regular file", entry.Name())
		}
		path := filepath.Join(directory, entry.Name())
		if seen[path] {
			return nil, fmt.Errorf("workflow %q is repeated", entry.Name())
		}
		snapshot, readErr := safefile.Read(ws.Root, path, declaredroute.MaxDocument)
		if readErr != nil {
			return nil, fmt.Errorf("read workflow %q: %w", entry.Name(), readErr)
		}
		seen[path] = true
		documents = append(documents, declaredroute.Document{Path: filepath.ToSlash(filepath.Join(value, entry.Name())), Data: snapshot.Data})
	}
	if len(documents) == 0 {
		return nil, errors.New("directory contains no direct workflow YAML files")
	}
	return documents, nil
}

// SPDX-License-Identifier: MIT
package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"github.com/ozgurcd/achta/internal/countcheck"
	"github.com/ozgurcd/achta/internal/safefile"
	"github.com/ozgurcd/achta/internal/workspace"
)

func runCount(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	rest, err := requireSubcommand(args, "check")
	if err != nil {
		return renderError(stdout, stderr, opts.json, countcheck.Schema, err)
	}
	set := flagSet("count check")
	var files repeatedValue
	var directories repeatedValue
	var aliases repeatedValue
	var claimPatterns repeatedValue
	var claimTargetPatterns repeatedValue
	var claimAssertionPatterns repeatedValue
	var proofClaimPatterns repeatedValue
	var proofPatterns repeatedValue
	var exemptPatterns repeatedValue
	set.Var(&files, "file", "repeatable Markdown path (inside the workspace)")
	set.Var(&directories, "dir", "repeatable directory of direct Markdown files (inside the workspace)")
	totalPattern := set.String("total-pattern", "", "caller regexp with named count capture for a stated total")
	partPattern := set.String("part-pattern", "", "caller regexp with named count capture for breakdown parts")
	set.Var(&claimPatterns, "claim-pattern", "repeatable caller regexp with named count capture for a citation-backed claim")
	set.Var(&claimTargetPatterns, "claim-target-pattern", "repeatable caller regexp with named count capture for a counted target")
	set.Var(&claimAssertionPatterns, "claim-assertion-pattern", "repeatable caller regexp classifying an assertion in the same paragraph as a counted target")
	citationPattern := set.String("citation-pattern", "", "caller regexp whose full matches are distinct citations")
	set.Var(&proofClaimPatterns, "proof-claim-pattern", "repeatable caller regexp with named count capture for a proof-backed claim")
	set.Var(&proofPatterns, "proof-pattern", "repeatable caller regexp with named count capture for a proof count")
	proofWithinLines := set.Int("proof-within-lines", 0, "maximum line distance between a proof-backed claim and its nearest proof")
	claimSectionPattern := set.String("claim-section-pattern", "", "caller regexp selecting the last matching section for citation claims")
	set.Var(&exemptPatterns, "exempt-pattern", "repeatable caller regexp whose match exempts the next non-blank paragraph")
	set.Var(&aliases, "count-alias", "repeatable exact TOKEN=N alias for a non-decimal count")
	jsonMode := set.Bool("json", opts.json, "emit JSON")
	if err := parseFlags(set, rest); err != nil {
		opts.json = *jsonMode
		return renderError(stdout, stderr, opts.json, countcheck.Schema, err)
	}
	opts.json = *jsonMode
	if len(files) == 0 && len(directories) == 0 {
		return renderError(stdout, stderr, opts.json, countcheck.Schema, invalid("at least one --file or --dir is required"))
	}
	ws, err := resolveWorkspace(opts)
	if err != nil {
		return renderError(stdout, stderr, opts.json, countcheck.Schema, err)
	}
	documents := make([]countcheck.Document, 0, len(files))
	seen := map[string]bool{}
	for _, value := range files {
		path, pathErr := confinedPath(ws, value)
		if pathErr != nil {
			return renderError(stdout, stderr, opts.json, countcheck.Schema, pathErr)
		}
		snapshot, readErr := safefile.Read(ws.Root, path, countcheck.MaxDocument)
		if readErr != nil {
			return renderError(stdout, stderr, opts.json, countcheck.Schema, invalid("read document %q: %v", value, readErr))
		}
		if seen[path] {
			return renderError(stdout, stderr, opts.json, countcheck.Schema, invalid("document %q is repeated", value))
		}
		seen[path] = true
		documents = append(documents, countcheck.Document{Path: filepath.ToSlash(value), Data: snapshot.Data})
	}
	for _, value := range directories {
		input, readErr := readCountDirectory(ws, value, seen)
		if readErr != nil {
			return renderError(stdout, stderr, opts.json, countcheck.Schema, invalid("read document directory %q: %v", value, readErr))
		}
		documents = append(documents, input...)
		if len(documents) > countcheck.MaxDocuments {
			return renderError(stdout, stderr, opts.json, countcheck.Schema, invalid("document count exceeds %d", countcheck.MaxDocuments))
		}
	}
	result, err := countcheck.Check(documents, countcheck.Options{
		TotalPattern:           *totalPattern,
		PartPattern:            *partPattern,
		ClaimPatterns:          claimPatterns,
		ClaimTargetPatterns:    claimTargetPatterns,
		ClaimAssertionPatterns: claimAssertionPatterns,
		CitationPattern:        *citationPattern,
		ProofClaimPatterns:     proofClaimPatterns,
		ProofPatterns:          proofPatterns,
		ProofWithinLines:       *proofWithinLines,
		ClaimSectionPattern:    *claimSectionPattern,
		ExemptPatterns:         exemptPatterns,
		CountAliases:           aliases,
	})
	if err != nil {
		return renderError(stdout, stderr, opts.json, countcheck.Schema, invalid("count check: %v", err))
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
		fmt.Fprintf(stdout, "count check: %s:%d %s claimed=%d observed=%d — %s\n", violation.File, violation.Line, violation.Rule, violation.Claimed, violation.Observed, violation.Text)
	}
	if len(proofPatterns) > 0 {
		fmt.Fprintf(stdout, "count check: %s; %d file(s), %d breakdown(s), %d citation claim(s), %d proof claim(s), %d violation(s)\n", result.Status, result.Files, result.Breakdowns, result.Claims, result.ProofClaims, len(result.Violations))
	} else {
		fmt.Fprintf(stdout, "count check: %s; %d file(s), %d breakdown(s), %d claim(s), %d violation(s)\n", result.Status, result.Files, result.Breakdowns, result.Claims, len(result.Violations))
	}
	for _, refused := range result.Refused {
		fmt.Fprintf(stdout, "count check: refused — %s\n", refused)
	}
	return code
}

func readCountDirectory(ws workspace.Workspace, value string, seen map[string]bool) ([]countcheck.Document, error) {
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
	entries, err := f.ReadDir(countcheck.MaxDirectoryEntries + 1)
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, err
	}
	if len(entries) > countcheck.MaxDirectoryEntries {
		return nil, fmt.Errorf("directory exceeds %d entries", countcheck.MaxDirectoryEntries)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	documents := make([]countcheck.Document, 0)
	for _, entry := range entries {
		extension := filepath.Ext(entry.Name())
		if extension != ".md" && extension != ".MD" {
			continue
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("document %q is linked", entry.Name())
		}
		entryInfo, infoErr := entry.Info()
		if infoErr != nil {
			return nil, fmt.Errorf("inspect document %q: %w", entry.Name(), infoErr)
		}
		if !entryInfo.Mode().IsRegular() {
			return nil, fmt.Errorf("document %q is not a regular file", entry.Name())
		}
		path := filepath.Join(directory, entry.Name())
		if seen[path] {
			return nil, fmt.Errorf("document %q is repeated", entry.Name())
		}
		snapshot, readErr := safefile.Read(ws.Root, path, countcheck.MaxDocument)
		if readErr != nil {
			return nil, fmt.Errorf("read document %q: %w", entry.Name(), readErr)
		}
		seen[path] = true
		documents = append(documents, countcheck.Document{Path: filepath.ToSlash(filepath.Join(value, entry.Name())), Data: snapshot.Data})
	}
	if len(documents) == 0 {
		return nil, errors.New("directory contains no direct Markdown files")
	}
	return documents, nil
}

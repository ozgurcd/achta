package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/ozgurcd/achta/internal/census"
	"github.com/ozgurcd/achta/internal/safefile"
)

func runLedger(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	rest, err := requireSubcommand(args, "census")
	if err != nil {
		return renderError(stdout, stderr, opts.json, census.Schema, err)
	}
	set := flagSet("ledger census")
	fileValue := set.String("file", "", "markdown ledger (inside the workspace)")
	var dirs repeatedValue
	var trees repeatedValue
	set.Var(&dirs, "dir", "repeatable [LABEL=]PATH: every regular file directly under PATH must have a row; size = line count")
	set.Var(&trees, "tree", "repeatable [LABEL=]PATH: every subdirectory of PATH must have a row; size = summed lines of --ext files under it")
	ext := set.String("ext", "", "file extension counted under --tree sources (required with --tree)")
	jsonMode := set.Bool("json", opts.json, "emit JSON")
	if err := parseFlags(set, rest); err != nil {
		opts.json = *jsonMode
		return renderError(stdout, stderr, opts.json, census.Schema, err)
	}
	opts.json = *jsonMode
	if *fileValue == "" {
		return renderError(stdout, stderr, opts.json, census.Schema, invalid("--file is required"))
	}
	if len(dirs)+len(trees) == 0 {
		return renderError(stdout, stderr, opts.json, census.Schema, invalid("at least one --dir or --tree is required"))
	}
	if len(trees) > 0 && *ext == "" {
		return renderError(stdout, stderr, opts.json, census.Schema, invalid("--tree requires --ext"))
	}
	ws, err := resolveWorkspace(opts)
	if err != nil {
		return renderError(stdout, stderr, opts.json, census.Schema, err)
	}
	file, err := confinedPath(ws, *fileValue)
	if err != nil {
		return renderError(stdout, stderr, opts.json, census.Schema, err)
	}
	snapshot, err := safefile.Read(ws.Root, file, census.MaxLedger)
	if err != nil {
		return renderError(stdout, stderr, opts.json, census.Schema, invalid("read ledger: %v", err))
	}
	var sources []census.Source
	for _, spec := range dirs {
		label, path := splitLabel(spec)
		confined, err := confinedPath(ws, path)
		if err != nil {
			return renderError(stdout, stderr, opts.json, census.Schema, err)
		}
		sources = append(sources, census.Source{Label: label, Path: confined, Kind: "files"})
	}
	for _, spec := range trees {
		label, path := splitLabel(spec)
		confined, err := confinedPath(ws, path)
		if err != nil {
			return renderError(stdout, stderr, opts.json, census.Schema, err)
		}
		sources = append(sources, census.Source{Label: label, Path: confined, Kind: "dirs", Ext: *ext})
	}
	result, err := census.Check(snapshot.Data, sources)
	if err != nil {
		return renderError(stdout, stderr, opts.json, census.Schema, invalid("ledger census: %v", err))
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
	for _, v := range result.Violations {
		where := "census"
		if v.Line > 0 {
			where = fmt.Sprintf("line %d", v.Line)
		}
		fmt.Fprintf(stdout, "ledger census: %s %s — %s\n", where, v.Rule, v.Text)
	}
	fmt.Fprintf(stdout, "ledger census: %s; %d row(s), %d source(s), %d violation(s)\n", result.Status, result.Rows, len(result.Sources), len(result.Violations))
	return code
}

func splitLabel(spec string) (string, string) {
	if i := strings.Index(spec, "="); i > 0 && !strings.Contains(spec[:i], "/") {
		return spec[:i], spec[i+1:]
	}
	return "", spec
}

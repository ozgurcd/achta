package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/ozgurcd/achta/internal/census"
	"github.com/ozgurcd/achta/internal/ledgerrows"
	"github.com/ozgurcd/achta/internal/safefile"
)

func runLedger(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	if len(args) == 0 {
		return renderError(stdout, stderr, opts.json, "achta.error.v1", invalid("expected subcommand census or rows"))
	}
	switch args[0] {
	case "census":
		return runLedgerCensus(args[1:], stdout, stderr, opts)
	case "rows":
		return runLedgerRows(args[1:], stdout, stderr, opts)
	default:
		return renderError(stdout, stderr, opts.json, "achta.error.v1", invalid("expected subcommand census or rows"))
	}
}

func runLedgerCensus(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	set := flagSet("ledger census")
	fileValue := set.String("file", "", "markdown ledger (inside the workspace)")
	var dirs repeatedValue
	var trees repeatedValue
	set.Var(&dirs, "dir", "repeatable [LABEL=]PATH: every regular file directly under PATH must have a row; size = line count")
	set.Var(&trees, "tree", "repeatable [LABEL=]PATH: every subdirectory of PATH must have a row; size = summed lines of --ext files under it")
	ext := set.String("ext", "", "file extension counted under --tree sources (required with --tree)")
	jsonMode := set.Bool("json", opts.json, "emit JSON")
	if err := parseFlags(set, args); err != nil {
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

func runLedgerRows(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	set := flagSet("ledger rows")
	fileValue := set.String("file", "", "markdown ledger (inside the workspace)")
	idCell := set.Int("id-cell", 0, "one-based cell containing the state marker and row identity")
	proseCell := set.Int("prose-cell", 0, "one-based cell containing row prose")
	openMarker := set.String("open-marker", "", "exact prefix marking an open ID cell")
	closedMarker := set.String("closed-marker", "", "exact prefix marking a closed ID cell")
	identityEnd := set.String("identity-end-marker", "", "exact suffix ending the row identity")
	quoteMarker := set.String("quote-marker", "", "exact marker introducing a double-quoted closure string")
	var completionMarkers repeatedValue
	var exemptMarkers repeatedValue
	var conditionMarkers repeatedValue
	set.Var(&completionMarkers, "completion-marker", "repeatable exact prose marker that claims completion")
	set.Var(&exemptMarkers, "exempt-marker", "repeatable exact prose marker exempting an open completion claim")
	set.Var(&conditionMarkers, "condition-marker", "repeatable exact prose marker that states a close condition")
	jsonMode := set.Bool("json", opts.json, "emit JSON")
	if err := parseFlags(set, args); err != nil {
		opts.json = *jsonMode
		return renderError(stdout, stderr, opts.json, ledgerrows.Schema, err)
	}
	opts.json = *jsonMode
	if *fileValue == "" {
		return renderError(stdout, stderr, opts.json, ledgerrows.Schema, invalid("--file is required"))
	}
	if *idCell < 1 || *proseCell < 1 {
		return renderError(stdout, stderr, opts.json, ledgerrows.Schema, invalid("--id-cell and --prose-cell are required positive indexes"))
	}
	if *openMarker == "" || *closedMarker == "" || *identityEnd == "" || *quoteMarker == "" {
		return renderError(stdout, stderr, opts.json, ledgerrows.Schema, invalid("--open-marker, --closed-marker, --identity-end-marker, and --quote-marker are required"))
	}
	if len(completionMarkers) == 0 || len(conditionMarkers) == 0 {
		return renderError(stdout, stderr, opts.json, ledgerrows.Schema, invalid("at least one --completion-marker and --condition-marker are required"))
	}
	ws, err := resolveWorkspace(opts)
	if err != nil {
		return renderError(stdout, stderr, opts.json, ledgerrows.Schema, err)
	}
	file, err := confinedPath(ws, *fileValue)
	if err != nil {
		return renderError(stdout, stderr, opts.json, ledgerrows.Schema, err)
	}
	snapshot, err := safefile.Read(ws.Root, file, ledgerrows.MaxLedger)
	if err != nil {
		return renderError(stdout, stderr, opts.json, ledgerrows.Schema, invalid("read ledger: %v", err))
	}
	result, err := ledgerrows.Check(snapshot.Data, ledgerrows.Options{
		File:              *fileValue,
		IDCell:            *idCell,
		ProseCell:         *proseCell,
		OpenMarker:        *openMarker,
		ClosedMarker:      *closedMarker,
		IdentityEndMarker: *identityEnd,
		CompletionMarkers: completionMarkers,
		ExemptMarkers:     exemptMarkers,
		ConditionMarkers:  conditionMarkers,
		QuoteMarker:       *quoteMarker,
	})
	if err != nil {
		return renderError(stdout, stderr, opts.json, ledgerrows.Schema, invalid("ledger rows: %v", err))
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
		fmt.Fprintf(stdout, "ledger rows: line %d %s %s — %s\n", violation.Line, violation.Rule, violation.Identity, violation.Text)
	}
	fmt.Fprintf(stdout, "ledger rows: %s; %d row(s), %d open, %d closed, %d violation(s)\n", result.Status, result.Rows, result.OpenRows, result.ClosedRows, len(result.Violations))
	for _, refused := range result.Refused {
		fmt.Fprintf(stdout, "ledger rows: refused — %s\n", refused)
	}
	return code
}

func splitLabel(spec string) (string, string) {
	if i := strings.Index(spec, "="); i > 0 && !strings.Contains(spec[:i], "/") {
		return spec[:i], spec[i+1:]
	}
	return "", spec
}

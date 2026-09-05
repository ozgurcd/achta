package cli

import (
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"slices"
	"strings"

	achtawiki "github.com/ozgurcd/achta/internal/wiki"
)

const wikiCheckSchema = "achta.wiki-check.v1"

// wikiCheckNames is the canonical evaluation order of `wiki check`. `--only`
// selects a subset of it; the selection is a set, never a new exit code.
var wikiCheckNames = []string{"freshness", "derive"}

type wikiDerivedPrint struct {
	SchemaVersion string `json:"schema_version"`
	Status        string `json:"status"`
	Repository    string `json:"repository"`
	Block         string `json:"block"`
}

type wikiCheckItem struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Detail any    `json:"detail"`
}

type wikiCheckDocument struct {
	SchemaVersion string          `json:"schema_version"`
	Status        string          `json:"status"`
	WikiDir       string          `json:"wiki_dir"`
	Checks        []wikiCheckItem `json:"checks"`
}

func runWikiDerive(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	set := flagSet("wiki derive")
	check := set.Bool("check", false, "report stale blocks without writing")
	printRepository := set.String("print", "", "print one repository block")
	jsonMode := set.Bool("json", opts.json, "emit JSON")
	if err := parseFlags(set, args); err != nil {
		opts.json = *jsonMode
		return renderError(stdout, stderr, opts.json, achtawiki.DeriveSchema, err)
	}
	opts.json = *jsonMode
	if *check && *printRepository != "" {
		return renderError(stdout, stderr, opts.json, achtawiki.DeriveSchema, invalid("--check and --print are mutually exclusive"))
	}
	ws, err := resolveWorkspace(opts)
	if err != nil {
		return renderError(stdout, stderr, opts.json, achtawiki.DeriveSchema, err)
	}
	if *printRepository != "" {
		block, err := achtawiki.DerivedBlock(ws.Root, *printRepository)
		if err != nil {
			return renderError(stdout, stderr, opts.json, achtawiki.DeriveSchema, invalid("wiki derive: %v", err))
		}
		if opts.json {
			return writeJSON(stdout, stderr, wikiDerivedPrint{SchemaVersion: achtawiki.DeriveSchema, Status: "printed", Repository: *printRepository, Block: string(block)})
		}
		_, _ = stdout.Write(block)
		return 0
	}
	result, err := achtawiki.Derive(ws.Root, *check)
	if err != nil {
		return renderError(stdout, stderr, opts.json, achtawiki.DeriveSchema, invalid("wiki derive: %v", err))
	}
	code := 0
	if *check && result.Status == "would_change" {
		code = 1
	}
	if opts.json {
		if writeJSON(stdout, stderr, result) != 0 {
			return 2
		}
		return code
	}
	for _, page := range result.Pages {
		fmt.Fprintf(stdout, "%s %s %s\n", page.Status, page.Repository, page.Page)
	}
	fmt.Fprintf(stdout, "wiki derive: %d scanned, %d changed\n", result.Scanned, result.Changed)
	return code
}

func runWikiCheck(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	set := flagSet("wiki check")
	only := set.String("only", "", "evaluate only these comma-separated checks: freshness, derive")
	jsonMode := set.Bool("json", opts.json, "emit JSON")
	if err := parseFlags(set, args); err != nil {
		opts.json = *jsonMode
		return renderError(stdout, stderr, opts.json, wikiCheckSchema, err)
	}
	opts.json = *jsonMode
	selected := wikiCheckNames
	if flagGiven(set, "only") {
		parsed, err := parseWikiCheckSelection(*only)
		if err != nil {
			return renderError(stdout, stderr, opts.json, wikiCheckSchema, err)
		}
		selected = parsed
	}
	ws, err := resolveWorkspace(opts)
	if err != nil {
		return renderError(stdout, stderr, opts.json, wikiCheckSchema, err)
	}
	document := wikiCheckDocument{SchemaVersion: wikiCheckSchema, Status: "pass", WikiDir: filepath.Join(ws.Root, "wiki"), Checks: []wikiCheckItem{}}
	code := 0
	for _, name := range selected {
		switch name {
		case "freshness":
			freshness, freshnessErr := achtawiki.Freshness(ws.Root, "")
			if freshnessErr != nil {
				document.Status, code = "cannot_evaluate", 2
				document.Checks = append(document.Checks, wikiCheckItem{Name: "freshness", Status: "cannot_evaluate", Detail: boundedCLIError(freshnessErr)})
				continue
			}
			document.Checks = append(document.Checks, wikiCheckItem{Name: "freshness", Status: freshness.Status, Detail: freshness})
			if freshness.Status == "cannot_evaluate" {
				document.Status, code = "cannot_evaluate", 2
			} else if freshness.Status != "pass" {
				document.Status, code = "fail", 1
			}
		case "derive":
			derived, deriveErr := achtawiki.Derive(ws.Root, true)
			if deriveErr != nil {
				document.Status, code = "cannot_evaluate", 2
				document.Checks = append(document.Checks, wikiCheckItem{Name: "derive", Status: "cannot_evaluate", Detail: boundedCLIError(deriveErr)})
				continue
			}
			status := "pass"
			if derived.Status == "would_change" {
				status = "fail"
				if code == 0 {
					document.Status, code = "fail", 1
				}
			}
			document.Checks = append(document.Checks, wikiCheckItem{Name: "derive", Status: status, Detail: derived})
		}
	}
	if opts.json {
		if writeJSON(stdout, stderr, document) != 0 {
			return 2
		}
		return code
	}
	fmt.Fprintf(stdout, "wiki check wiki_dir: %s\n", document.WikiDir)
	for _, check := range document.Checks {
		fmt.Fprintf(stdout, "wiki check %s: %s\n", check.Name, check.Status)
	}
	return code
}

// parseWikiCheckSelection turns a --only value into the canonical-ordered set
// of checks to evaluate. It fails closed: an empty list, an empty name, an
// unknown name, or a name given twice is invalid input and nothing runs.
func parseWikiCheckSelection(only string) ([]string, error) {
	known := strings.Join(wikiCheckNames, ", ")
	seen := map[string]bool{}
	for name := range strings.SplitSeq(only, ",") {
		name = strings.TrimSpace(name)
		if name == "" {
			return nil, invalid("--only requires one or more check names from: %s", known)
		}
		if !slices.Contains(wikiCheckNames, name) {
			return nil, invalid("--only: unknown check %q; known checks are: %s", name, known)
		}
		if seen[name] {
			return nil, invalid("--only: check %q is named more than once", name)
		}
		seen[name] = true
	}
	selected := make([]string, 0, len(seen))
	for _, name := range wikiCheckNames {
		if seen[name] {
			selected = append(selected, name)
		}
	}
	return selected, nil
}

// flagGiven reports whether the caller set the named flag at all, so an
// explicitly empty value can be refused instead of read as "not selected".
func flagGiven(set *flag.FlagSet, name string) bool {
	given := false
	set.Visit(func(f *flag.Flag) {
		if f.Name == name {
			given = true
		}
	})
	return given
}

func boundedCLIError(err error) string {
	message := sanitize(err.Error())
	if len(message) > 256 {
		message = message[:256] + "..."
	}
	return message
}

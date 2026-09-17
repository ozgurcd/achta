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
	SchemaVersion string            `json:"schema_version"`
	Status        string            `json:"status"`
	WikiDir       string            `json:"wiki_dir"`
	Checks        []wikiCheckItem   `json:"checks"`
	NotJudged     map[string]string `json:"not_judged,omitempty"`
}

func runWikiDerive(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	set := flagSet("wiki derive")
	check := set.Bool("check", false, "preview stale pages without writing (the default)")
	write := set.Bool("write", false, "write generated blocks and name each updated page")
	printRepository := set.String("print", "", "print one repository block")
	jsonMode := set.Bool("json", opts.json, "emit JSON")
	if err := parseFlags(set, args); err != nil {
		opts.json = *jsonMode
		return renderError(stdout, stderr, opts.json, achtawiki.DeriveSchema, err)
	}
	opts.json = *jsonMode
	if (*check && *write) || (*printRepository != "" && (*check || *write)) {
		return renderError(stdout, stderr, opts.json, achtawiki.DeriveSchema, invalid("--check, --write and --print are mutually exclusive"))
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
	result, err := achtawiki.Derive(ws.Root, !*write)
	if err != nil {
		return renderError(stdout, stderr, opts.json, achtawiki.DeriveSchema, invalid("wiki derive: %v", err))
	}
	code := 0
	if !*write && result.Status == "would_change" {
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
		if page.Status == "stale" {
			fmt.Fprintf(stdout, "--- before %s\n%s+++ after %s\n%s", page.Page, page.Before, page.Page, page.After)
		}
	}
	fmt.Fprintf(stdout, "wiki derive: %d scanned, %d changed\n", result.Scanned, result.Changed)
	return code
}

func runWikiCheck(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	set := flagSet("wiki check")
	only := set.String("only", "", "evaluate only these comma-separated checks: freshness, derive")
	var repositories, exclusions repeatedValue
	set.Var(&repositories, "repo", "judge only named repository pages (repeatable)")
	set.Var(&exclusions, "exclude", "do not judge NAME=REASON (repeatable; reason required)")
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
	scope, err := achtawiki.SelectRepositories(ws.Root, repositories, exclusions)
	if err != nil {
		return renderError(stdout, stderr, opts.json, wikiCheckSchema, invalid("wiki check scope: %v", err))
	}
	document := wikiCheckDocument{SchemaVersion: wikiCheckSchema, Status: "pass", WikiDir: filepath.Join(ws.Root, "wiki"), Checks: []wikiCheckItem{}, NotJudged: scope}
	code := 0
	for _, name := range selected {
		switch name {
		case "freshness":
			freshness, freshnessErr := achtawiki.FreshnessSelected(ws.Root, "", scope)
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
			derived, deriveErr := achtawiki.DeriveSelected(ws.Root, true, scope)
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
	omitted := make([]string, 0, len(scope))
	for name := range scope {
		omitted = append(omitted, name)
	}
	slices.Sort(omitted)
	for _, name := range omitted {
		fmt.Fprintf(stdout, "NOT judged repository page %s: %s\n", name, scope[name])
	}
	for _, check := range document.Checks {
		fmt.Fprintf(stdout, "wiki check %s: %s\n", check.Name, check.Status)
		switch detail := check.Detail.(type) {
		case achtawiki.FreshnessResult:
			for _, page := range detail.Pages {
				if page.Status == "not_judged" {
					fmt.Fprintf(stdout, "NOT judged %s %s (%s): %s\n", page.Repository, page.Page, check.Name, page.Problem)
				}
			}
		case achtawiki.DeriveResult:
			for _, page := range detail.Pages {
				if page.Status == "not_judged" {
					fmt.Fprintf(stdout, "NOT judged %s %s (%s): %s\n", page.Repository, page.Page, check.Name, page.Reason)
				}
			}
		}
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

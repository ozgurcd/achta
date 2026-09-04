package cli

import (
	"fmt"
	"io"

	achtawiki "github.com/ozgurcd/achta/internal/wiki"
)

const wikiCheckSchema = "achta.wiki-check.v1"

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
	jsonMode := set.Bool("json", opts.json, "emit JSON")
	if err := parseFlags(set, args); err != nil {
		opts.json = *jsonMode
		return renderError(stdout, stderr, opts.json, wikiCheckSchema, err)
	}
	opts.json = *jsonMode
	ws, err := resolveWorkspace(opts)
	if err != nil {
		return renderError(stdout, stderr, opts.json, wikiCheckSchema, err)
	}
	document := wikiCheckDocument{SchemaVersion: wikiCheckSchema, Status: "pass", Checks: []wikiCheckItem{}}
	code := 0
	freshness, freshnessErr := achtawiki.Freshness(ws.Root, "")
	if freshnessErr != nil {
		document.Status, code = "cannot_evaluate", 2
		document.Checks = append(document.Checks, wikiCheckItem{Name: "freshness", Status: "cannot_evaluate", Detail: boundedCLIError(freshnessErr)})
	} else {
		document.Checks = append(document.Checks, wikiCheckItem{Name: "freshness", Status: freshness.Status, Detail: freshness})
		if freshness.Status == "cannot_evaluate" {
			document.Status, code = "cannot_evaluate", 2
		} else if freshness.Status != "pass" {
			document.Status, code = "fail", 1
		}
	}
	derived, deriveErr := achtawiki.Derive(ws.Root, true)
	if deriveErr != nil {
		document.Status, code = "cannot_evaluate", 2
		document.Checks = append(document.Checks, wikiCheckItem{Name: "derive", Status: "cannot_evaluate", Detail: boundedCLIError(deriveErr)})
	} else {
		status := "pass"
		if derived.Status == "would_change" {
			status = "fail"
			if code == 0 {
				document.Status, code = "fail", 1
			}
		}
		document.Checks = append(document.Checks, wikiCheckItem{Name: "derive", Status: status, Detail: derived})
	}
	if opts.json {
		if writeJSON(stdout, stderr, document) != 0 {
			return 2
		}
		return code
	}
	for _, check := range document.Checks {
		fmt.Fprintf(stdout, "wiki check %s: %s\n", check.Name, check.Status)
	}
	return code
}

func boundedCLIError(err error) string {
	message := sanitize(err.Error())
	if len(message) > 256 {
		message = message[:256] + "..."
	}
	return message
}

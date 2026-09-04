package cli

import (
	"fmt"
	"io"

	achtawiki "github.com/ozgurcd/achta/internal/wiki"
)

func runWikiFreshness(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	set := flagSet("wiki freshness")
	strict := set.Bool("strict", false, "exit nonzero on pin drift")
	repository := set.String("repo", "", "check one repository page")
	jsonMode := set.Bool("json", opts.json, "emit JSON")
	if err := parseFlags(set, args); err != nil {
		opts.json = *jsonMode
		return renderError(stdout, stderr, opts.json, achtawiki.FreshnessSchema, err)
	}
	opts.json = *jsonMode
	ws, err := resolveWorkspace(opts)
	if err != nil {
		return renderError(stdout, stderr, opts.json, achtawiki.FreshnessSchema, err)
	}
	result, err := achtawiki.Freshness(ws.Root, *repository)
	if err != nil {
		return renderError(stdout, stderr, opts.json, achtawiki.FreshnessSchema, invalid("wiki freshness: %v", err))
	}
	code := 0
	if result.Status == "cannot_evaluate" {
		code = 2
	} else if *strict && result.Status == "drift" {
		code = 1
	}
	if opts.json {
		if writeJSON(stdout, stderr, result) != 0 {
			return 2
		}
		return code
	}
	for _, page := range result.Pages {
		if page.Status == "fresh" && opts.quiet {
			continue
		}
		fmt.Fprintf(stdout, "%s %s", page.Status, page.Repository)
		if page.HeadSHA != "" {
			fmt.Fprintf(stdout, " @ %s", page.HeadSHA[:7])
		}
		if page.Problem != "" {
			fmt.Fprintf(stdout, " — %s", page.Problem)
		}
		fmt.Fprintln(stdout)
	}
	fmt.Fprintf(stdout, "wiki freshness: %d fresh, %d behind, %d unpinned, %d unreadable\n", result.Fresh, result.Behind, result.Unpinned, result.Unreadable)
	return code
}

func runWikiUnpushed(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	set := flagSet("wiki unpushed")
	repoValue := set.String("repo", "", "repository path")
	jsonMode := set.Bool("json", opts.json, "emit JSON")
	if err := parseFlags(set, args); err != nil {
		opts.json = *jsonMode
		return renderError(stdout, stderr, opts.json, achtawiki.UnpushedSchema, err)
	}
	opts.json = *jsonMode
	if *repoValue == "" {
		return renderError(stdout, stderr, opts.json, achtawiki.UnpushedSchema, invalid("--repo is required"))
	}
	ws, err := resolveWorkspace(opts)
	if err != nil {
		return renderError(stdout, stderr, opts.json, achtawiki.UnpushedSchema, err)
	}
	repo, err := confinedPath(ws, *repoValue)
	if err != nil {
		return renderError(stdout, stderr, opts.json, achtawiki.UnpushedSchema, err)
	}
	result, err := achtawiki.Unpushed(repo)
	if err != nil {
		return renderError(stdout, stderr, opts.json, achtawiki.UnpushedSchema, invalid("wiki unpushed: %v", err))
	}
	code := 0
	if result.Status != "published" {
		code = 1
	}
	if opts.json {
		if writeJSON(stdout, stderr, result) != 0 {
			return 2
		}
		return code
	}
	fmt.Fprintf(stdout, "wiki unpushed %s: %s; %d ahead, %d behind %s\n", result.Repository, result.Status, result.Ahead, result.Behind, result.Upstream)
	return code
}

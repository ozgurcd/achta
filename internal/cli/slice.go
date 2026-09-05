package cli

import (
	"fmt"
	"io"

	"github.com/ozgurcd/achta/internal/slicecheck"
)

func runSlice(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	rest, err := requireSubcommand(args, "check")
	if err != nil {
		return renderError(stdout, stderr, opts.json, slicecheck.Schema, err)
	}
	set := flagSet("slice check")
	repoValue := set.String("repo", "", "repository path")
	logDirectory := set.String("log-dir", "", "repository-relative log directory")
	commits := set.Int("commits", 1, "number of slice commits")
	entries := set.Int("entries", 1, "expected appended log entries")
	ahead := set.Int("ahead", -1, "exact total commits ahead of upstream")
	jsonMode := set.Bool("json", opts.json, "emit JSON")
	if err := parseFlags(set, rest); err != nil {
		opts.json = *jsonMode
		return renderError(stdout, stderr, opts.json, slicecheck.Schema, err)
	}
	opts.json = *jsonMode
	if *repoValue == "" {
		return renderError(stdout, stderr, opts.json, slicecheck.Schema, invalid("--repo is required"))
	}
	ws, err := resolveWorkspace(opts)
	if err != nil {
		return renderError(stdout, stderr, opts.json, slicecheck.Schema, err)
	}
	repo, err := confinedPath(ws, *repoValue)
	if err != nil {
		return renderError(stdout, stderr, opts.json, slicecheck.Schema, err)
	}
	var exactAhead *int
	if *ahead >= 0 {
		exactAhead = ahead
	}
	result := slicecheck.Run(slicecheck.Options{Workspace: ws.Root, Repository: repo, Commits: *commits, ExpectedEntries: *entries, ExactAhead: exactAhead, LogDirectory: *logDirectory})
	code := 0
	if result.Status == "cannot_evaluate" {
		code = 2
	} else if result.Status != "pass" {
		code = 1
	}
	if opts.json {
		if writeJSON(stdout, stderr, result) != 0 {
			return 2
		}
		return code
	}
	for _, check := range result.Checks {
		fmt.Fprintf(stdout, "slice check %s: %s — %s\n", check.Name, check.Status, check.Detail)
	}
	fmt.Fprintf(stdout, "slice check: %d passed, %d failed, %d skipped\n", result.Passed, result.Failed, result.Skipped)
	return code
}

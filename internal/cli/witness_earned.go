package cli

import (
	"fmt"
	"io"

	"github.com/ozgurcd/achta/internal/earned"
	"github.com/ozgurcd/achta/internal/gitstate"
)

func runWitnessEarned(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	set := flagSet("witness earned")
	repoValue := set.String("repo", "", "repository path")
	recordValue := set.String("record", "", "gate-run.v1 record path")
	jsonMode := set.Bool("json", opts.json, "emit JSON")
	if err := parseFlags(set, args); err != nil {
		opts.json = *jsonMode
		return renderError(stdout, stderr, opts.json, earned.Schema, err)
	}
	opts.json = *jsonMode
	_, repo, recordPath, _, err := resolveWitnessPaths(opts, *repoValue, *recordValue)
	if err != nil {
		return renderError(stdout, stderr, opts.json, earned.Schema, err)
	}
	base, err := gitstate.NewestWitness(repo, recordPath)
	if err != nil {
		return renderError(stdout, stderr, opts.json, earned.Schema, invalid("select accepted witness: %v", err))
	}
	changed, err := gitstate.ChangedPaths(repo, base)
	if err != nil {
		return renderError(stdout, stderr, opts.json, earned.Schema, invalid("list changes since accepted witness: %v", err))
	}
	result := earned.Classify(changed)
	code := 0
	if result.Decision != "EARNED" {
		code = 1
	}
	if opts.json {
		if writeJSON(stdout, stderr, result) != 0 {
			return 2
		}
		return code
	}
	fmt.Fprintf(stdout, "witness cycle %s: %d substantive, %d record-only paths\n", result.Decision, len(result.Substantive), len(result.RecordOnly))
	return code
}

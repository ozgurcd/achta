package cli

import (
	"fmt"
	"io"

	"github.com/ozgurcd/achta/internal/safefile"
	"github.com/ozgurcd/achta/internal/witness"
)

func runWitness(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	rest, err := requireSubcommand(args, "summarize")
	if err != nil {
		return renderError(stdout, stderr, opts.json, "achta.witness-summary.v1", err)
	}
	set := flagSet("witness summarize")
	repoValue := set.String("repo", "", "repository path")
	recordValue := set.String("record", "", "gate-run.v1 record path")
	requireHead := set.String("require-head", "", "required full repository HEAD")
	slowest := set.Int("slowest", 10, "number of slow targets to report")
	jsonMode := set.Bool("json", opts.json, "emit JSON")
	if err := parseFlags(set, rest); err != nil {
		opts.json = *jsonMode
		return renderError(stdout, stderr, opts.json, "achta.witness-summary.v1", err)
	}
	opts.json = *jsonMode
	if *repoValue == "" || *recordValue == "" {
		return renderError(stdout, stderr, opts.json, "achta.witness-summary.v1", invalid("--repo and --record are required"))
	}
	ws, err := resolveWorkspace(opts)
	if err != nil {
		return renderError(stdout, stderr, opts.json, "achta.witness-summary.v1", err)
	}
	repo, err := confinedPath(ws, *repoValue)
	if err != nil {
		return renderError(stdout, stderr, opts.json, "achta.witness-summary.v1", err)
	}
	recordPath, err := confinedPath(ws, *recordValue)
	if err != nil {
		return renderError(stdout, stderr, opts.json, "achta.witness-summary.v1", err)
	}
	snapshot, err := safefile.Read(ws.Root, recordPath, witness.MaxRecord)
	if err != nil {
		return renderError(stdout, stderr, opts.json, "achta.witness-summary.v1", invalid("read witness: %v", err))
	}
	record, err := witness.Parse(snapshot.Data)
	if err != nil {
		return renderError(stdout, stderr, opts.json, "achta.witness-summary.v1", invalid("parse witness: %v", err))
	}
	summary, err := witness.Summarize(record, repo, recordPath, *requireHead, *slowest)
	if err != nil {
		return renderError(stdout, stderr, opts.json, "achta.witness-summary.v1", invalid("summarize witness: %v", err))
	}
	code := 0
	if summary.Status != "green" || summary.Freshness != "current" {
		code = 1
	}
	if opts.json {
		if writeJSON(stdout, stderr, summary) != 0 {
			return 2
		}
	} else {
		fmt.Fprintf(stdout, "witness %s: %s, %s, %s; targets %d planned/%d passed/%d failed/%d missing\n", baseName(recordPath), summary.Status, summary.Completeness, summary.Freshness, summary.PlannedTargets, summary.PassedTargets, summary.FailedTargets, summary.MissingTargets)
		for _, target := range summary.SlowTargets {
			fmt.Fprintf(stdout, "  %s: %dms\n", target.Name, target.ElapsedMS)
		}
	}
	return code
}

package cli

import (
	"fmt"
	"io"

	"github.com/ozgurcd/achta/internal/replacement"
	"github.com/ozgurcd/achta/internal/safefile"
)

func runReplacement(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	rest, err := requireSubcommand(args, "check")
	if err != nil {
		return renderError(stdout, stderr, opts.json, replacement.Schema, err)
	}
	set := flagSet("replacement check")
	fileValue := set.String("file", "", "structured replacement-claim manifest (inside the workspace)")
	var claimStatuses repeatedValue
	set.Var(&claimStatuses, "claim-status", "repeatable caller-declared status that asserts replacement or retirement")
	jsonMode := set.Bool("json", opts.json, "emit JSON")
	if err := parseFlags(set, rest); err != nil {
		opts.json = *jsonMode
		return renderError(stdout, stderr, opts.json, replacement.Schema, err)
	}
	opts.json = *jsonMode
	if *fileValue == "" || len(claimStatuses) == 0 {
		return renderError(stdout, stderr, opts.json, replacement.Schema, invalid("--file and at least one --claim-status are required"))
	}
	ws, err := resolveWorkspace(opts)
	if err != nil {
		return renderError(stdout, stderr, opts.json, replacement.Schema, err)
	}
	path, err := confinedPath(ws, *fileValue)
	if err != nil {
		return renderError(stdout, stderr, opts.json, replacement.Schema, err)
	}
	snapshot, err := safefile.Read(ws.Root, path, replacement.MaxManifest)
	if err != nil {
		return renderError(stdout, stderr, opts.json, replacement.Schema, invalid("read replacement manifest %q: %v", *fileValue, err))
	}
	manifest, err := replacement.Parse(snapshot.Data)
	if err != nil {
		return renderError(stdout, stderr, opts.json, replacement.Schema, invalid("replacement check: %v", err))
	}
	evidenceCache := map[string][]byte{}
	readEvidence := func(value string) ([]byte, error) {
		if data, ok := evidenceCache[value]; ok {
			return data, nil
		}
		evidencePath, err := confinedPath(ws, value)
		if err != nil {
			return nil, err
		}
		evidence, err := safefile.Read(ws.Root, evidencePath, replacement.MaxReplay)
		if err != nil {
			return nil, err
		}
		evidenceCache[value] = evidence.Data
		return evidence.Data, nil
	}
	result, err := replacement.Check(manifest, claimStatuses, readEvidence)
	if err != nil {
		return renderError(stdout, stderr, opts.json, replacement.Schema, invalid("replacement check: %v", err))
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
		fixture := ""
		if violation.Fixture != "" {
			fixture = " fixture=" + violation.Fixture
		}
		fmt.Fprintf(stdout, "replacement check: entry=%d verb=%q script=%q status=%q%s %s — %s\n", violation.Entry, violation.Verb, violation.Script, violation.Status, fixture, violation.Rule, violation.Text)
	}
	fmt.Fprintf(stdout, "replacement check: %s; %d claim(s), %d violation(s)\n", result.Status, result.Claims, len(result.Violations))
	return code
}

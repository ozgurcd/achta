package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/ozgurcd/achta/internal/gitstate"
	"github.com/ozgurcd/achta/internal/reachability"
)

type repeatedValue []string

func (values *repeatedValue) String() string {
	return strings.Join(*values, ",")
}

func (values *repeatedValue) Set(value string) error {
	*values = append(*values, value)
	return nil
}

func runReachability(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	if len(args) == 0 || args[0] != "classify" {
		return renderError(stdout, stderr, opts.json, reachability.Schema, invalid("reachability requires the classify subcommand"))
	}
	set := flagSet("reachability classify")
	repoValue := set.String("repo", "", "repository path")
	baseValue := set.String("base", "", "base commit")
	var declarations repeatedValue
	set.Var(&declarations, "no-reach", "repeatable PATTERN=WHY declaration")
	jsonMode := set.Bool("json", opts.json, "emit JSON")
	if err := parseFlags(set, args[1:]); err != nil {
		opts.json = *jsonMode
		return renderError(stdout, stderr, opts.json, reachability.Schema, err)
	}
	opts.json = *jsonMode
	if *repoValue == "" || *baseValue == "" {
		return renderError(stdout, stderr, opts.json, reachability.Schema, invalid("--repo and --base are required"))
	}
	ws, err := resolveWorkspace(opts)
	if err != nil {
		return renderError(stdout, stderr, opts.json, reachability.Schema, err)
	}
	repo, err := confinedPath(ws, *repoValue)
	if err != nil {
		return renderError(stdout, stderr, opts.json, reachability.Schema, err)
	}
	base, err := gitstate.ResolveCommit(repo, *baseValue)
	if err != nil {
		return renderError(stdout, stderr, opts.json, reachability.Schema, invalid("resolve base: %v", err))
	}
	ancestor, err := gitstate.IsAncestor(repo, base)
	if err != nil {
		return renderError(stdout, stderr, opts.json, reachability.Schema, invalid("check base ancestry: %v", err))
	}
	if !ancestor {
		return renderError(stdout, stderr, opts.json, reachability.Schema, mismatch("base commit is not on HEAD ancestry"))
	}
	changed, err := gitstate.ChangedPaths(repo, base)
	if err != nil {
		return renderError(stdout, stderr, opts.json, reachability.Schema, invalid("list changed paths: %v", err))
	}
	exclusions, err := parseNoReach(declarations)
	if err != nil {
		return renderError(stdout, stderr, opts.json, reachability.Schema, invalid("%v", err))
	}
	result, err := reachability.Classify(changed, exclusions)
	if err != nil {
		return renderError(stdout, stderr, opts.json, reachability.Schema, invalid("classify reachability: %v", err))
	}
	code := 0
	if result.Decision == "REQUIRED" {
		code = 1
	}
	if opts.json {
		if writeJSON(stdout, stderr, result) != 0 {
			return 2
		}
		return code
	}
	fmt.Fprintf(stdout, "reachability %s: %d changed, %d excluded, %d requiring the gate\n", strings.ToLower(result.Decision), len(result.Changed), len(result.Excluded), len(result.Reaching))
	return code
}

func parseNoReach(values []string) ([]reachability.Exclusion, error) {
	exclusions := make([]reachability.Exclusion, 0, len(values))
	for _, value := range values {
		pattern, why, ok := strings.Cut(value, "=")
		if !ok || pattern == "" || why == "" {
			return nil, fmt.Errorf("--no-reach values must be PATTERN=WHY")
		}
		exclusions = append(exclusions, reachability.Exclusion{Pattern: pattern, Why: why})
	}
	return exclusions, nil
}

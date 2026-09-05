package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/ozgurcd/achta/internal/recipe"
	"github.com/ozgurcd/achta/internal/safefile"
)

func runRecipe(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	rest, err := requireSubcommand(args, "check")
	if err != nil {
		return renderError(stdout, stderr, opts.json, recipe.Schema, err)
	}
	set := flagSet("recipe check")
	makefileValue := set.String("makefile", "", "makefile path (inside the workspace)")
	target := set.String("target", "", "target whose recipe is checked")
	var expectLines repeatedValue
	set.Var(&expectLines, "expect-line", "repeatable recipe line that must be present, byte for byte, without its leading tab")
	expectFile := set.String("expect-file", "", "file whose every line must be present, byte for byte")
	forbidNoop := set.Bool("forbid-noop", false, "refuse a recipe line whose command is true, :, echo or printf")
	jsonMode := set.Bool("json", opts.json, "emit JSON")
	if err := parseFlags(set, rest); err != nil {
		opts.json = *jsonMode
		return renderError(stdout, stderr, opts.json, recipe.Schema, err)
	}
	opts.json = *jsonMode
	if *makefileValue == "" || *target == "" {
		return renderError(stdout, stderr, opts.json, recipe.Schema, invalid("--makefile and --target are required"))
	}
	ws, err := resolveWorkspace(opts)
	if err != nil {
		return renderError(stdout, stderr, opts.json, recipe.Schema, err)
	}
	makefile, err := confinedPath(ws, *makefileValue)
	if err != nil {
		return renderError(stdout, stderr, opts.json, recipe.Schema, err)
	}
	snapshot, err := safefile.Read(ws.Root, makefile, recipe.MaxMakefile)
	if err != nil {
		return renderError(stdout, stderr, opts.json, recipe.Schema, invalid("read makefile: %v", err))
	}
	expected := append([]string(nil), expectLines...)
	if *expectFile != "" {
		path, err := confinedPath(ws, *expectFile)
		if err != nil {
			return renderError(stdout, stderr, opts.json, recipe.Schema, err)
		}
		content, err := safefile.Read(ws.Root, path, recipe.MaxMakefile)
		if err != nil {
			return renderError(stdout, stderr, opts.json, recipe.Schema, invalid("read expect-file: %v", err))
		}
		for l := range strings.SplitSeq(strings.TrimRight(string(content.Data), "\n"), "\n") {
			expected = append(expected, l)
		}
	}
	result, err := recipe.Check(snapshot.Data, recipe.Options{Target: *target, ExpectLines: expected, ForbidNoop: *forbidNoop})
	if err != nil {
		return renderError(stdout, stderr, opts.json, recipe.Schema, invalid("recipe check: %v", err))
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
		fmt.Fprintf(stdout, "recipe check %s: line %d %s — %s\n", result.Target, v.Line, v.Rule, v.Text)
	}
	for _, m := range result.MissingExpected {
		fmt.Fprintf(stdout, "recipe check %s: expected line absent — %s\n", result.Target, m)
	}
	fmt.Fprintf(stdout, "recipe check %s: %s; %d recipe line(s), %d violation(s), %d expected line(s) absent\n", result.Target, result.Status, result.Lines, len(result.Violations), len(result.MissingExpected))
	return code
}

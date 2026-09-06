package cli

import (
	"fmt"
	"io"
	"path/filepath"

	"github.com/ozgurcd/achta/internal/declaredroute"
	"github.com/ozgurcd/achta/internal/safefile"
)

func runDeclaredRoute(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	rest, err := requireSubcommand(args, "check")
	if err != nil {
		return renderError(stdout, stderr, opts.json, declaredroute.Schema, err)
	}
	set := flagSet("declared-route check")
	var files, routePatterns, banPatterns repeatedValue
	set.Var(&files, "file", "repeatable workflow YAML path (inside the workspace)")
	set.Var(&routePatterns, "route-pattern", "repeatable caller regexp for an allowed route alternative")
	set.Var(&banPatterns, "ban-pattern", "repeatable caller regexp forbidden in live YAML scalar text")
	requiredRoutePattern := set.String("required-route-pattern", "", "configured route alternative that requires the scoped key")
	requiredKey := set.String("required-key", "", "caller-owned mapping key required exactly once")
	requiredScope := set.String("required-scope", "", "JSON Pointer to the YAML mapping containing the required key")
	jsonMode := set.Bool("json", opts.json, "emit JSON")
	if err := parseFlags(set, rest); err != nil {
		opts.json = *jsonMode
		return renderError(stdout, stderr, opts.json, declaredroute.Schema, err)
	}
	opts.json = *jsonMode
	if len(files) > declaredroute.MaxDocuments {
		return renderError(stdout, stderr, opts.json, declaredroute.Schema, invalid("workflow count exceeds %d", declaredroute.MaxDocuments))
	}
	ws, err := resolveWorkspace(opts)
	if err != nil {
		return renderError(stdout, stderr, opts.json, declaredroute.Schema, err)
	}
	documents := make([]declaredroute.Document, 0, len(files))
	seen := map[string]bool{}
	for _, value := range files {
		path, pathErr := confinedPath(ws, value)
		if pathErr != nil {
			return renderError(stdout, stderr, opts.json, declaredroute.Schema, pathErr)
		}
		if seen[path] {
			return renderError(stdout, stderr, opts.json, declaredroute.Schema, invalid("workflow %q is repeated", value))
		}
		seen[path] = true
		snapshot, readErr := safefile.Read(ws.Root, path, declaredroute.MaxDocument)
		if readErr != nil {
			return renderError(stdout, stderr, opts.json, declaredroute.Schema, invalid("read workflow %q: %v", value, readErr))
		}
		documents = append(documents, declaredroute.Document{Path: filepath.ToSlash(value), Data: snapshot.Data})
	}
	result, err := declaredroute.Check(documents, declaredroute.Options{
		RoutePatterns:        routePatterns,
		BanPatterns:          banPatterns,
		RequiredRoutePattern: *requiredRoutePattern,
		RequiredKey:          *requiredKey,
		RequiredScope:        *requiredScope,
	})
	if err != nil {
		return renderError(stdout, stderr, opts.json, declaredroute.Schema, invalid("declared-route check: %v", err))
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
		fmt.Fprintf(stdout, "declared-route check: %s:%d %s expected=%d observed=%d — %s\n", violation.File, violation.Line, violation.Rule, violation.Expected, violation.Observed, violation.Text)
	}
	fmt.Fprintf(stdout, "declared-route check: %s; %d workflow(s), %d violation(s)\n", result.Status, len(result.Documents), len(result.Violations))
	for _, refused := range result.Refused {
		fmt.Fprintf(stdout, "declared-route check: refused — %s\n", refused)
	}
	return code
}

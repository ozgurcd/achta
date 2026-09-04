package cli

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/ozgurcd/achta/internal/safefile"
	"github.com/ozgurcd/achta/internal/toolchain"
)

func runToolchain(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	if len(args) == 0 || args[0] != "check" {
		return renderError(stdout, stderr, opts.json, toolchain.ResultSchema, invalid("toolchain requires the check subcommand"))
	}
	set := flagSet("toolchain check")
	repoValue := set.String("repo", "", "repository path")
	manifestValue := set.String("manifest", "", "toolchain manifest path")
	workflowValue := set.String("workflow", "", "CI workflow path")
	jsonMode := set.Bool("json", opts.json, "emit JSON")
	if err := parseFlags(set, args[1:]); err != nil {
		opts.json = *jsonMode
		return renderError(stdout, stderr, opts.json, toolchain.ResultSchema, err)
	}
	opts.json = *jsonMode
	if *repoValue == "" || *manifestValue == "" || *workflowValue == "" {
		return renderError(stdout, stderr, opts.json, toolchain.ResultSchema, invalid("--repo, --manifest, and --workflow are required"))
	}
	ws, err := resolveWorkspace(opts)
	if err != nil {
		return renderError(stdout, stderr, opts.json, toolchain.ResultSchema, err)
	}
	repo, err := confinedPath(ws, *repoValue)
	if err != nil {
		return renderError(stdout, stderr, opts.json, toolchain.ResultSchema, err)
	}
	manifestPath, err := confinedPath(ws, *manifestValue)
	if err != nil {
		return renderError(stdout, stderr, opts.json, toolchain.ResultSchema, err)
	}
	workflowPath, err := confinedPath(ws, *workflowValue)
	if err != nil {
		return renderError(stdout, stderr, opts.json, toolchain.ResultSchema, err)
	}
	if !insideRepository(repo, manifestPath) || !insideRepository(repo, workflowPath) {
		return renderError(stdout, stderr, opts.json, toolchain.ResultSchema, invalid("manifest and workflow must be inside the selected repository"))
	}
	manifest, err := safefile.Read(ws.Root, manifestPath, toolchain.MaxManifest)
	if err != nil {
		return renderError(stdout, stderr, opts.json, toolchain.ResultSchema, invalid("read toolchain manifest: %v", err))
	}
	workflow, err := safefile.Read(ws.Root, workflowPath, toolchain.MaxWorkflow)
	if err != nil {
		return renderError(stdout, stderr, opts.json, toolchain.ResultSchema, invalid("read workflow: %v", err))
	}
	result, err := toolchain.Check(repo, manifest.Data, workflow.Data)
	if err != nil {
		return renderError(stdout, stderr, opts.json, toolchain.ResultSchema, invalid("check toolchain parity: %v", err))
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
	fmt.Fprintf(stdout, "toolchain parity %s: %d pins checked\n", result.Status, len(result.Pins))
	return code
}

func insideRepository(repo, target string) bool {
	relative, err := filepath.Rel(repo, target)
	return err == nil && relative != ".." && !filepath.IsAbs(relative) && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

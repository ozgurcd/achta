package cli

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/ozgurcd/achta/internal/mirror"
	"github.com/ozgurcd/achta/internal/safefile"
)

func runMirror(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	rest, err := requireSubcommand(args, "check")
	if err != nil {
		return renderError(stdout, stderr, opts.json, mirror.Schema, err)
	}
	set := flagSet("mirror check")
	masterValue := set.String("master", "", "master path (inside the workspace)")
	var mirrorValues repeatedValue
	set.Var(&mirrorValues, "mirror", "repeatable mirror path (inside the workspace)")
	jsonMode := set.Bool("json", opts.json, "emit JSON")
	if err := parseFlags(set, rest); err != nil {
		opts.json = *jsonMode
		return renderError(stdout, stderr, opts.json, mirror.Schema, err)
	}
	opts.json = *jsonMode
	if *masterValue == "" || len(mirrorValues) == 0 {
		return renderError(stdout, stderr, opts.json, mirror.Schema, invalid("--master and at least one --mirror are required"))
	}

	ws, err := resolveWorkspace(opts)
	if err != nil {
		return renderError(stdout, stderr, opts.json, mirror.Schema, err)
	}
	masterPath, err := confinedPath(ws, *masterValue)
	if err != nil {
		return renderError(stdout, stderr, opts.json, mirror.Schema, err)
	}
	master, err := safefile.Read(ws.Root, masterPath, mirror.MaxFile)
	if errors.Is(err, os.ErrNotExist) {
		return renderError(stdout, stderr, opts.json, mirror.Schema, invalid("master %q is absent", *masterValue))
	}
	if err != nil {
		return renderError(stdout, stderr, opts.json, mirror.Schema, invalid("read master: %v", err))
	}

	inputs := make([]mirror.Input, 0, len(mirrorValues))
	for _, value := range mirrorValues {
		path, err := confinedPath(ws, value)
		if err != nil {
			return renderError(stdout, stderr, opts.json, mirror.Schema, err)
		}
		snapshot, err := safefile.Read(ws.Root, path, mirror.MaxFile)
		if errors.Is(err, os.ErrNotExist) {
			inputs = append(inputs, mirror.Input{Path: value, Absent: true})
			continue
		}
		if err != nil {
			return renderError(stdout, stderr, opts.json, mirror.Schema, invalid("read mirror %q: %v", value, err))
		}
		inputs = append(inputs, mirror.Input{Path: value, Data: snapshot.Data})
	}

	result, err := mirror.Check(*masterValue, master.Data, inputs)
	if err != nil {
		return renderError(stdout, stderr, opts.json, mirror.Schema, invalid("mirror check: %v", err))
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
	for _, item := range result.Mirrors {
		switch item.Status {
		case "absent":
			fmt.Fprintf(stdout, "mirror %s: absent; master_sha256=%s\n", item.Path, item.MasterSHA256)
		default:
			fmt.Fprintf(stdout, "mirror %s: %s; master_sha256=%s mirror_sha256=%s\n", item.Path, item.Status, item.MasterSHA256, item.MirrorSHA256)
		}
	}
	fmt.Fprintf(stdout, "mirror check: %s; %d mirror(s)\n", result.Status, len(result.Mirrors))
	return code
}

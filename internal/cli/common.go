package cli

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/ozgurcd/achta/internal/workspace"
)

func flagSet(name string) *flag.FlagSet {
	set := flag.NewFlagSet(name, flag.ContinueOnError)
	set.SetOutput(io.Discard)
	return set
}

func resolveWorkspace(opts globalOptions) (workspace.Workspace, error) {
	if opts.workspace != "" && opts.wikiDir != "" {
		return workspace.Workspace{}, invalid("--workspace and --wiki-dir are mutually exclusive")
	}
	if opts.wikiDir != "" {
		resolved, err := workspace.ResolveWikiDir(opts.wikiDir)
		if err != nil {
			return workspace.Workspace{}, invalid("%v", err)
		}
		return resolved, nil
	}
	cwd, err := os.Getwd()
	if err != nil {
		return workspace.Workspace{}, invalid("current directory: %v", err)
	}
	resolved, err := workspace.Resolve(opts.workspace, cwd)
	if err != nil {
		return workspace.Workspace{}, invalid("%v", err)
	}
	return resolved, nil
}

func confinedPath(root workspace.Workspace, path string) (string, error) {
	confined, err := root.Confine(path)
	if err != nil {
		return "", invalid("%v", err)
	}
	return confined, nil
}

func rejectSecretLikeInput(path string) error {
	name := strings.ToLower(filepath.Base(path))
	if name == ".env" || strings.HasPrefix(name, ".env.") || strings.HasSuffix(name, ".env") {
		return invalid("refusing secret-like input path")
	}
	switch filepath.Ext(name) {
	case ".key", ".pem", ".p12", ".pfx":
		return invalid("refusing secret-like input path")
	}
	for _, token := range strings.FieldsFunc(name, func(value rune) bool {
		return !unicode.IsLetter(value) && !unicode.IsNumber(value)
	}) {
		switch token {
		case "secret", "secrets", "token", "tokens", "credential", "credentials", "password", "passwords", "passwd", "cookie", "cookies", "totp", "license", "recovery":
			return invalid("refusing secret-like input path")
		}
	}
	return nil
}

func requireSubcommand(args []string, wanted string) ([]string, error) {
	if len(args) == 0 || args[0] != wanted {
		return nil, invalid("expected subcommand %q", wanted)
	}
	return args[1:], nil
}

func parseFlags(set *flag.FlagSet, args []string) error {
	if err := set.Parse(args); err != nil {
		return invalid("%v", err)
	}
	if set.NArg() != 0 {
		return invalid("unexpected argument %q", set.Arg(0))
	}
	return nil
}

func baseName(path string) string {
	if path == "" {
		return ""
	}
	return filepath.Base(path)
}

func operationHuman(stdout io.Writer, operation, status, detail string) {
	fmt.Fprintf(stdout, "%s: %s", operation, status)
	if detail != "" {
		fmt.Fprintf(stdout, " (%s)", detail)
	}
	fmt.Fprintln(stdout)
}

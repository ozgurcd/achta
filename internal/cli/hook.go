package cli

import (
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/ozgurcd/achta/internal/hookcd"
)

const (
	hookMaxVariablesEnv  = "ACHTA_HOOK_CD_MAX_VARIABLES"
	hookMaxValueBytesEnv = "ACHTA_HOOK_CD_MAX_VALUE_BYTES"
)

// hookLimits reads the variable-table limits from the environment. An unset
// value takes the hookcd default; an invalid one warns and takes it too.
func hookLimits(stderr io.Writer) hookcd.Limits {
	read := func(name string) int {
		raw, set := os.LookupEnv(name)
		if !set {
			return 0
		}
		value, err := strconv.Atoi(raw)
		if err != nil || value <= 0 {
			fmt.Fprintf(stderr, "achta hook cd: WARNING — %s must be a positive integer; using the default\n", name)
			return 0
		}
		return value
	}
	return hookcd.Limits{MaxVariables: read(hookMaxVariablesEnv), MaxValueBytes: read(hookMaxValueBytesEnv)}
}

func runHook(args []string, stdin io.Reader, stderr io.Writer) int {
	rest, err := requireSubcommand(args, "cd")
	if err != nil {
		fmt.Fprintf(stderr, "achta hook cd: %v\n", err)
		return 2
	}
	if len(rest) != 0 {
		fmt.Fprintf(stderr, "achta hook cd: unexpected argument %q\n", rest[0])
		return 2
	}
	payload, err := io.ReadAll(io.LimitReader(stdin, hookcd.MaxInput+1))
	if err != nil {
		fmt.Fprintln(stderr, "achta hook cd: WARNING — could not read PreToolUse input; allowing")
		return 0
	}
	result := hookcd.EvaluateWithLimits(payload, hookLimits(stderr))
	switch result.Verdict {
	case hookcd.Allow:
		return 0
	case hookcd.Warn:
		fmt.Fprintf(stderr, "achta hook cd: WARNING — %s; allowing\n", result.Reason)
		return 0
	case hookcd.Deny:
		fmt.Fprintf(stderr, "achta hook cd: DENY — statement %d (%s) may write without an absolute repository selector\n", result.Statement, result.Verb)
		if result.Problem != "" {
			fmt.Fprintf(stderr, "why: %s\n", result.Problem)
		}
		if result.Hint != "" {
			fmt.Fprintf(stderr, "hint: %s\n", result.Hint)
		}
		fmt.Fprintf(stderr, "fix: %s\n", result.Fix)
		return 2
	default:
		fmt.Fprintln(stderr, "achta hook cd: WARNING — evaluator returned an unknown verdict; allowing")
		return 0
	}
}

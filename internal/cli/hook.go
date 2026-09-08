package cli

import (
	"fmt"
	"io"

	"github.com/ozgurcd/achta/internal/hookcd"
)

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
	result := hookcd.Evaluate(payload)
	switch result.Verdict {
	case hookcd.Allow:
		return 0
	case hookcd.Warn:
		fmt.Fprintf(stderr, "achta hook cd: WARNING — %s; allowing\n", result.Reason)
		return 0
	case hookcd.Deny:
		fmt.Fprintf(stderr, "achta hook cd: DENY — statement %d (%s) may write without an absolute repository selector\n", result.Statement, result.Verb)
		fmt.Fprintln(stderr, "fix: make the first statement `cd /absolute/path`, `cd ~/path`, or `cd $HOME/path`; otherwise give every writing statement `-C /absolute/path`")
		return 2
	default:
		fmt.Fprintln(stderr, "achta hook cd: WARNING — evaluator returned an unknown verdict; allowing")
		return 0
	}
}

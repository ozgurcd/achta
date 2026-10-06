package cli

import (
	"fmt"
	"io"
	"time"

	"github.com/ozgurcd/achta/internal/claim"
)

func runClaim(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	opts.json = opts.json || jsonRequested(args)
	if len(args) == 0 {
		return renderError(stdout, stderr, opts.json, claim.Schema, invalid("claim requires take, status, check or release"))
	}
	set := flagSet("claim")
	repo := set.String("repo", "", "absolute checkout root")
	slice := set.String("slice", "", "slice name")
	note := set.String("note", "", "claim note")
	reason := set.String("reason", "", "forced release reason")
	force := set.Bool("force", false, "release another slice's claim")
	jsonMode := set.Bool("json", opts.json, "emit JSON")
	if err := parseFlags(set, args[1:]); err != nil {
		return renderError(stdout, stderr, opts.json, claim.Schema, invalid("invalid claim flags"))
	}
	opts.json = *jsonMode
	if err := rejectSecretLikeInput(*repo); err != nil {
		return renderError(stdout, stderr, opts.json, claim.Schema, err)
	}
	result, err := claim.Run(args[0], claim.Options{Repo: *repo, Slice: *slice, Note: *note, Reason: *reason, Force: *force, Now: time.Now()})
	if err != nil {
		return renderError(stdout, stderr, opts.json, claim.Schema, invalid("%v", err))
	}
	code := 0
	if result.Status == "fail" {
		code = 1
	}
	if opts.json {
		if writeJSON(stdout, stderr, result) != 0 {
			return 2
		}
		return code
	}
	fmt.Fprintf(stdout, "claim %s: %s\n", result.Operation, result.Status)
	if result.Claim == nil {
		fmt.Fprintln(stdout, "holder: none")
	} else {
		fmt.Fprintf(stdout, "holder: %s; age %ds; since %s; note %s\n", result.Claim.Slice, result.AgeSeconds, result.Claim.Time.Format(time.RFC3339), result.Claim.Note)
	}
	if result.Reason != "" {
		fmt.Fprintln(stdout, result.Reason)
	}
	if result.Checkout != nil {
		fmt.Fprintf(stdout, "tracked changes: %d\n", result.TrackedChangeCount)
		for _, path := range result.TrackedChanges {
			fmt.Fprintf(stdout, "  %q\n", path)
		}
		if result.UnpushedCommitCount == nil {
			fmt.Fprintln(stdout, "unpushed commits: unknown (no local upstream)")
		} else {
			fmt.Fprintf(stdout, "unpushed commits: %d\n", *result.UnpushedCommitCount)
		}
		for _, path := range result.UnpushedFiles {
			fmt.Fprintf(stdout, "  %q\n", path)
		}
	}
	for _, release := range result.Releases {
		fmt.Fprintf(stdout, "released %s by %s at %s; forced=%t; reason %s\n", release.Holder, release.By, release.Time.Format(time.RFC3339), release.Forced, release.Reason)
	}
	return code
}

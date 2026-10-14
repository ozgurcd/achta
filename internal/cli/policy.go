package cli

import (
	"fmt"
	"io"

	"github.com/ozgurcd/achta/internal/repopolicy"
)

const policyHelp = `policy read --repo ABS --repo-id ID --registry ABS --policy ABS [--minimum-epoch N] [--json]
Read-only closed registered-repository policy projection; missing policy is off.
Paths and minimum epoch are explicit inputs, not activation or owner authority.
A1 has no production trust verifier: existing policy refuses evaluation.
No setter, leases, queue, locks, repair or protected-history rollback claim.
Legacy checkout claims remain separate and unchanged.
`

func runPolicy(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	opts.json = opts.json || jsonRequested(args)
	if len(args) == 0 || args[0] != "read" {
		return renderError(stdout, stderr, opts.json, repopolicy.ReadSchema, invalid("policy requires read"))
	}
	if commandHelpRequested(args[1:]) {
		fmt.Fprint(stdout, policyHelp)
		return 0
	}
	set := flagSet("policy read")
	repo := set.String("repo", "", "exact absolute checkout path")
	id := set.String("repo-id", "", "owner-registered repository ID")
	registry := set.String("registry", "", "explicit canonical registry path")
	policy := set.String("policy", "", "explicit canonical policy path")
	epoch := set.Uint64("minimum-epoch", 0, "injected minimum; no protected-history claim")
	jsonMode := set.Bool("json", opts.json, "emit JSON")
	if err := parseFlags(set, args[1:]); err != nil {
		return renderError(stdout, stderr, opts.json, repopolicy.ReadSchema, invalid("invalid policy read flags"))
	}
	opts.json = *jsonMode
	for _, value := range []string{*repo, *id, *registry, *policy} {
		if err := rejectSecretLikeInput(value); err != nil {
			return renderError(stdout, stderr, opts.json, repopolicy.ReadSchema, err)
		}
	}
	result, err := repopolicy.Read(repopolicy.Options{RepoID: *id, CheckoutPath: *repo, RegistryPath: *registry, PolicyPath: *policy, MinimumEpoch: *epoch})
	if err != nil {
		return renderError(stdout, stderr, opts.json, repopolicy.ReadSchema, invalid("%v", err))
	}
	if opts.json {
		return writeJSON(stdout, stderr, result)
	}
	fmt.Fprintf(stdout, "policy read: %s; desired %s; effective %s; epoch %d; registered %t\n", result.Status, result.DesiredMode, result.EffectiveMode, result.Epoch, result.Registered)
	return 0
}

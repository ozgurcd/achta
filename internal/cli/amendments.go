package cli

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/ozgurcd/achta/internal/amendments"
	"github.com/ozgurcd/achta/internal/gitstate"
	"github.com/ozgurcd/achta/internal/rulefloorclient"
	"github.com/ozgurcd/achta/internal/safefile"
)

type amendmentOperation struct {
	SchemaVersion string `json:"schema_version"`
	Operation     string `json:"operation"`
	Status        string `json:"status"`
	Manifest      string `json:"manifest"`
	BaseCommit    string `json:"base_commit"`
	RuleID        string `json:"rule_id,omitempty"`
	ChangeClass   string `json:"change_class,omitempty"`
	Changed       bool   `json:"changed"`
}

func runAmendments(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	if len(args) == 0 {
		return renderError(stdout, stderr, opts.json, "achta.amendments-operation.v1", invalid("amendments requires a subcommand"))
	}
	switch args[0] {
	case "declare":
		return runAmendmentsDeclare(args[1:], stdout, stderr, opts)
	case "rebase":
		return runAmendmentsRebase(args[1:], stdout, stderr, opts)
	case "reconcile":
		return runAmendmentsReconcile(args[1:], stdout, stderr, opts)
	default:
		return renderError(stdout, stderr, opts.json, "achta.amendments-operation.v1", invalid("unknown amendments subcommand %q", args[0]))
	}
}

func runAmendmentsDeclare(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	set := flagSet("amendments declare")
	manifestValue := set.String("manifest", "", "manifest path")
	rule := set.String("rule", "", "Rulefloor rule ID")
	class := set.String("class", "", "change class")
	reasonValue := set.String("reason-file", "", "one-line reason file")
	digest := set.String("after-sentence-sha256", "", "Rulefloor sentence digest")
	check := set.Bool("check", false, "validate without writing")
	jsonMode := set.Bool("json", opts.json, "emit JSON")
	if err := parseFlags(set, args); err != nil {
		opts.json = *jsonMode
		return renderError(stdout, stderr, opts.json, "achta.amendments-operation.v1", err)
	}
	opts.json = *jsonMode
	if *manifestValue == "" || *rule == "" || *class == "" || *reasonValue == "" {
		return renderError(stdout, stderr, opts.json, "achta.amendments-operation.v1", invalid("--manifest, --rule, --class, and --reason-file are required"))
	}
	if err := rejectSecretLikeInput(*reasonValue); err != nil {
		return renderError(stdout, stderr, opts.json, "achta.amendments-operation.v1", err)
	}
	ws, err := resolveWorkspace(opts)
	if err != nil {
		return renderError(stdout, stderr, opts.json, "achta.amendments-operation.v1", err)
	}
	manifestPath, err := confinedPath(ws, *manifestValue)
	if err != nil {
		return renderError(stdout, stderr, opts.json, "achta.amendments-operation.v1", err)
	}
	reasonPath, err := confinedPath(ws, *reasonValue)
	if err != nil {
		return renderError(stdout, stderr, opts.json, "achta.amendments-operation.v1", err)
	}
	manifestSnapshot, err := safefile.Read(ws.Root, manifestPath, amendments.MaxManifest)
	if err != nil {
		return renderError(stdout, stderr, opts.json, "achta.amendments-operation.v1", invalid("read manifest: %v", err))
	}
	reasonSnapshot, err := safefile.Read(ws.Root, reasonPath, amendments.MaxReason+2)
	if err != nil {
		return renderError(stdout, stderr, opts.json, "achta.amendments-operation.v1", invalid("read reason: %v", err))
	}
	reason := strings.TrimSuffix(string(reasonSnapshot.Data), "\n")
	manifest, err := amendments.Parse(manifestSnapshot.Data)
	if err != nil {
		return renderError(stdout, stderr, opts.json, "achta.amendments-operation.v1", invalid("parse manifest: %v", err))
	}
	manifest, err = amendments.Declare(manifest, amendments.Change{RuleID: *rule, ChangeClass: *class, AfterSentenceSHA256: *digest, Reason: reason})
	if err != nil {
		if errors.Is(err, amendments.ErrDuplicateDeclaration) {
			return renderError(stdout, stderr, opts.json, "achta.amendments-operation.v1", mismatch("declare amendment: %v", err))
		}
		return renderError(stdout, stderr, opts.json, "achta.amendments-operation.v1", invalid("declare amendment: %v", err))
	}
	data, err := amendments.Encode(manifest)
	if err != nil {
		return renderError(stdout, stderr, opts.json, "achta.amendments-operation.v1", invalid("encode manifest: %v", err))
	}
	status, code := "updated", 0
	if *check {
		status, code = "would_change", 1
	} else if err := manifestSnapshot.Replace(data, amendments.MaxManifest); err != nil {
		return renderError(stdout, stderr, opts.json, "achta.amendments-operation.v1", invalid("replace manifest: %v", err))
	}
	doc := amendmentOperation{SchemaVersion: "achta.amendments-operation.v1", Operation: "declare", Status: status, Manifest: *manifestValue, BaseCommit: manifest.BaseCommit, RuleID: *rule, ChangeClass: *class, Changed: true}
	return renderAmendmentOperation(stdout, stderr, opts, doc, code)
}

func runAmendmentsRebase(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	set := flagSet("amendments rebase")
	manifestValue := set.String("manifest", "", "manifest path")
	repoValue := set.String("repo", "", "repository path")
	witnessValue := set.String("witness-log", "", "accepted witness record path")
	check := set.Bool("check", false, "validate without writing")
	jsonMode := set.Bool("json", opts.json, "emit JSON")
	if err := parseFlags(set, args); err != nil {
		opts.json = *jsonMode
		return renderError(stdout, stderr, opts.json, "achta.amendments-operation.v1", err)
	}
	opts.json = *jsonMode
	if *manifestValue == "" || *repoValue == "" || *witnessValue == "" {
		return renderError(stdout, stderr, opts.json, "achta.amendments-operation.v1", invalid("--manifest, --repo, and --witness-log are required"))
	}
	ws, err := resolveWorkspace(opts)
	if err != nil {
		return renderError(stdout, stderr, opts.json, "achta.amendments-operation.v1", err)
	}
	manifestPath, err := confinedPath(ws, *manifestValue)
	if err != nil {
		return renderError(stdout, stderr, opts.json, "achta.amendments-operation.v1", err)
	}
	repo, err := confinedPath(ws, *repoValue)
	if err != nil {
		return renderError(stdout, stderr, opts.json, "achta.amendments-operation.v1", err)
	}
	witnessPath, err := confinedPath(ws, *witnessValue)
	if err != nil {
		return renderError(stdout, stderr, opts.json, "achta.amendments-operation.v1", err)
	}
	if _, err := safefile.Read(ws.Root, witnessPath, 8<<20); err != nil {
		return renderError(stdout, stderr, opts.json, "achta.amendments-operation.v1", invalid("read witness log: %v", err))
	}
	snapshot, err := safefile.Read(ws.Root, manifestPath, amendments.MaxManifest)
	if err != nil {
		return renderError(stdout, stderr, opts.json, "achta.amendments-operation.v1", invalid("read manifest: %v", err))
	}
	manifest, err := amendments.Parse(snapshot.Data)
	if err != nil {
		return renderError(stdout, stderr, opts.json, "achta.amendments-operation.v1", invalid("parse manifest: %v", err))
	}
	base, err := gitstate.NewestWitness(repo, witnessPath)
	if err != nil {
		return renderError(stdout, stderr, opts.json, "achta.amendments-operation.v1", invalid("select witness: %v", err))
	}
	changed := manifest.BaseCommit != base
	if !changed && !*check {
		return renderError(stdout, stderr, opts.json, "achta.amendments-operation.v1", mismatch("manifest already uses selected witness"))
	}
	status, code := "unchanged", 0
	if changed {
		manifest, err = amendments.Rebase(manifest, base)
		if err != nil {
			return renderError(stdout, stderr, opts.json, "achta.amendments-operation.v1", invalid("rebase manifest: %v", err))
		}
		data, encodeErr := amendments.Encode(manifest)
		if encodeErr != nil {
			return renderError(stdout, stderr, opts.json, "achta.amendments-operation.v1", invalid("encode manifest: %v", encodeErr))
		}
		if *check {
			status, code = "would_change", 1
		} else {
			status = "updated"
			if replaceErr := snapshot.Replace(data, amendments.MaxManifest); replaceErr != nil {
				return renderError(stdout, stderr, opts.json, "achta.amendments-operation.v1", invalid("replace manifest: %v", replaceErr))
			}
		}
	}
	doc := amendmentOperation{SchemaVersion: "achta.amendments-operation.v1", Operation: "rebase", Status: status, Manifest: *manifestValue, BaseCommit: base, Changed: changed}
	return renderAmendmentOperation(stdout, stderr, opts, doc, code)
}

func runAmendmentsReconcile(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	set := flagSet("amendments reconcile")
	manifestValue := set.String("manifest", "", "manifest path")
	repoValue := set.String("repo", "", "repository path")
	rulefloorPath := set.String("rulefloor", "", "Rulefloor executable path")
	jsonMode := set.Bool("json", opts.json, "emit JSON")
	if err := parseFlags(set, args); err != nil {
		opts.json = *jsonMode
		return renderError(stdout, stderr, opts.json, "achta.amendments-reconciliation.v1", err)
	}
	opts.json = *jsonMode
	if *manifestValue == "" || *repoValue == "" {
		return renderError(stdout, stderr, opts.json, "achta.amendments-reconciliation.v1", invalid("--manifest and --repo are required"))
	}
	ws, err := resolveWorkspace(opts)
	if err != nil {
		return renderError(stdout, stderr, opts.json, "achta.amendments-reconciliation.v1", err)
	}
	manifestPath, err := confinedPath(ws, *manifestValue)
	if err != nil {
		return renderError(stdout, stderr, opts.json, "achta.amendments-reconciliation.v1", err)
	}
	repo, err := confinedPath(ws, *repoValue)
	if err != nil {
		return renderError(stdout, stderr, opts.json, "achta.amendments-reconciliation.v1", err)
	}
	snapshot, err := safefile.Read(ws.Root, manifestPath, amendments.MaxManifest)
	if err != nil {
		return renderError(stdout, stderr, opts.json, "achta.amendments-reconciliation.v1", invalid("read manifest: %v", err))
	}
	manifest, err := amendments.Parse(snapshot.Data)
	if err != nil {
		return renderError(stdout, stderr, opts.json, "achta.amendments-reconciliation.v1", invalid("parse manifest: %v", err))
	}
	client, selectedRulefloor, err := (rulefloorclient.Client{Path: *rulefloorPath}).Resolve()
	if err != nil {
		return renderError(stdout, stderr, opts.json, "achta.amendments-reconciliation.v1", invalid("Rulefloor executable: %v", err))
	}
	if err := client.CheckCompatibility(); err != nil {
		return renderError(stdout, stderr, opts.json, "achta.amendments-reconciliation.v1", invalid("Rulefloor capabilities using %s: %v", selectedRulefloor, err))
	}
	diff, rawDiff, err := client.LedgerDiff(manifest.BaseCommit, repo)
	if err != nil {
		return renderError(stdout, stderr, opts.json, "achta.amendments-reconciliation.v1", invalid("Rulefloor ledger diff using %s: %v", selectedRulefloor, err))
	}
	result := amendments.Reconcile(manifest, diff, rawDiff)
	result.RulefloorExecutable = selectedRulefloor
	code := 0
	if result.Status != "pass" {
		code = 1
	}
	if opts.json {
		if writeJSON(stdout, stderr, result) != 0 {
			return 2
		}
	} else {
		fmt.Fprintf(stdout, "amendments reconcile: %s using %s; %d declared, %d measured, %d problem(s), %d header change(s)\n", result.Status, result.RulefloorExecutable, result.DeclaredChanges, result.ActualChanges, len(result.Problems), len(result.HeaderChanges))
	}
	return code
}

func renderAmendmentOperation(stdout, stderr io.Writer, opts globalOptions, doc amendmentOperation, code int) int {
	if opts.json {
		if writeJSON(stdout, stderr, doc) != 0 {
			return 2
		}
	} else {
		operationHuman(stdout, "amendments "+doc.Operation, doc.Status, doc.Manifest)
	}
	return code
}

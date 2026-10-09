package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"runtime/debug"
	"sort"
	"strings"
	"time"

	"github.com/ozgurcd/achta/internal/buildinfo"
	"github.com/ozgurcd/achta/internal/census"
	"github.com/ozgurcd/achta/internal/countcheck"
	"github.com/ozgurcd/achta/internal/declaredroute"
	"github.com/ozgurcd/achta/internal/floorcensus"
	"github.com/ozgurcd/achta/internal/hookcd"
	"github.com/ozgurcd/achta/internal/ledgerrows"
	"github.com/ozgurcd/achta/internal/mirror"
	"github.com/ozgurcd/achta/internal/recipe"
	achtawiki "github.com/ozgurcd/achta/internal/wiki"
)

const (
	versionSchema      = "achta.version.v1"
	capabilitiesSchema = "achta.capabilities.v1"
)

type globalOptions struct {
	workspace string
	wikiDir   string
	json      bool
	quiet     bool
	timing    bool
}

type machineError struct {
	SchemaVersion string `json:"schema_version"`
	Status        string `json:"status"`
	Error         string `json:"error"`
}

// ExitError carries Achta's stable three-class process exit contract.
type ExitError struct {
	Code int
	Kind string
	Err  error
}

func (e *ExitError) Error() string { return e.Err.Error() }
func (e *ExitError) Unwrap() error { return e.Err }

func invalid(format string, args ...any) error {
	return &ExitError{Code: 2, Kind: "cannot_evaluate", Err: fmt.Errorf(format, args...)}
}

func mismatch(format string, args ...any) error {
	return &ExitError{Code: 1, Kind: "mismatch", Err: fmt.Errorf(format, args...)}
}

// Run executes one CLI invocation and returns its process exit code.
func Run(args []string, stdout, stderr io.Writer, releaseVersion string) int {
	return runWithInput(args, os.Stdin, stdout, stderr, releaseVersion)
}

func runWithInput(args []string, stdin io.Reader, stdout, stderr io.Writer, releaseVersion string) int {
	if len(args) >= 2 && args[0] == "hook" && args[1] == "journal" {
		if len(args) != 2 {
			return 0
		}
		return runJournalHook(stdin, stdout)
	}
	started := time.Now()
	opts, command, rest, err := parseGlobal(args)
	if command == "hook" && len(rest) > 0 && rest[0] == "journal" {
		if err != nil || len(rest) != 1 {
			return 0
		}
		return runJournalHook(stdin, stdout)
	}
	if command == "gate" && err == nil {
		if commandHelpRequested(rest) {
			fmt.Fprintln(stdout, "gate run --repo ABS [--json] -- COMMAND [ARG...]\nKeeps 20 private Git-directory logs; prints at most 25 lines; returns the command exit code.")
			return 0
		}
		return runGate(rest, stdout, stderr, opts)
	}
	jsonMode := jsonRequested(args)
	if !opts.timing && (!opts.quiet || jsonMode) {
		return dispatch(args, stdin, stdout, stderr, releaseVersion, opts, command, rest, err)
	}
	if jsonMode {
		opts.json = true
	}

	var commandStdout, commandStderr bytes.Buffer
	code := dispatch(args, stdin, &commandStdout, &commandStderr, releaseVersion, opts, command, rest, err)
	if jsonMode {
		document := commandStdout.Bytes()
		if opts.timing {
			var appendErr error
			document, appendErr = appendElapsedJSON(document, time.Since(started).Milliseconds())
			if appendErr != nil {
				fmt.Fprintf(stderr, "achta: append timing to JSON: %v\n", appendErr)
				return 2
			}
		}
		_, _ = stdout.Write(document)
		_, _ = stderr.Write(commandStderr.Bytes())
		return code
	}

	if !opts.quiet || code != 0 {
		_, _ = stdout.Write(commandStdout.Bytes())
	}
	_, _ = stderr.Write(commandStderr.Bytes())
	if opts.timing {
		elapsedMS := time.Since(started).Milliseconds()
		if code == 0 {
			fmt.Fprintf(stdout, "elapsed: %dms\n", elapsedMS)
		} else {
			fmt.Fprintf(stderr, "elapsed: %dms\n", elapsedMS)
		}
	}
	return code
}

func dispatch(args []string, stdin io.Reader, stdout, stderr io.Writer, releaseVersion string, opts globalOptions, command string, rest []string, err error) int {
	if err != nil {
		return renderError(stdout, stderr, opts.json, "achta.error.v1", err)
	}
	if command != "" && command != "help" && commandHelpRequested(rest) {
		fmt.Fprint(stdout, helpText)
		if len(rest) > 0 {
			fmt.Fprint(stdout, scopedCommandHelp[command+" "+rest[0]])
		}
		return 0
	}

	switch command {
	case "help", "--help", "-h", "":
		if command == "" && len(args) > 0 {
			return renderError(stdout, stderr, opts.json, "achta.error.v1", invalid("missing command"))
		}
		fmt.Fprint(stdout, helpText)
		return 0
	case "version":
		return runVersion(rest, stdout, stderr, opts, releaseVersion)
	case "capabilities":
		return runCapabilities(rest, stdout, stderr, opts, releaseVersion)
	case "claim":
		return runClaim(rest, stdout, stderr, opts)
	case "journal":
		return runJournal(rest, stdout, stderr, opts)
	case "count":
		return runCount(rest, stdout, stderr, opts)
	case "declared-route":
		return runDeclaredRoute(rest, stdout, stderr, opts)
	case "wiki":
		return runWiki(rest, stdout, stderr, opts)
	case "witness":
		return runWitness(rest, stdout, stderr, opts)
	case "reachability":
		return runReachability(rest, stdout, stderr, opts)
	case "toolchain":
		return runToolchain(rest, stdout, stderr, opts)
	case "amendments":
		return runAmendments(rest, stdout, stderr, opts)
	case "decision":
		return runDecision(rest, stdin, stdout, stderr, opts)
	case "slice":
		return runSlice(rest, stdout, stderr, opts)
	case "recipe":
		return runRecipe(rest, stdout, stderr, opts)
	case "ledger":
		return runLedger(rest, stdout, stderr, opts)
	case "floor":
		return runFloor(rest, stdout, stderr, opts)
	case "hook":
		return runHook(rest, stdin, stderr)
	case "mirror":
		return runMirror(rest, stdout, stderr, opts)
	case "parts":
		return runParts(rest, stdout, stderr, opts)
	case "replacement":
		return runReplacement(rest, stdout, stderr, opts)
	default:
		return renderError(stdout, stderr, opts.json, "achta.error.v1", invalid("unknown command %q", command))
	}
}

func commandHelpRequested(args []string) bool {
	if len(args) == 1 {
		return args[0] == "--help" || args[0] == "-h"
	}
	if len(args) == 2 {
		return args[1] == "--help" || args[1] == "-h"
	}
	return false
}

func parseGlobal(args []string) (globalOptions, string, []string, error) {
	var opts globalOptions
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--workspace":
			if i+1 >= len(args) || strings.TrimSpace(args[i+1]) == "" {
				return opts, "", nil, invalid("--workspace requires a path")
			}
			if opts.wikiDir != "" {
				return opts, "", nil, invalid("--workspace and --wiki-dir are mutually exclusive")
			}
			opts.workspace = args[i+1]
			i++
		case "--wiki-dir":
			if i+1 >= len(args) || strings.TrimSpace(args[i+1]) == "" {
				return opts, "", nil, invalid("--wiki-dir requires a path")
			}
			if opts.workspace != "" {
				return opts, "", nil, invalid("--workspace and --wiki-dir are mutually exclusive")
			}
			opts.wikiDir = args[i+1]
			i++
		case "--json":
			opts.json = true
		case "--quiet":
			opts.quiet = true
		case "--timing":
			opts.timing = true
		case "--help", "-h":
			return opts, "help", nil, nil
		default:
			return opts, args[i], args[i+1:], nil
		}
	}
	return opts, "", nil, nil
}

func jsonRequested(args []string) bool {
	for _, arg := range args {
		if arg == "--json" {
			return true
		}
	}
	return false
}

func appendElapsedJSON(document []byte, elapsedMS int64) ([]byte, error) {
	trimmed := bytes.TrimSpace(document)
	if !json.Valid(trimmed) || len(trimmed) < 2 || trimmed[0] != '{' || trimmed[len(trimmed)-1] != '}' {
		return nil, errors.New("command output is not one JSON object")
	}
	separator := byte(',')
	if len(trimmed) == 2 {
		separator = 0
	}
	result := make([]byte, 0, len(trimmed)+32)
	result = append(result, trimmed[:len(trimmed)-1]...)
	if separator != 0 {
		result = append(result, separator)
	}
	result = fmt.Appendf(result, "\"elapsed_ms\":%d}\n", elapsedMS)
	return result, nil
}

type versionDocument struct {
	SchemaVersion    string `json:"schema_version"`
	Version          string `json:"version"`
	ToolchainVersion string `json:"toolchain_version"`
	VersionAgreement string `json:"version_agreement"`
}

func runVersion(args []string, stdout, stderr io.Writer, opts globalOptions, releaseVersion string) int {
	opts, err := commandJSON(args, opts)
	if err != nil {
		return renderError(stdout, stderr, opts.json, versionSchema, err)
	}
	doc := currentVersion(releaseVersion)
	if opts.json {
		if writeJSON(stdout, stderr, doc) != 0 {
			return 2
		}
		if doc.VersionAgreement == "fail" {
			return 1
		}
		return 0
	}
	fmt.Fprintf(stdout, "achta %s (module %s, agreement %s)\n", doc.Version, doc.ToolchainVersion, doc.VersionAgreement)
	if doc.VersionAgreement == "fail" {
		return 1
	}
	return 0
}

func currentVersion(releaseVersion string) versionDocument {
	toolchainVersion := buildinfo.Version
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		toolchainVersion = info.Main.Version
	}
	return versionDocumentFor(releaseVersion, toolchainVersion)
}

func versionDocumentFor(releaseVersion, toolchainVersion string) versionDocument {
	releaseVersion = normalizeVersion(releaseVersion)
	agreement := "cannot_evaluate"
	if toolchainVersion != "unknown" {
		if normalizeVersion(toolchainVersion) == releaseVersion {
			agreement = "pass"
		} else {
			agreement = "fail"
		}
	}
	return versionDocument{SchemaVersion: versionSchema, Version: releaseVersion, ToolchainVersion: toolchainVersion, VersionAgreement: agreement}
}

func normalizeVersion(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || value == "dev" || value == "(devel)" {
		return "dev"
	}
	if !strings.HasPrefix(value, "v") {
		return "v" + value
	}
	return value
}

type capabilityCommand struct {
	Name              string   `json:"name"`
	Reads             bool     `json:"reads"`
	Writes            bool     `json:"writes"`
	ExecutesExternal  bool     `json:"executes_external"`
	RequiresGit       bool     `json:"requires_git"`
	RequiresWorkspace bool     `json:"requires_workspace"`
	InputForms        []string `json:"input_forms,omitempty"`
}

type capabilitiesDocument struct {
	SchemaVersion     string              `json:"schema_version"`
	Version           string              `json:"version"`
	MachineInterfaces []string            `json:"machine_interfaces"`
	GlobalOptions     []string            `json:"global_options"`
	Commands          []capabilityCommand `json:"commands"`
	ArtifactSchemas   []string            `json:"artifact_schemas"`
	RulefloorSchemas  []string            `json:"rulefloor_input_schemas"`
	SupportedOS       []string            `json:"supported_os"`
	Limitations       []string            `json:"limitations"`
}

func runCapabilities(args []string, stdout, stderr io.Writer, opts globalOptions, releaseVersion string) int {
	opts, err := commandJSON(args, opts)
	if err != nil {
		return renderError(stdout, stderr, opts.json, capabilitiesSchema, err)
	}
	commands := []capabilityCommand{
		{Name: "journal brief", Reads: true, ExecutesExternal: true, RequiresGit: true, RequiresWorkspace: true},
		{Name: "journal note", Reads: true, Writes: true, ExecutesExternal: true, RequiresGit: true, RequiresWorkspace: true},
		{Name: "hook journal", Reads: true, ExecutesExternal: true, RequiresGit: true},
		{Name: "gate run", Reads: true, Writes: true, ExecutesExternal: true, RequiresGit: true},
		{Name: "amendments declare", Reads: true, Writes: true, RequiresWorkspace: true},
		{Name: "amendments rebase", Reads: true, Writes: true, ExecutesExternal: true, RequiresGit: true, RequiresWorkspace: true},
		{Name: "amendments reconcile", Reads: true, ExecutesExternal: true, RequiresGit: true, RequiresWorkspace: true},
		{Name: "capabilities"},
		{Name: "claim take", Reads: true, Writes: true, ExecutesExternal: true, RequiresGit: true},
		{Name: "claim status", Reads: true, ExecutesExternal: true, RequiresGit: true},
		{Name: "claim check", Reads: true, ExecutesExternal: true, RequiresGit: true},
		{Name: "claim release", Reads: true, Writes: true, ExecutesExternal: true, RequiresGit: true},
		{Name: "count check", Reads: true, RequiresWorkspace: true},
		{Name: "declared-route check", Reads: true, RequiresWorkspace: true},
		{Name: "decision add", Reads: true, Writes: true, RequiresWorkspace: true, InputForms: []string{"--body-file PATH", "--body-file -"}},
		{Name: "reachability classify", Reads: true, ExecutesExternal: true, RequiresGit: true, RequiresWorkspace: true},
		{Name: "recipe check", Reads: true, RequiresWorkspace: true},
		{Name: "replacement check", Reads: true, RequiresWorkspace: true},
		{Name: "ledger census", Reads: true, RequiresWorkspace: true},
		{Name: "ledger rows", Reads: true, RequiresWorkspace: true},
		{Name: "mirror check", Reads: true, RequiresWorkspace: true},
		{Name: "parts lock", Reads: true, Writes: true, RequiresWorkspace: true},
		{Name: "parts verify", Reads: true, RequiresWorkspace: true},
		{Name: "floor census", Reads: true, ExecutesExternal: true, RequiresWorkspace: true},
		{Name: "hook cd", Reads: true},
		{Name: "slice check", Reads: true, ExecutesExternal: true, RequiresGit: true, RequiresWorkspace: true},
		{Name: "toolchain check", Reads: true, RequiresWorkspace: true},
		{Name: "version"},
		{Name: "wiki pin", Reads: true, Writes: true, ExecutesExternal: true, RequiresGit: true, RequiresWorkspace: true},
		{Name: "wiki freshness", Reads: true, ExecutesExternal: true, RequiresGit: true, RequiresWorkspace: true},
		{Name: "wiki unpushed", Reads: true, ExecutesExternal: true, RequiresGit: true, RequiresWorkspace: true},
		{Name: "wiki derive", Reads: true, Writes: true, ExecutesExternal: true, RequiresGit: true, RequiresWorkspace: true},
		{Name: "wiki check", Reads: true, ExecutesExternal: true, RequiresGit: true, RequiresWorkspace: true},
		{Name: "witness summarize", Reads: true, ExecutesExternal: true, RequiresGit: true, RequiresWorkspace: true},
		{Name: "witness init", Reads: true, Writes: true, ExecutesExternal: true, RequiresGit: true, RequiresWorkspace: true},
		{Name: "witness step", Reads: true, Writes: true, ExecutesExternal: true, RequiresGit: true, RequiresWorkspace: true},
		{Name: "witness finalize", Reads: true, Writes: true, ExecutesExternal: true, RequiresGit: true, RequiresWorkspace: true},
		{Name: "witness check", Reads: true, ExecutesExternal: true, RequiresGit: true, RequiresWorkspace: true},
		{Name: "witness earned", Reads: true, ExecutesExternal: true, RequiresGit: true, RequiresWorkspace: true},
	}
	sort.Slice(commands, func(i, j int) bool { return commands[i].Name < commands[j].Name })
	doc := capabilitiesDocument{
		SchemaVersion:     capabilitiesSchema,
		Version:           normalizeVersion(releaseVersion),
		MachineInterfaces: []string{capabilitiesSchema, versionSchema, "achta.wiki-pin.v1", achtawiki.FreshnessSchema, achtawiki.DeriveSchema, achtawiki.UnpushedSchema, wikiCheckSchema, decisionAddSchema, "achta.reachability.v1", "achta.toolchain-parity.v1", "achta.witness-summary.v1", witnessOperationSchema, witnessCheckSchema, "achta.witness-earned.v1", "achta.amendments-operation.v1", "achta.amendments-reconciliation.v1", "achta.slice-check.v1", recipe.Schema, census.Schema, ledgerrows.Schema, floorcensus.Schema, hookcd.Schema, mirror.Schema, "achta.parts-operation.v1", "achta.parts-verify.v1", countcheck.Schema, declaredroute.Schema, "achta.replacement-check.v1"},
		GlobalOptions:     []string{"--help", "--json", "--quiet", "--timing", "--wiki-dir", "--workspace"},
		Commands:          commands,
		ArtifactSchemas:   []string{"achta.parts-lock.v1", "achta.replacement-claims.v1", "achta.replacement-claims.v2", "achta.replacement-replay.v1", "achta.toolchain-manifest.v1", "gate-run.v1", "ledger-amendments.v1"},
		RulefloorSchemas:  []string{"rulefloor.capabilities.v1", "rulefloor.covers.v1", "rulefloor.ledger-diff.v1"},
		SupportedOS:       []string{"darwin", "linux"},
		Limitations: []string{
			"local filesystem only",
			"no implicit shell execution; gate run executes only caller-supplied argv",
			"no Git mutation except effects of the explicit gate run child",
		},
	}
	doc.MachineInterfaces = append(doc.MachineInterfaces, "achta.claim.v1")
	doc.MachineInterfaces = append(doc.MachineInterfaces, gateSchema)
	doc.MachineInterfaces = append(doc.MachineInterfaces, journalSchema, journalNoteSchema)
	doc.ArtifactSchemas = append(doc.ArtifactSchemas, "achta.claim-state.v1")
	if opts.json {
		return writeJSON(stdout, stderr, doc)
	}
	fmt.Fprintf(stdout, "Achta %s capabilities\n", doc.Version)
	for _, command := range doc.Commands {
		fmt.Fprintf(stdout, "  %s\n", command.Name)
		for _, form := range command.InputForms {
			fmt.Fprintf(stdout, "    %s\n", form)
		}
	}
	return 0
}

func commandJSON(args []string, opts globalOptions) (globalOptions, error) {
	for _, arg := range args {
		switch arg {
		case "--json":
			opts.json = true
		default:
			return opts, invalid("unexpected argument %q", arg)
		}
	}
	return opts, nil
}

func renderError(stdout, stderr io.Writer, jsonMode bool, schema string, err error) int {
	code := 2
	kind := "cannot_evaluate"
	var exitErr *ExitError
	if errors.As(err, &exitErr) {
		code = exitErr.Code
		kind = exitErr.Kind
	}
	if jsonMode {
		if writeJSON(stdout, stderr, machineError{SchemaVersion: schema, Status: kind, Error: sanitize(err.Error())}) != 0 {
			return 2
		}
		return code
	}
	fmt.Fprintf(stderr, "achta: %s\n", sanitize(err.Error()))
	return code
}

func writeJSON(stdout, stderr io.Writer, value any) int {
	encoder := json.NewEncoder(stdout)
	encoder.SetEscapeHTML(true)
	if err := encoder.Encode(value); err != nil {
		fmt.Fprintf(stderr, "achta: encode JSON: %v\n", err)
		return 2
	}
	return 0
}

func sanitize(value string) string {
	value = strings.Map(func(r rune) rune {
		if r < 0x20 && r != '\t' {
			return ' '
		}
		return r
	}, value)
	const max = 512
	if len(value) > max {
		return value[:max] + "..."
	}
	return value
}

const helpText = `Achta is a deterministic workspace governance and evidence tool.

Usage:
  achta [--workspace PATH | --wiki-dir PATH] [--json] [--quiet] [--timing] COMMAND [options]

Global options:
  --workspace PATH select the workspace root explicitly
  --wiki-dir PATH  select the workspace's wiki directory explicitly
  --json           emit one versioned JSON document
  --quiet          suppress non-problem human detail where supported
  --timing         append total elapsed milliseconds to the final output
  --help           show this help without requiring a workspace

Commands:
  gate run --repo ABS [--json] -- COMMAND [ARG...] keep a private log and short summary
  journal brief --workspace ABS [--json] show live claims, evidence and recent notes
  journal note --slice NAME TEXT [--workspace ABS] record an owner answer in Git metadata
  hook journal  add a brief on compact/resume; errors are silent and exit 0
  version       report release and Go module versions
  capabilities report supported machine interfaces and operations
  claim take --repo ABS --slice NAME [--note TEXT] claim a checkout
  claim status --repo ABS report its holder, tracked changes and local commits
  claim check --repo ABS --slice NAME check before writing
  claim release --repo ABS --slice NAME [--force --reason TEXT] release a claim
  count check   reconcile caller-declared breakdown totals and claim citations
  declared-route check enforce caller-declared alternatives, bans, and a YAML-scoped key
  decision add  allocate and insert an explicit decision
  reachability classify decide REQUIRED or SKIPPABLE from changed paths and declared no-reach patterns
  slice check   audit a landed slice from Git and wiki evidence
  recipe check  refuse neutralized make recipe lines; --expect-line is byte-exact
  replacement check require cited script-side replay before a named replacement or retirement claim
  ledger census recount a markdown ledger table against its totals and the files on disk
  ledger rows   enforce explicit state/prose marker relationships in markdown ledger rows
  floor census  recount a fenced completeness census against itself and rulefloor's covers map
  hook cd       require an absolute working-directory selector for recognized Bash writes
  mirror check  compare one master with mirrors and/or a recorded exact-byte SHA-256 digest
  parts lock    write the canonical versioned SHA-256 lock for direct .txt parts
  parts verify  compare every locked part and reject unlocked neighboring .txt files
  toolchain check compare declared workspace versions and script digests with CI pins
  wiki pin      update a reviewed repository verification pin
  wiki freshness enforce repository-page pins; --report-only reports drift
  wiki derive   preview generated facts by default; --write applies changes
  wiki unpushed compare a repository with its local upstream ref
  wiki check    report freshness and derived-block checks separately; --only NAME[,NAME] selects a subset
  witness summarize summarize a gate-run.v1 record
  witness init  create or restart a bounded witness record
  witness step  append one explicitly observed target result
  witness finalize close a witness and bind a green result to the tree
  witness check validate a complete green witness against the current tree
  witness earned refuse a new witness cycle when only witness machinery changed
  amendments declare add an explicit ledger change declaration
  amendments rebase move a manifest to an accepted witness commit
  amendments reconcile compare declarations with Rulefloor's logical diff
  help          show this help

Claims are advisory and local; no expiry or network calls. All four accept --json.
Example with two slices:
  achta claim take --repo /work/project --slice first --note 'database work'
  achta claim check --repo /work/project --slice second
  achta claim release --repo /work/project --slice first
The second call refuses with the holder and age. --force requires --reason TEXT
and records the releasing slice. Status lists tracked changes and local commits;
an unavailable upstream is unknown and an unclaimed check refuses it.
`

// scopedCommandHelp documents side effects and exit semantics at the command.
var scopedCommandHelp = map[string]string{
	"journal brief": "\njournal brief --workspace ABS [--json]\nSeven live category lines; local origin refs only, dirty includes untracked files.\n",
	"journal note":  "\njournal note [--workspace ABS] --slice NAME TEXT\nFlags precede TEXT. Writes each claimed checkout's private Git journal; retains 200 lines.\n",
	"decision add": `
decision add --title TITLE --body-file PATH|- [--prefix PREFIX] [--register PATH] [--check] [--json]
--body-file - reads the body from stdin; PATH must remain inside the workspace.
Bodies must contain non-whitespace text, at most 65536 bytes, and no register headings.
--prefix selects the ID series and the section that already holds it (default P).
For a register with D decisions in Identity and P decisions in Platform:
  achta decision add --prefix D --title "Identity choice" --body-file -
  achta decision add --prefix P --title "Platform choice" --body-file body.md
Sections come from the existing register; the prefix does not create a section.
--register defaults to wiki/platform/decisions.md. --check validates without writing
and exits 1 for would_change; a successful write exits 0, invalid input exits 2.
`,
	"wiki check": `
wiki check [--only freshness,derive] [--repo NAME]... [--exclude NAME=REASON]... [--json]
Without --only both checks run; --only selects check names, not repository names.
--repo selects repository pages to judge; other pages are NOT judged and named.
--exclude requires a reason, reported for each NOT judged page in every selected check.
Unknown, duplicate or conflicting selectors exit 2 before judging. Without repository
selectors all pages are judged. Selected judgements are unchanged: 0 pass, 1 fail,
2 cannot_evaluate. --wiki-dir is a global option, before wiki.
`,
	"wiki freshness": `
wiki freshness [--repo NAME] [--strict | --report-only] [--json]
Enforcing by default (--strict is accepted): drift exits 1; unavailable evidence
exits 2. --report-only explicitly reports its mode, retains drift status, and
exits 0 for evaluated drift; unavailable evidence still exits 2. No files written.
`,
	"wiki derive": `
wiki derive [--check | --write | --print REPOSITORY] [--json]
Default and --check preview changed pages with before/after bytes; no files written.
A stale preview exits 1, unchanged exits 0, unavailable evidence exits 2.
--write applies blocks and names updated pages; --print prints one generated block.
The three explicit modes are mutually exclusive. Headers name Achta and its version.
`,
	"wiki pin": `
wiki pin REPOSITORY --sha FULL_HEAD_SHA --verified YYYY-MM-DD --attest-reviewed [--check] [--json]
Moves verified_against, verified and updated together. Adds updated if absent.
Preserves DERIVED content, lead text and other bytes. A new front lead is the PM's
 to write. --check writes nothing (1 would_change, 0 unchanged); missing attestation
or an already-current write exits 1; unavailable or malformed input exits 2.
`,
	"recipe check": `
recipe check --makefile PATH --target NAME [--expect-line TEXT]... [--expect-file PATH] [--expect-order] [--forbid-noop] [--json]
Read-only. Expectations are byte-exact membership by default. --expect-order
requires them as an ordered subsequence, matching each occurrence once (repeatable
--expect-line values first, then file lines). Other recipe lines remain checked.
Reordered or absent expectations and neutralizers exit 1; unavailable/invalid
input exits 2. --expect-order without expectations is invalid.
`,
}

package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"runtime/debug"
	"sort"
	"strings"
	"time"

	"github.com/ozgurcd/achta/internal/buildinfo"
	"github.com/ozgurcd/achta/internal/census"
	"github.com/ozgurcd/achta/internal/countcheck"
	"github.com/ozgurcd/achta/internal/declaredroute"
	"github.com/ozgurcd/achta/internal/floorcensus"
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
	started := time.Now()
	opts, command, rest, err := parseGlobal(args)
	jsonMode := jsonRequested(args)
	if !opts.timing && (!opts.quiet || jsonMode) {
		return dispatch(args, stdout, stderr, releaseVersion, opts, command, rest, err)
	}
	if jsonMode {
		opts.json = true
	}

	var commandStdout, commandStderr bytes.Buffer
	code := dispatch(args, &commandStdout, &commandStderr, releaseVersion, opts, command, rest, err)
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

func dispatch(args []string, stdout, stderr io.Writer, releaseVersion string, opts globalOptions, command string, rest []string, err error) int {
	if err != nil {
		return renderError(stdout, stderr, opts.json, "achta.error.v1", err)
	}
	if command != "" && command != "help" && commandHelpRequested(rest) {
		fmt.Fprint(stdout, helpText)
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
		return runDecision(rest, stdout, stderr, opts)
	case "slice":
		return runSlice(rest, stdout, stderr, opts)
	case "recipe":
		return runRecipe(rest, stdout, stderr, opts)
	case "ledger":
		return runLedger(rest, stdout, stderr, opts)
	case "floor":
		return runFloor(rest, stdout, stderr, opts)
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
	Name              string `json:"name"`
	Reads             bool   `json:"reads"`
	Writes            bool   `json:"writes"`
	ExecutesExternal  bool   `json:"executes_external"`
	RequiresGit       bool   `json:"requires_git"`
	RequiresWorkspace bool   `json:"requires_workspace"`
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
		{Name: "amendments declare", Reads: true, Writes: true, RequiresWorkspace: true},
		{Name: "amendments rebase", Reads: true, Writes: true, ExecutesExternal: true, RequiresGit: true, RequiresWorkspace: true},
		{Name: "amendments reconcile", Reads: true, ExecutesExternal: true, RequiresGit: true, RequiresWorkspace: true},
		{Name: "capabilities"},
		{Name: "count check", Reads: true, RequiresWorkspace: true},
		{Name: "declared-route check", Reads: true, RequiresWorkspace: true},
		{Name: "decision add", Reads: true, Writes: true, RequiresWorkspace: true},
		{Name: "reachability classify", Reads: true, ExecutesExternal: true, RequiresGit: true, RequiresWorkspace: true},
		{Name: "recipe check", Reads: true, RequiresWorkspace: true},
		{Name: "replacement check", Reads: true, RequiresWorkspace: true},
		{Name: "ledger census", Reads: true, RequiresWorkspace: true},
		{Name: "ledger rows", Reads: true, RequiresWorkspace: true},
		{Name: "mirror check", Reads: true, RequiresWorkspace: true},
		{Name: "parts lock", Reads: true, Writes: true, RequiresWorkspace: true},
		{Name: "parts verify", Reads: true, RequiresWorkspace: true},
		{Name: "floor census", Reads: true, ExecutesExternal: true, RequiresWorkspace: true},
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
		MachineInterfaces: []string{capabilitiesSchema, versionSchema, "achta.wiki-pin.v1", achtawiki.FreshnessSchema, achtawiki.DeriveSchema, achtawiki.UnpushedSchema, wikiCheckSchema, decisionAddSchema, "achta.reachability.v1", "achta.toolchain-parity.v1", "achta.witness-summary.v1", witnessOperationSchema, witnessCheckSchema, "achta.witness-earned.v1", "achta.amendments-operation.v1", "achta.amendments-reconciliation.v1", "achta.slice-check.v1", recipe.Schema, census.Schema, ledgerrows.Schema, floorcensus.Schema, mirror.Schema, "achta.parts-operation.v1", "achta.parts-verify.v1", countcheck.Schema, declaredroute.Schema, "achta.replacement-check.v1"},
		GlobalOptions:     []string{"--help", "--json", "--quiet", "--timing", "--wiki-dir", "--workspace"},
		Commands:          commands,
		ArtifactSchemas:   []string{"achta.parts-lock.v1", "achta.replacement-claims.v1", "achta.toolchain-manifest.v1", "gate-run.v1", "ledger-amendments.v1"},
		RulefloorSchemas:  []string{"rulefloor.capabilities.v1", "rulefloor.covers.v1", "rulefloor.ledger-diff.v1"},
		SupportedOS:       []string{"darwin", "linux"},
		Limitations: []string{
			"local filesystem only",
			"no shell execution",
			"no Git mutation",
		},
	}
	if opts.json {
		return writeJSON(stdout, stderr, doc)
	}
	fmt.Fprintf(stdout, "Achta %s capabilities\n", doc.Version)
	for _, command := range doc.Commands {
		fmt.Fprintf(stdout, "  %s\n", command.Name)
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
  version       report release and Go module versions
  capabilities report supported machine interfaces and operations
  count check   reconcile caller-declared breakdown totals and claim citations
  declared-route check enforce caller-declared alternatives, bans, and a YAML-scoped key
  decision add  allocate and insert an explicit platform decision
  reachability classify decide REQUIRED or SKIPPABLE from changed paths and declared no-reach patterns
  slice check   audit a landed slice from Git and wiki evidence
  recipe check  refuse neutralized make recipe lines; --expect-line is byte-exact
  replacement check require cited script-side replay before a named replacement or retirement claim
  ledger census recount a markdown ledger table against its totals and the files on disk
  ledger rows   enforce explicit state/prose marker relationships in markdown ledger rows
  floor census  recount a fenced completeness census against itself and rulefloor's covers map
  mirror check  compare one master with mirrors and/or a recorded exact-byte SHA-256 digest
  parts lock    write the canonical versioned SHA-256 lock for direct .txt parts
  parts verify  compare every locked part and reject unlocked neighboring .txt files
  toolchain check compare declared workspace versions and script digests with CI pins
  wiki pin      update a reviewed repository verification pin
  wiki freshness compare repository-page pins with local HEADs
  wiki derive   check, print, or update generated repository facts
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
`

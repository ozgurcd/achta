package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ozgurcd/achta/internal/gitstate"
	"github.com/ozgurcd/achta/internal/safefile"
	"github.com/ozgurcd/achta/internal/witness"
	"github.com/ozgurcd/achta/internal/workspace"
)

const (
	witnessOperationSchema = "achta.witness-operation.v1"
	witnessCheckSchema     = "achta.witness-check.v1"
)

type witnessOperationDocument struct {
	SchemaVersion string `json:"schema_version"`
	Operation     string `json:"operation"`
	Status        string `json:"status"`
	Record        string `json:"record"`
	Target        string `json:"target,omitempty"`
	ExitCode      *int   `json:"exit_code,omitempty"`
	Changed       bool   `json:"changed"`
}

type witnessCheckDocument struct {
	SchemaVersion string          `json:"schema_version"`
	Status        string          `json:"status"`
	Summary       witness.Summary `json:"summary"`
}

func runWitnessInit(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	set := flagSet("witness init")
	repoValue := set.String("repo", "", "repository path")
	recordValue := set.String("record", "", "gate-run.v1 record path")
	label := set.String("label", "", "gate label")
	targetValues := set.String("targets", "", "comma-separated target names")
	jsonMode := set.Bool("json", opts.json, "emit JSON")
	if err := parseFlags(set, args); err != nil {
		opts.json = *jsonMode
		return renderError(stdout, stderr, opts.json, witnessOperationSchema, err)
	}
	opts.json = *jsonMode
	if *label == "" || *targetValues == "" {
		return renderError(stdout, stderr, opts.json, witnessOperationSchema, invalid("--label and --targets are required"))
	}
	ws, repo, recordPath, relative, err := resolveWitnessPaths(opts, *repoValue, *recordValue)
	if err != nil {
		return renderError(stdout, stderr, opts.json, witnessOperationSchema, err)
	}
	clean, err := gitstate.IsCleanExcept(repo, relative)
	if err != nil {
		return renderError(stdout, stderr, opts.json, witnessOperationSchema, invalid("repository status: %v", err))
	}
	if !clean {
		return renderError(stdout, stderr, opts.json, witnessOperationSchema, mismatch("repository has changes outside the witness record"))
	}
	head, err := gitstate.Head(repo)
	if err != nil {
		return renderError(stdout, stderr, opts.json, witnessOperationSchema, invalid("repository HEAD: %v", err))
	}
	targets := strings.Split(*targetValues, ",")
	data, err := witness.Init(*label, head, targets, time.Now())
	if err != nil {
		return renderError(stdout, stderr, opts.json, witnessOperationSchema, invalid("initialize witness: %v", err))
	}
	if err := replaceOrCreate(ws.Root, recordPath, data); err != nil {
		return renderError(stdout, stderr, opts.json, witnessOperationSchema, invalid("write witness: %v", err))
	}
	return renderWitnessOperation(stdout, stderr, opts, witnessOperationDocument{SchemaVersion: witnessOperationSchema, Operation: "init", Status: "initialized", Record: *recordValue, Changed: true}, 0)
}

func runWitnessStep(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	const unsetElapsed = int64(-1 << 63)
	set := flagSet("witness step")
	repoValue := set.String("repo", "", "repository path")
	recordValue := set.String("record", "", "gate-run.v1 record path")
	target := set.String("target", "", "planned target name")
	exitCode := set.Int("exit-code", -1<<30, "observed target exit code")
	elapsedMS := set.Int64("elapsed-ms", unsetElapsed, "observed target elapsed milliseconds")
	evidenceValue := set.String("evidence-file", "", "optional bounded evidence lines")
	jsonMode := set.Bool("json", opts.json, "emit JSON")
	if err := parseFlags(set, args); err != nil {
		opts.json = *jsonMode
		return renderError(stdout, stderr, opts.json, witnessOperationSchema, err)
	}
	opts.json = *jsonMode
	if *target == "" || *exitCode == -1<<30 {
		return renderError(stdout, stderr, opts.json, witnessOperationSchema, invalid("--target and --exit-code are required"))
	}
	ws, repo, recordPath, relative, err := resolveWitnessPaths(opts, *repoValue, *recordValue)
	if err != nil {
		return renderError(stdout, stderr, opts.json, witnessOperationSchema, err)
	}
	snapshot, record, err := readOpenWitness(ws.Root, repo, recordPath, relative)
	if err != nil {
		return renderError(stdout, stderr, opts.json, witnessOperationSchema, err)
	}
	var evidence []string
	if *evidenceValue != "" {
		if err := rejectSecretLikeInput(*evidenceValue); err != nil {
			return renderError(stdout, stderr, opts.json, witnessOperationSchema, err)
		}
		evidencePath, err := confinedPath(ws, *evidenceValue)
		if err != nil {
			return renderError(stdout, stderr, opts.json, witnessOperationSchema, err)
		}
		evidenceSnapshot, err := safefile.Read(ws.Root, evidencePath, 1<<20)
		if err != nil {
			return renderError(stdout, stderr, opts.json, witnessOperationSchema, invalid("read evidence: %v", err))
		}
		evidence, err = witness.Evidence(evidenceSnapshot.Data)
		if err != nil {
			return renderError(stdout, stderr, opts.json, witnessOperationSchema, invalid("parse evidence: %v", err))
		}
	}
	var elapsed *int64
	if *elapsedMS != unsetElapsed {
		elapsed = elapsedMS
	}
	data, err := witness.Step(snapshot.Data, *target, *exitCode, elapsed, evidence)
	if err != nil {
		if errors.Is(err, witness.ErrOperationRefused) {
			return renderError(stdout, stderr, opts.json, witnessOperationSchema, mismatch("record witness step: %v", err))
		}
		return renderError(stdout, stderr, opts.json, witnessOperationSchema, invalid("record witness step: %v", err))
	}
	if record.RepoDirty {
		return renderError(stdout, stderr, opts.json, witnessOperationSchema, mismatch("witness was initialized on a dirty repository"))
	}
	if err := snapshot.Replace(data, witness.MaxRecord); err != nil {
		return renderError(stdout, stderr, opts.json, witnessOperationSchema, invalid("replace witness: %v", err))
	}
	code, status := 0, "recorded"
	if *exitCode != 0 {
		code, status = 1, "red"
	}
	return renderWitnessOperation(stdout, stderr, opts, witnessOperationDocument{SchemaVersion: witnessOperationSchema, Operation: "step", Status: status, Record: *recordValue, Target: *target, ExitCode: exitCode, Changed: true}, code)
}

func runWitnessFinalize(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	set := flagSet("witness finalize")
	repoValue := set.String("repo", "", "repository path")
	recordValue := set.String("record", "", "gate-run.v1 record path")
	commitTie := set.Bool("commit-tie", false, "tie a clean CI record to HEAD instead of a tree digest")
	jsonMode := set.Bool("json", opts.json, "emit JSON")
	if err := parseFlags(set, args); err != nil {
		opts.json = *jsonMode
		return renderError(stdout, stderr, opts.json, witnessOperationSchema, err)
	}
	opts.json = *jsonMode
	ws, repo, recordPath, relative, err := resolveWitnessPaths(opts, *repoValue, *recordValue)
	if err != nil {
		return renderError(stdout, stderr, opts.json, witnessOperationSchema, err)
	}
	snapshot, record, err := readOpenWitness(ws.Root, repo, recordPath, relative)
	if err != nil {
		return renderError(stdout, stderr, opts.json, witnessOperationSchema, err)
	}
	if record.RepoDirty {
		return renderError(stdout, stderr, opts.json, witnessOperationSchema, mismatch("witness was initialized on a dirty repository"))
	}
	treeKind := "sha256"
	treeValue, err := gitstate.TreeDigest(repo, relative)
	if *commitTie {
		treeKind = "commit"
		clean, cleanErr := gitstate.IsClean(repo)
		if cleanErr != nil {
			err = cleanErr
		} else if !clean {
			return renderError(stdout, stderr, opts.json, witnessOperationSchema, mismatch("commit-tied finalization requires a completely clean repository"))
		} else {
			treeValue, err = gitstate.Head(repo)
		}
	}
	if err != nil {
		return renderError(stdout, stderr, opts.json, witnessOperationSchema, invalid("calculate witness tie: %v", err))
	}
	data, err := witness.Finalize(snapshot.Data, treeKind, treeValue, time.Now())
	if err != nil {
		if errors.Is(err, witness.ErrOperationRefused) {
			return renderError(stdout, stderr, opts.json, witnessOperationSchema, mismatch("finalize witness: %v", err))
		}
		return renderError(stdout, stderr, opts.json, witnessOperationSchema, invalid("finalize witness: %v", err))
	}
	parsed, err := witness.Parse(data)
	if err != nil {
		return renderError(stdout, stderr, opts.json, witnessOperationSchema, invalid("validate finalized witness: %v", err))
	}
	if err := snapshot.Replace(data, witness.MaxRecord); err != nil {
		return renderError(stdout, stderr, opts.json, witnessOperationSchema, invalid("replace witness: %v", err))
	}
	code, status := 0, parsed.Result
	if status != "green" {
		code = 1
	}
	return renderWitnessOperation(stdout, stderr, opts, witnessOperationDocument{SchemaVersion: witnessOperationSchema, Operation: "finalize", Status: status, Record: *recordValue, Changed: true}, code)
}

func runWitnessCheck(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	set := flagSet("witness check")
	repoValue := set.String("repo", "", "repository path")
	recordValue := set.String("record", "", "gate-run.v1 record path")
	jsonMode := set.Bool("json", opts.json, "emit JSON")
	if err := parseFlags(set, args); err != nil {
		opts.json = *jsonMode
		return renderError(stdout, stderr, opts.json, witnessCheckSchema, err)
	}
	opts.json = *jsonMode
	ws, repo, recordPath, _, err := resolveWitnessPaths(opts, *repoValue, *recordValue)
	if err != nil {
		return renderError(stdout, stderr, opts.json, witnessCheckSchema, err)
	}
	snapshot, err := safefile.Read(ws.Root, recordPath, witness.MaxRecord)
	if err != nil {
		return renderError(stdout, stderr, opts.json, witnessCheckSchema, invalid("read witness: %v", err))
	}
	record, err := witness.Parse(snapshot.Data)
	if err != nil {
		return renderError(stdout, stderr, opts.json, witnessCheckSchema, invalid("parse witness: %v", err))
	}
	summary, err := witness.Summarize(record, repo, recordPath, "", 0)
	if err != nil {
		return renderError(stdout, stderr, opts.json, witnessCheckSchema, invalid("check witness: %v", err))
	}
	status, code := "pass", 0
	if record.RepoDirty || summary.Status != "green" || summary.Completeness != "complete" || summary.Freshness != "current" {
		status, code = "fail", 1
	}
	document := witnessCheckDocument{SchemaVersion: witnessCheckSchema, Status: status, Summary: summary}
	if opts.json {
		if writeJSON(stdout, stderr, document) != 0 {
			return 2
		}
		return code
	}
	fmt.Fprintf(stdout, "witness check %s: %s; %s, %s, %s\n", baseName(recordPath), status, summary.Status, summary.Completeness, summary.Freshness)
	return code
}

func resolveWitnessPaths(opts globalOptions, repoValue, recordValue string) (workspace.Workspace, string, string, string, error) {
	if repoValue == "" || recordValue == "" {
		return workspace.Workspace{}, "", "", "", invalid("--repo and --record are required")
	}
	ws, err := resolveWorkspace(opts)
	if err != nil {
		return workspace.Workspace{}, "", "", "", err
	}
	repo, err := confinedPath(ws, repoValue)
	if err != nil {
		return workspace.Workspace{}, "", "", "", err
	}
	recordPath, err := confinedPath(ws, recordValue)
	if err != nil {
		return workspace.Workspace{}, "", "", "", err
	}
	relative, err := filepath.Rel(repo, recordPath)
	if err != nil || relative == ".." || filepath.IsAbs(relative) || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return workspace.Workspace{}, "", "", "", invalid("witness record must be inside the selected repository")
	}
	return ws, repo, recordPath, relative, nil
}

func readOpenWitness(root, repo, recordPath, relative string) (safefile.Snapshot, witness.Record, error) {
	snapshot, err := safefile.Read(root, recordPath, witness.MaxRecord)
	if err != nil {
		return safefile.Snapshot{}, witness.Record{}, invalid("read witness: %v", err)
	}
	record, err := witness.Parse(snapshot.Data)
	if err != nil {
		return safefile.Snapshot{}, witness.Record{}, invalid("parse witness: %v", err)
	}
	head, err := gitstate.Head(repo)
	if err != nil {
		return safefile.Snapshot{}, witness.Record{}, invalid("repository HEAD: %v", err)
	}
	resolved, err := gitstate.ResolveCommit(repo, record.RepoHead)
	if err != nil || resolved != head {
		return safefile.Snapshot{}, witness.Record{}, mismatch("witness repository HEAD is stale")
	}
	clean, err := gitstate.IsCleanExcept(repo, relative)
	if err != nil {
		return safefile.Snapshot{}, witness.Record{}, invalid("repository status: %v", err)
	}
	if !clean {
		return safefile.Snapshot{}, witness.Record{}, mismatch("repository has changes outside the witness record")
	}
	return snapshot, record, nil
}

func replaceOrCreate(root, path string, data []byte) error {
	snapshot, err := safefile.Read(root, path, witness.MaxRecord)
	if err == nil {
		if _, parseErr := witness.Parse(snapshot.Data); parseErr != nil {
			return fmt.Errorf("parse existing witness: %w", parseErr)
		}
		return snapshot.Replace(data, witness.MaxRecord)
	}
	if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return safefile.Create(root, path, data, witness.MaxRecord, 0o644)
}

func renderWitnessOperation(stdout, stderr io.Writer, opts globalOptions, document witnessOperationDocument, code int) int {
	if opts.json {
		if writeJSON(stdout, stderr, document) != 0 {
			return 2
		}
		return code
	}
	fmt.Fprintf(stdout, "witness %s %s: %s\n", document.Operation, document.Record, document.Status)
	return code
}

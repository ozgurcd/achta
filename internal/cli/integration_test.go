package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ozgurcd/achta/internal/amendments"
	"github.com/ozgurcd/achta/internal/gitstate"
)

// RULE: WIKI-PIN-1
func TestWikiPinWorkflowAndAttestationRefusal(t *testing.T) {
	root, repo, head := testWorkspaceRepo(t, "sample")
	page := strings.ReplaceAll(pageFixtureCLI, "REPOSITORY", "sample")
	pagePath := filepath.Join(root, "wiki", "repos", "sample.md")
	writeTestFile(t, pagePath, page)
	original := readTestFile(t, pagePath)

	var stdout, stderr bytes.Buffer
	args := []string{"--workspace", root, "wiki", "pin", "sample", "--sha", head, "--verified", "2026-09-04", "--json"}
	if code := Run(args, &stdout, &stderr, "v0.1.0"); code != 1 {
		t.Fatalf("missing attestation code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if got := readTestFile(t, pagePath); got != original {
		t.Fatal("refused pin changed page bytes")
	}

	stdout.Reset()
	stderr.Reset()
	checkArgs := append(append([]string{}, args...), "--attest-reviewed", "--check")
	if code := Run(checkArgs, &stdout, &stderr, "v0.1.0"); code != 1 {
		t.Fatalf("pin check code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if got := readTestFile(t, pagePath); got != original {
		t.Fatal("pin check changed page bytes")
	}

	stdout.Reset()
	stderr.Reset()
	args = append(args, "--attest-reviewed")
	if code := Run(args, &stdout, &stderr, "v0.1.0"); code != 0 {
		t.Fatalf("pin code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	got := readTestFile(t, pagePath)
	if !strings.Contains(got, "verified_against: sample @ "+head+" (main). Keep this suffix.") || !strings.Contains(got, "| Repo HEAD | "+head[:7]+" (main) |") {
		t.Fatalf("page not pinned correctly:\n%s", got)
	}
	if _, err := os.Stat(repo); err != nil {
		t.Fatal(err)
	}

	stdout.Reset()
	stderr.Reset()
	if code := Run([]string{"--workspace", root, "wiki", "freshness", "--repo", "sample", "--strict", "--json"}, &stdout, &stderr, "v0.1.0"); code != 0 {
		t.Fatalf("freshness code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), `"status":"pass"`) {
		t.Fatalf("freshness did not pass: %s", stdout.String())
	}

	beforeNoop := readTestFile(t, pagePath)
	stdout.Reset()
	stderr.Reset()
	if code := Run(args, &stdout, &stderr, "v0.1.0"); code != 1 {
		t.Fatalf("pin no-op code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if got := readTestFile(t, pagePath); got != beforeNoop {
		t.Fatal("pin no-op changed page bytes")
	}
}

// RULE: WIKI-DIR-1
func TestWikiDirSelectsWikiAndRejectsWorkspaceConflict(t *testing.T) {
	root, _, _ := testWorkspaceRepo(t, "sample")
	var stdout, stderr bytes.Buffer
	if code := Run([]string{"--wiki-dir", filepath.Join(root, "wiki"), "wiki", "freshness", "--json"}, &stdout, &stderr, "v0.1.0"); code != 0 {
		t.Fatalf("wiki-dir code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "achta.wiki-freshness.v1") {
		t.Fatalf("wiki-dir did not run freshness: %s", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	conflictArgs := []string{"--json", "--workspace", root, "--wiki-dir", filepath.Join(root, "wiki"), "wiki", "freshness"}
	if code := Run(conflictArgs, &stdout, &stderr, "v0.1.0"); code != 2 {
		t.Fatalf("conflicting selectors code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "mutually exclusive") {
		t.Fatalf("conflicting selectors did not explain the error: %s", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := Run([]string{"--json", "--wiki-dir", root, "wiki", "freshness"}, &stdout, &stderr, "v0.1.0"); code != 2 {
		t.Fatalf("workspace-as-wiki code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "direct wiki child") {
		t.Fatalf("workspace-as-wiki did not explain the layout requirement: %s", stdout.String())
	}
}

func TestDecisionAddCheckThenWrite(t *testing.T) {
	root, _, _ := testWorkspaceRepo(t, "sample")
	registerPath := filepath.Join(root, "wiki", "platform", "decisions.md")
	bodyPath := filepath.Join(root, "decision-body.md")
	register := "# Decisions\n\n## Platform\n\n### P-001 — first choice\nExisting body.\n\n## UI\n"
	writeTestFile(t, registerPath, register)
	writeTestFile(t, bodyPath, "Measured owner choice.\n")

	var stdout, stderr bytes.Buffer
	args := []string{"--workspace", root, "decision", "add", "--title", "second choice", "--body-file", "decision-body.md", "--check", "--json"}
	if code := Run(args, &stdout, &stderr, "v0.1.0"); code != 1 {
		t.Fatalf("check code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	assertGolden(t, "decision-add.json", stdout.String())
	if got := readTestFile(t, registerPath); got != register {
		t.Fatal("decision check changed register")
	}

	stdout.Reset()
	stderr.Reset()
	args = []string{"--workspace", root, "decision", "add", "--title", "second choice", "--body-file", "decision-body.md", "--json"}
	if code := Run(args, &stdout, &stderr, "v0.1.0"); code != 0 {
		t.Fatalf("write code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	got := readTestFile(t, registerPath)
	if !strings.Contains(got, "### P-002 — second choice\nMeasured owner choice.\n\n## UI") {
		t.Fatalf("decision was not inserted at the section boundary:\n%s", got)
	}
}

func TestWitnessLifecycleRecordsWithoutExecutingCommands(t *testing.T) {
	for _, cycle := range []string{"first", "second"} {
		t.Run(cycle, func(t *testing.T) {
			root, repo, _ := testWorkspaceRepo(t, "sample")
			record := filepath.Join("sample", "GATE-RUN.txt")
			var stdout, stderr bytes.Buffer
			if code := Run([]string{"--workspace", root, "witness", "init", "--repo", "sample", "--record", record, "--label", "fixture gate", "--targets", "build,test", "--json"}, &stdout, &stderr, "v0.2.0"); code != 0 {
				t.Fatalf("init code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
			}
			if cycle == "first" {
				stdout.Reset()
				stderr.Reset()
				if code := Run([]string{"--workspace", root, "witness", "step", "--repo", "sample", "--record", record, "--target", "unplanned", "--exit-code", "0", "--json"}, &stdout, &stderr, "v0.2.0"); code != 1 {
					t.Fatalf("unplanned target refusal code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
				}
				stdout.Reset()
				stderr.Reset()
				if code := Run([]string{"--workspace", root, "witness", "step", "--repo", "sample", "--record", record, "--target", "build", "--exit-code", "0", "--elapsed-ms", "-1", "--json"}, &stdout, &stderr, "v0.2.0"); code != 2 {
					t.Fatalf("negative elapsed code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
				}
			}
			stdout.Reset()
			stderr.Reset()
			if code := Run([]string{"--workspace", root, "witness", "step", "--repo", "sample", "--record", record, "--target", "build", "--exit-code", "0", "--elapsed-ms", "12", "--json"}, &stdout, &stderr, "v0.2.0"); code != 0 {
				t.Fatalf("step build code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
			}
			if cycle == "first" {
				stdout.Reset()
				stderr.Reset()
				if code := Run([]string{"--workspace", root, "witness", "step", "--repo", "sample", "--record", record, "--target", "build", "--exit-code", "0", "--json"}, &stdout, &stderr, "v0.2.0"); code != 1 {
					t.Fatalf("duplicate target refusal code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
				}
			}
			stdout.Reset()
			stderr.Reset()
			if code := Run([]string{"--workspace", root, "witness", "step", "--repo", "sample", "--record", record, "--target", "test", "--exit-code", "0", "--json"}, &stdout, &stderr, "v0.2.0"); code != 0 {
				t.Fatalf("step test code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
			}
			stdout.Reset()
			stderr.Reset()
			if code := Run([]string{"--workspace", root, "witness", "finalize", "--repo", "sample", "--record", record, "--json"}, &stdout, &stderr, "v0.2.0"); code != 0 {
				t.Fatalf("finalize code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
			}
			stdout.Reset()
			stderr.Reset()
			if code := Run([]string{"--workspace", root, "witness", "check", "--repo", "sample", "--record", record, "--json"}, &stdout, &stderr, "v0.2.0"); code != 0 {
				t.Fatalf("check code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
			}
			if !strings.Contains(stdout.String(), `"schema_version":"achta.witness-check.v1"`) || !strings.Contains(stdout.String(), `"status":"pass"`) {
				t.Fatalf("unexpected check JSON: %s", stdout.String())
			}
			if _, err := os.Stat(filepath.Join(repo, "GATE-RUN.txt")); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestWitnessInitRefusesMalformedExistingRecord(t *testing.T) {
	root, repo, _ := testWorkspaceRepo(t, "sample")
	recordPath := filepath.Join(repo, "GATE-RUN.txt")
	writeTestFile(t, recordPath, "malformed witness\n")
	original := readTestFile(t, recordPath)
	var stdout, stderr bytes.Buffer
	args := []string{"--workspace", root, "witness", "init", "--repo", "sample", "--record", "sample/GATE-RUN.txt", "--label", "fixture gate", "--targets", "build", "--json"}
	if code := Run(args, &stdout, &stderr, "v0.2.0"); code != 2 {
		t.Fatalf("malformed existing witness code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if got := readTestFile(t, recordPath); got != original {
		t.Fatal("refused witness initialization changed existing bytes")
	}
}

// RULE: SLICE-CHECK-1
func TestSliceCheckAuditsLandedCommit(t *testing.T) {
	root, repo, _ := testWorkspaceRepo(t, "sample")
	if err := os.MkdirAll(filepath.Join(repo, "log"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, filepath.Join(repo, "log", "001-baseline.md"), "## [2026-09-04] baseline\n")
	runGit(t, repo, "add", "log/001-baseline.md")
	runGit(t, repo, "commit", "-m", "log baseline")
	base := strings.TrimSpace(runGit(t, repo, "rev-parse", "HEAD"))
	runGit(t, repo, "remote", "add", "origin", repo)
	runGit(t, repo, "update-ref", "refs/remotes/origin/main", base)
	runGit(t, repo, "branch", "--set-upstream-to=origin/main", "main")
	writeTestFile(t, filepath.Join(repo, "slice.txt"), "landed slice\n")
	writeTestFile(t, filepath.Join(repo, "log", "002-slice.md"), "## [2026-09-05] slice\n")
	runGit(t, repo, "add", "slice.txt", "log/002-slice.md")
	runGit(t, repo, "commit", "-m", "fixture slice")
	head := strings.TrimSpace(runGit(t, repo, "rev-parse", "HEAD"))
	page := "---\ntitle: sample\ncategory: repo\nverified: 2026-09-04\nverified_against: sample @ " + head + " (main)\n---\n"
	writeTestFile(t, filepath.Join(root, "wiki", "repos", "sample.md"), page)

	var stdout, stderr bytes.Buffer
	args := []string{"--workspace", root, "slice", "check", "--repo", "sample", "--log-dir", "log", "--commits", "1", "--entries", "1", "--ahead", "1", "--json"}
	if code := Run(args, &stdout, &stderr, "v0.2.0"); code != 0 {
		t.Fatalf("code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), `"schema_version":"achta.slice-check.v1"`) || !strings.Contains(stdout.String(), `"status":"pass"`) {
		t.Fatalf("unexpected slice JSON: %s", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	args = []string{"--workspace", root, "slice", "check", "--repo", "sample", "--log-dir", "missing", "--commits", "1", "--entries", "1", "--ahead", "1", "--json"}
	if code := Run(args, &stdout, &stderr, "v0.2.0"); code != 2 {
		t.Fatalf("missing log directory code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), `"status":"cannot_evaluate"`) {
		t.Fatalf("missing directory did not fail closed: %s", stdout.String())
	}
}

func TestWitnessSummaryDigestWorkflow(t *testing.T) {
	root, repo, head := testWorkspaceRepo(t, "sample")
	recordPath := filepath.Join(repo, "GATE-RUN.txt")
	writeTestFile(t, recordPath, "placeholder\n")
	digest, err := gitstate.TreeDigest(repo, "GATE-RUN.txt")
	if err != nil {
		t.Fatal(err)
	}
	record := "schema: gate-run.v1\nrepo-head: " + head[:7] + "\nstarted: 2026-09-04T10:00:00Z\nplan: build test\nelapsed: build 1s\ntarget: build exit=0\nelapsed: test 2s\ntarget: test exit=0\nfinished: 2026-09-04T10:00:04Z\ntree: sha256=" + digest + "\nresult: green\n"
	writeTestFile(t, recordPath, record)
	var stdout, stderr bytes.Buffer
	args := []string{"--workspace", root, "witness", "summarize", "--repo", "sample", "--record", "sample/GATE-RUN.txt", "--slowest", "2", "--json"}
	if code := Run(args, &stdout, &stderr, "v0.1.0"); code != 0 {
		t.Fatalf("code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	var summary map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &summary); err != nil {
		t.Fatal(err)
	}
	if summary["freshness"] != "current" || summary["status"] != "green" {
		t.Fatalf("summary = %v", summary)
	}

	runGit(t, repo, "commit", "--allow-empty", "-m", "change repository HEAD after witness")
	stdout.Reset()
	stderr.Reset()
	if code := Run(args, &stdout, &stderr, "v0.1.0"); code != 1 {
		t.Fatalf("stale summary code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if err := json.Unmarshal(stdout.Bytes(), &summary); err != nil {
		t.Fatal(err)
	}
	if summary["freshness"] != "stale" {
		t.Fatalf("stale summary = %v", summary)
	}
}

func TestAmendmentDeclareCheckThenWrite(t *testing.T) {
	root, repo, head := testWorkspaceRepo(t, "sample")
	manifestPath := filepath.Join(repo, "ledger-amendments.json")
	data, err := amendments.Encode(amendments.Manifest{SchemaVersion: amendments.Schema, BaseCommit: head, Changes: []amendments.Change{}})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(manifestPath, data, 0o600); err != nil {
		t.Fatal(err)
	}
	reasonPath := filepath.Join(repo, "reason.txt")
	writeTestFile(t, reasonPath, "Adds a rule deliberately.\n")
	var stdout, stderr bytes.Buffer
	baseArgs := []string{"--workspace", root, "amendments", "declare", "--manifest", "sample/ledger-amendments.json", "--rule", "A-1", "--class", "rule_added", "--reason-file", "sample/reason.txt", "--json"}
	invalidArgs := append([]string{}, baseArgs...)
	invalidArgs[9] = "not_a_change_class"
	if code := Run(invalidArgs, &stdout, &stderr, "v0.1.0"); code != 2 {
		t.Fatalf("invalid class code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	checkArgs := append(append([]string{}, baseArgs...), "--check")
	if code := Run(checkArgs, &stdout, &stderr, "v0.1.0"); code != 1 {
		t.Fatalf("check code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	parsed, err := amendments.Parse([]byte(readTestFile(t, manifestPath)))
	if err != nil || len(parsed.Changes) != 0 {
		t.Fatalf("check changed manifest: %+v, %v", parsed, err)
	}
	stdout.Reset()
	stderr.Reset()
	if code := Run(baseArgs, &stdout, &stderr, "v0.1.0"); code != 0 {
		t.Fatalf("write code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	parsed, err = amendments.Parse([]byte(readTestFile(t, manifestPath)))
	if err != nil || len(parsed.Changes) != 1 || parsed.Changes[0].RuleID != "A-1" {
		t.Fatalf("manifest = %+v, %v", parsed, err)
	}
	stdout.Reset()
	stderr.Reset()
	if code := Run(baseArgs, &stdout, &stderr, "v0.1.0"); code != 1 {
		t.Fatalf("duplicate declaration refusal code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
}

func TestAmendmentRebasePreservesChanges(t *testing.T) {
	root, repo, firstHead := testWorkspaceRepo(t, "sample")
	witnessPath := filepath.Join(repo, "GATE-RUN.txt")
	writeTestFile(t, witnessPath, "schema: gate-run.v1\nplan: verify\n")
	runGit(t, repo, "add", "GATE-RUN.txt")
	runGit(t, repo, "commit", "-m", "Witness: make verify green at fixture")
	witnessHead := strings.TrimSpace(runGit(t, repo, "rev-parse", "HEAD"))
	manifest := amendments.Manifest{SchemaVersion: amendments.Schema, BaseCommit: firstHead, Changes: []amendments.Change{{RuleID: "A-1", ChangeClass: "rule_added", Reason: "Keeps this reason."}}}
	data, err := amendments.Encode(manifest)
	if err != nil {
		t.Fatal(err)
	}
	manifestPath := filepath.Join(repo, "ledger-amendments.json")
	if err := os.WriteFile(manifestPath, data, 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	args := []string{"--workspace", root, "amendments", "rebase", "--manifest", "sample/ledger-amendments.json", "--repo", "sample", "--witness-log", "sample/GATE-RUN.txt", "--json"}
	if code := Run(args, &stdout, &stderr, "v0.1.0"); code != 0 {
		t.Fatalf("code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	got, err := amendments.Parse([]byte(readTestFile(t, manifestPath)))
	if err != nil {
		t.Fatal(err)
	}
	if got.BaseCommit != witnessHead || len(got.Changes) != 1 || got.Changes[0].Reason != "Keeps this reason." {
		t.Fatalf("manifest = %+v", got)
	}
}

func TestAmendmentReconcileCompleteCLIWorkflow(t *testing.T) {
	root, repo, head := testWorkspaceRepo(t, "sample")
	manifestPath := filepath.Join(repo, "ledger-amendments.json")
	writeManifest := func(ruleID string) {
		t.Helper()
		data, err := amendments.Encode(amendments.Manifest{
			SchemaVersion: amendments.Schema,
			BaseCommit:    head,
			Changes: []amendments.Change{{
				RuleID:      ruleID,
				ChangeClass: "rule_added",
				Reason:      "Declares the measured rule addition.",
			}},
		})
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(manifestPath, data, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	writeManifest("A-1")

	rulefloorPath := filepath.Join(root, "fake-rulefloor")
	script := fmt.Sprintf(`#!/bin/sh
case "$1" in
  capabilities)
    printf '%%s\n' '{"schema_version":"rulefloor.capabilities.v1","machine_interfaces":["rulefloor.ledger-diff.v1"],"ledger_features":["ledger-diff-sentence-sha256"]}'
    exit 0
    ;;
  ledger-diff)
    printf '%%s\n' '{"schema_version":"rulefloor.ledger-diff.v1","status":"different","base_commit":"%s","headers_changed":false,"header_changes":[],"rules":[{"rule_id":"A-1","changes":["rule_added"]}],"total_rule_changes":1,"truncated":false}'
    exit 1
    ;;
esac
exit 2
`, head)
	if err := os.WriteFile(rulefloorPath, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	args := []string{"--workspace", root, "amendments", "reconcile", "--manifest", "sample/ledger-amendments.json", "--repo", "sample", "--rulefloor", rulefloorPath, "--json"}
	var stdout, stderr bytes.Buffer
	if code := Run(args, &stdout, &stderr, "v0.2.0"); code != 0 {
		t.Fatalf("reconcile code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	var result struct {
		Status              string   `json:"status"`
		RulefloorExecutable string   `json:"rulefloor_executable"`
		Problems            []string `json:"problems"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Status != "pass" || result.RulefloorExecutable != rulefloorPath || len(result.Problems) != 0 {
		t.Fatalf("reconciliation = %+v", result)
	}

	writeManifest("B-1")
	stdout.Reset()
	stderr.Reset()
	if code := Run(args, &stdout, &stderr, "v0.2.0"); code != 1 {
		t.Fatalf("mismatch code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Status != "fail" || len(result.Problems) != 2 {
		t.Fatalf("mismatch reconciliation = %+v", result)
	}
}

func testWorkspaceRepo(t *testing.T, name string) (string, string, string) {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "wiki", "repos"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "wiki", "platform"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, filepath.Join(root, "wiki", "platform", "decisions.md"), "# Decisions\n")
	repo := filepath.Join(root, name)
	if err := os.Mkdir(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	runGit(t, repo, "init", "-b", "main")
	runGit(t, repo, "config", "user.name", "Achta Test")
	runGit(t, repo, "config", "user.email", "achta-test@example.invalid")
	writeTestFile(t, filepath.Join(repo, "payload.txt"), "payload\n")
	runGit(t, repo, "add", "payload.txt")
	runGit(t, repo, "commit", "-m", "fixture")
	head := strings.TrimSpace(runGit(t, repo, "rev-parse", "HEAD"))
	return root, repo, head
}

func runGit(t *testing.T, repo string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v: %s", args, err, out)
	}
	return string(out)
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func readTestFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

const pageFixtureCLI = `---
title: REPOSITORY
category: repo
updated: 2026-09-01
verified: 2026-09-01
verified_against: REPOSITORY @ aaaaaaa (main). Keep this suffix.
---

# REPOSITORY

<!-- BEGIN DERIVED: REPOSITORY -->
| Derived fact | Value |
| --- | --- |
| Repo HEAD | aaaaaaa (main) |
<!-- END DERIVED -->
`

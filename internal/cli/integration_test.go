package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ozgurcd/achta/internal/amendments"
	"github.com/ozgurcd/achta/internal/gitstate"
)

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

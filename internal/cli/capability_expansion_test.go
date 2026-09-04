package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReachabilityClassifyFailsClosedAndRecordsSkipPaths(t *testing.T) {
	root, repo, base := testWorkspaceRepo(t, "sample")
	if err := os.MkdirAll(filepath.Join(repo, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, filepath.Join(repo, "docs", "guide.md"), "guide\n")
	runGit(t, repo, "add", "docs/guide.md")
	runGit(t, repo, "commit", "-m", "docs")
	var stdout, stderr bytes.Buffer
	args := []string{"--workspace", root, "reachability", "classify", "--repo", "sample", "--base", base, "--no-reach", "docs/**=documentation only", "--json"}
	if code := Run(args, &stdout, &stderr, "v0.3.0"); code != 0 {
		t.Fatalf("skippable code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), `"decision":"SKIPPABLE"`) || !strings.Contains(stdout.String(), `"path":"docs/guide.md"`) {
		t.Fatalf("skip evidence missing: %s", stdout.String())
	}
	stdout.Reset()
	stderr.Reset()
	args = []string{"--workspace", root, "reachability", "classify", "--repo", "sample", "--base", base, "--json"}
	if code := Run(args, &stdout, &stderr, "v0.3.0"); code != 1 || !strings.Contains(stdout.String(), `"decision":"REQUIRED"`) {
		t.Fatalf("required code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	args = []string{"--workspace", root, "reachability", "classify", "--repo", "sample", "--base", base, "--no-reach", "**/*=too broad", "--json"}
	if code := Run(args, &stdout, &stderr, "v0.3.0"); code != 2 || !strings.Contains(stdout.String(), "catch-all") {
		t.Fatalf("catch-all code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
}

func TestWitnessEarnedRefusesRecordOnlyCycle(t *testing.T) {
	root, repo, _ := testWorkspaceRepo(t, "sample")
	recordPath := filepath.Join(repo, "GATE-RUN.txt")
	writeTestFile(t, recordPath, "schema: gate-run.v1\nplan: verify\n")
	runGit(t, repo, "add", "GATE-RUN.txt")
	runGit(t, repo, "commit", "-m", "Witness: make verify green at baseline")
	writeTestFile(t, recordPath, "schema: gate-run.v1\nplan: verify\nnote: bookkeeping\n")
	runGit(t, repo, "add", "GATE-RUN.txt")
	runGit(t, repo, "commit", "-m", "bookkeeping")
	var stdout, stderr bytes.Buffer
	args := []string{"--workspace", root, "witness", "earned", "--repo", "sample", "--record", "sample/GATE-RUN.txt", "--json"}
	if code := Run(args, &stdout, &stderr, "v0.3.0"); code != 1 || !strings.Contains(stdout.String(), `"decision":"REFUSE"`) {
		t.Fatalf("record-only code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	writeTestFile(t, filepath.Join(repo, "substantive.txt"), "change\n")
	runGit(t, repo, "add", "substantive.txt")
	runGit(t, repo, "commit", "-m", "substantive")
	stdout.Reset()
	stderr.Reset()
	if code := Run(args, &stdout, &stderr, "v0.3.0"); code != 0 || !strings.Contains(stdout.String(), `"decision":"EARNED"`) {
		t.Fatalf("earned code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
}

// RULE: WITNESS-DEPENDENCY-PINS-1
func TestWitnessSiblingPinAndNoReachAdvance(t *testing.T) {
	root, primary, _ := testWorkspaceRepo(t, "primary")
	sibling := filepath.Join(root, "sibling")
	if err := os.Mkdir(sibling, 0o755); err != nil {
		t.Fatal(err)
	}
	runGit(t, sibling, "init", "-b", "main")
	runGit(t, sibling, "config", "user.name", "Achta Test")
	runGit(t, sibling, "config", "user.email", "achta-test@example.invalid")
	writeTestFile(t, filepath.Join(sibling, "source.txt"), "source\n")
	runGit(t, sibling, "add", "source.txt")
	runGit(t, sibling, "commit", "-m", "fixture")
	record := "primary/GATE-RUN.txt"
	var stdout, stderr bytes.Buffer
	runWitnessCommand(t, root, []string{"witness", "init", "--repo", "primary", "--record", record, "--label", "two repo", "--targets", "verify", "--json"}, &stdout, &stderr)
	runWitnessCommand(t, root, []string{"witness", "step", "--repo", "primary", "--record", record, "--target", "verify", "--exit-code", "0", "--json"}, &stdout, &stderr)
	runWitnessCommand(t, root, []string{"witness", "finalize", "--repo", "primary", "--record", record, "--sibling", "sibling=sibling", "--json"}, &stdout, &stderr)
	data, err := os.ReadFile(filepath.Join(primary, "GATE-RUN.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "xrepo: sibling head=") {
		t.Fatalf("sibling pin missing: %s", data)
	}
	runWitnessCommand(t, root, []string{"witness", "check", "--repo", "primary", "--record", record, "--sibling", "sibling=sibling", "--json"}, &stdout, &stderr)
	if err := os.MkdirAll(filepath.Join(sibling, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, filepath.Join(sibling, "docs", "guide.md"), "guide\n")
	runGit(t, sibling, "add", "docs/guide.md")
	runGit(t, sibling, "commit", "-m", "docs")
	stdout.Reset()
	stderr.Reset()
	args := []string{"--workspace", root, "witness", "check", "--repo", "primary", "--record", record, "--sibling", "sibling=sibling", "--json"}
	if code := Run(args, &stdout, &stderr, "v0.3.0"); code != 1 || !strings.Contains(stdout.String(), `"status":"stale"`) {
		t.Fatalf("stale sibling code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	args = append(args[:len(args)-1], "--sibling-no-reach", "sibling:docs/**=documentation only", "--json")
	if code := Run(args, &stdout, &stderr, "v0.3.0"); code != 0 || !strings.Contains(stdout.String(), `"status":"proven_no_reach"`) || !strings.Contains(stdout.String(), `"path":"docs/guide.md"`) {
		t.Fatalf("no-reach sibling code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
}

func TestWitnessPrimaryNoReachAdvance(t *testing.T) {
	root, repo, _ := testWorkspaceRepo(t, "sample")
	record := "sample/GATE-RUN.txt"
	var stdout, stderr bytes.Buffer
	runWitnessCommand(t, root, []string{"witness", "init", "--repo", "sample", "--record", record, "--label", "primary", "--targets", "verify", "--json"}, &stdout, &stderr)
	runWitnessCommand(t, root, []string{"witness", "step", "--repo", "sample", "--record", record, "--target", "verify", "--exit-code", "0", "--json"}, &stdout, &stderr)
	runWitnessCommand(t, root, []string{"witness", "finalize", "--repo", "sample", "--record", record, "--json"}, &stdout, &stderr)
	if err := os.MkdirAll(filepath.Join(repo, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, filepath.Join(repo, "docs", "guide.md"), "guide\n")
	runGit(t, repo, "add", "docs/guide.md")
	runGit(t, repo, "commit", "-m", "docs")
	args := []string{"--workspace", root, "witness", "check", "--repo", "sample", "--record", record, "--json"}
	stdout.Reset()
	stderr.Reset()
	if code := Run(args, &stdout, &stderr, "v0.3.0"); code != 1 || !strings.Contains(stdout.String(), `"freshness":"stale"`) {
		t.Fatalf("primary stale code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	args = append(args[:len(args)-1], "--no-reach", "docs/**=documentation only", "--json")
	if code := Run(args, &stdout, &stderr, "v0.3.0"); code != 0 || !strings.Contains(stdout.String(), `"freshness":"proven_no_reach"`) || !strings.Contains(stdout.String(), `"path":"docs/guide.md"`) {
		t.Fatalf("primary no-reach code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
}

// RULE: WITNESS-CI-PROVENANCE-1
func TestWitnessCIProvenanceAcceptsCommitOnHeadAncestry(t *testing.T) {
	root, repo, _ := testWorkspaceRepo(t, "sample")
	writeTestFile(t, filepath.Join(repo, ".gitignore"), "GATE-RUN.txt\n")
	runGit(t, repo, "add", ".gitignore")
	runGit(t, repo, "commit", "-m", "ignore witness")
	head := strings.TrimSpace(runGit(t, repo, "rev-parse", "HEAD"))
	record := "sample/GATE-RUN.txt"
	var stdout, stderr bytes.Buffer
	runWitnessCommand(t, root, []string{"witness", "init", "--repo", "sample", "--record", record, "--label", "ci", "--targets", "verify", "--json"}, &stdout, &stderr)
	runWitnessCommand(t, root, []string{"witness", "step", "--repo", "sample", "--record", record, "--target", "verify", "--exit-code", "0", "--json"}, &stdout, &stderr)
	runWitnessCommand(t, root, []string{"witness", "finalize", "--repo", "sample", "--record", record, "--commit-tie", "--ci-run", "https://github.com/example/sample/actions/runs/42", "--ci-attempt", "1", "--ci-sha", head, "--json"}, &stdout, &stderr)
	runGit(t, repo, "commit", "--allow-empty", "-m", "later commit")
	stdout.Reset()
	stderr.Reset()
	args := []string{"--workspace", root, "witness", "check", "--repo", "sample", "--record", record, "--json"}
	if code := Run(args, &stdout, &stderr, "v0.3.0"); code != 0 || !strings.Contains(stdout.String(), `"freshness":"accepted_ci_ancestor"`) || !strings.Contains(stdout.String(), `"status":"accepted"`) {
		t.Fatalf("CI ancestry code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
}

func runWitnessCommand(t *testing.T, root string, args []string, stdout, stderr *bytes.Buffer) {
	t.Helper()
	stdout.Reset()
	stderr.Reset()
	full := append([]string{"--workspace", root}, args...)
	if code := Run(full, stdout, stderr, "v0.3.0"); code != 0 {
		t.Fatalf("command %v code=%d stdout=%q stderr=%q", args, code, stdout.String(), stderr.String())
	}
}

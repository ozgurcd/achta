package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type wikiCheckProbe struct {
	SchemaVersion string `json:"schema_version"`
	Status        string `json:"status"`
	WikiDir       string `json:"wiki_dir"`
	Checks        []struct {
		Name   string `json:"name"`
		Status string `json:"status"`
	} `json:"checks"`
}

func runWikiCheckProbe(t *testing.T, args ...string) (int, wikiCheckProbe) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	code := Run(args, &stdout, &stderr, "v0.1.0")
	var probe wikiCheckProbe
	if err := json.Unmarshal(stdout.Bytes(), &probe); err != nil {
		t.Fatalf("%v: stdout is not one JSON document: %v\nstdout=%s\nstderr=%s", args, err, stdout.String(), stderr.String())
	}
	return code, probe
}

func checkNames(probe wikiCheckProbe) string {
	names := make([]string, 0, len(probe.Checks))
	for _, check := range probe.Checks {
		names = append(names, check.Name)
	}
	return strings.Join(names, ",")
}

// freshPinStaleDerive builds the caller's shape: a page whose pin IS the
// repository HEAD (freshness passes) but whose derived block is stale on a
// DIRTY working tree (derive fails). `make verify` produces exactly this by
// writing its own gate record before the wiki gate runs.
func freshPinStaleDerive(t *testing.T) (string, string) {
	t.Helper()
	root, repo, head := testWorkspaceRepo(t, "sample")
	page := strings.ReplaceAll(pageFixtureCLI, "REPOSITORY", "sample")
	page = strings.Replace(page, "verified_against: sample @ aaaaaaa", "verified_against: sample @ "+head, 1)
	writeTestFile(t, filepath.Join(root, "wiki", "repos", "sample.md"), page)
	writeTestFile(t, filepath.Join(repo, "payload.txt"), "payload edited but not committed\n")
	return root, repo
}

// RULE: WIKI-CHECK-SELECTION-1
func TestWikiCheckOnlySelectsChecksAndFailsClosed(t *testing.T) {
	root, repo := freshPinStaleDerive(t)
	wikiDir := filepath.Join(root, "wiki")
	base := []string{"--json", "--wiki-dir", wikiDir, "wiki", "check"}

	// Without a selection the dirty tree fails the composed check through derive.
	code, all := runWikiCheckProbe(t, base...)
	if code != 1 || all.Status != "fail" || checkNames(all) != "freshness,derive" {
		t.Fatalf("composed check: code=%d status=%q checks=%q", code, all.Status, checkNames(all))
	}

	// --only freshness on the same dirty tree: fresh pins pass, derive is not evaluated.
	code, only := runWikiCheckProbe(t, append(base, "--only", "freshness")...)
	if code != 0 || only.Status != "pass" || checkNames(only) != "freshness" {
		t.Fatalf("--only freshness on a dirty tree: code=%d status=%q checks=%q", code, only.Status, checkNames(only))
	}

	// --only derive evaluates derive alone and keeps its exit.
	code, derive := runWikiCheckProbe(t, append(base, "--only", "derive")...)
	if code != 1 || derive.Status != "fail" || checkNames(derive) != "derive" {
		t.Fatalf("--only derive: code=%d status=%q checks=%q", code, derive.Status, checkNames(derive))
	}

	// Naming every check, in any order, is the composed check.
	for _, selection := range []string{"freshness,derive", "derive,freshness"} {
		code, both := runWikiCheckProbe(t, append(base, "--only", selection)...)
		if code != 1 || both.Status != all.Status || checkNames(both) != checkNames(all) {
			t.Fatalf("--only %s: code=%d status=%q checks=%q", selection, code, both.Status, checkNames(both))
		}
	}

	// An unknown, empty, or repeated name is invalid input: exit 2 and nothing is evaluated.
	for _, selection := range []string{"nosuchcheck", "", "freshness,", "freshness,freshness"} {
		var stdout, stderr bytes.Buffer
		code := Run(append(base, "--only", selection), &stdout, &stderr, "v0.1.0")
		if code != 2 || !strings.Contains(stdout.String(), `"status":"cannot_evaluate"`) || strings.Contains(stdout.String(), `"checks"`) {
			t.Fatalf("--only %q: code=%d stdout=%s stderr=%s", selection, code, stdout.String(), stderr.String())
		}
	}

	// An unreadable page under --only freshness is cannot_evaluate, exit 2, and still freshness only.
	broken := filepath.Join(root, "wiki", "repos", "broken.md")
	writeTestFile(t, broken, "---\ncategory: repo\ntitle: frontmatter never closes\n")
	code, unreadable := runWikiCheckProbe(t, append(base, "--only", "freshness")...)
	if code != 2 || unreadable.Status != "cannot_evaluate" || checkNames(unreadable) != "freshness" {
		t.Fatalf("unreadable page: code=%d status=%q checks=%q", code, unreadable.Status, checkNames(unreadable))
	}
	if err := os.Remove(broken); err != nil {
		t.Fatal(err)
	}

	// A pin behind its repository HEAD under --only freshness is exit 1.
	runGit(t, repo, "commit", "-am", "advance past the pin")
	code, behind := runWikiCheckProbe(t, append(base, "--only", "freshness")...)
	if code != 1 || behind.Status != "fail" || checkNames(behind) != "freshness" {
		t.Fatalf("pin behind HEAD: code=%d status=%q checks=%q", code, behind.Status, checkNames(behind))
	}
}

// RULE: WIKI-CHECK-WIKI-DIR-1
func TestWikiCheckNamesResolvedWikiDir(t *testing.T) {
	root, _ := freshPinStaleDerive(t)
	want, err := filepath.EvalSymlinks(filepath.Join(root, "wiki"))
	if err != nil {
		t.Fatal(err)
	}
	for _, selector := range [][]string{{"--wiki-dir", filepath.Join(root, "wiki")}, {"--workspace", root}} {
		args := append([]string{"--json"}, selector...)
		args = append(args, "wiki", "check", "--only", "freshness")
		code, probe := runWikiCheckProbe(t, args...)
		if code != 0 {
			t.Fatalf("%v: code=%d status=%q", selector, code, probe.Status)
		}
		if probe.WikiDir != want {
			t.Fatalf("%v: wiki_dir=%q, want the resolved wiki %q", selector, probe.WikiDir, want)
		}
	}

	var stdout, stderr bytes.Buffer
	if code := Run([]string{"--wiki-dir", filepath.Join(root, "wiki"), "wiki", "check", "--only", "freshness"}, &stdout, &stderr, "v0.1.0"); code != 0 {
		t.Fatalf("text mode code=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "wiki check wiki_dir: "+want+"\n") {
		t.Fatalf("text mode does not name the resolved wiki:\n%s", stdout.String())
	}
}

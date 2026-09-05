package census

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func fixtureSources(t *testing.T) (string, []Source) {
	t.Helper()
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "tools", "a.sh"), "a\nb\nc\n")
	writeFile(t, filepath.Join(root, "tools", "b.sh"), "1\n2\n3\n4\n5\n")
	writeFile(t, filepath.Join(root, "gotools", "alpha", "main.go"), "package main\nfunc main() {}\n\n")
	writeFile(t, filepath.Join(root, "gotools", "beta", "x.go"), "package beta\n\n")
	writeFile(t, filepath.Join(root, "gotools", "beta", "sub", "y.go"), "1\n2\n3\n4\n")
	writeFile(t, filepath.Join(root, "gotools", "beta", "README.md"), "not go\n")
	return root, []Source{
		{Path: filepath.Join(root, "tools"), Kind: "files"},
		{Label: "GO", Path: filepath.Join(root, "gotools"), Kind: "dirs", Ext: ".go"},
	}
}

const header = "| # | subject | lines | answers | call sites | rule | achta | mark |\n|---|---|---|---|---|---|---|---|\n"

const rowsClean = header +
	"| 1 | a.sh | 3 | x | y | none | NONE | KEEP — parts |\n" +
	"| 2 | b.sh | 5 | x | y | none | witness check | CANDIDATE — order 1 |\n" +
	"| 3 | gone.sh | 7 | x | y | none | slice check | RETIRED 2026-09-05 |\n" +
	"| 4 | gotools/alpha | 3 | x | y | none | NONE | KEEP |\n" +
	"| 5 | gotools/beta | 6 | x | y | RULE-1 | reachability classify | CANDIDATE — rules-bound |\n"

const totalsClean = "\n## Totals\n\n| mark | scripts | lines |\n|---|---|---|\n" +
	"| KEEP | 1 | 3 |\n| CANDIDATE | 1 | 5 |\n| RETIRED (2026-09-05) | 1 | 7 |\n| total | 3 | 15 |\n" +
	"| GO-KEEP | 1 | 3 |\n| GO-CANDIDATE | 1 | 6 |\n| GO-total | 2 | 9 |\n"

func rules(result Result) string {
	var out []string
	for _, v := range result.Violations {
		out = append(out, v.Rule)
	}
	return strings.Join(out, ",")
}

// RULE: LEDGER-CENSUS-1
func TestCheckRecountsRowsAgainstTotalsAndDisk(t *testing.T) {
	_, sources := fixtureSources(t)
	clean, err := Check([]byte(rowsClean+totalsClean), sources)
	if err != nil || clean.Status != "pass" || clean.Rows != 5 || len(clean.Violations) != 0 {
		t.Fatalf("clean ledger: err=%v status=%q rows=%d violations=%v", err, clean.Status, clean.Rows, clean.Violations)
	}

	// THE FOUNDING CASE: a row added, totals untouched.
	root := sources[0].Path
	writeFile(t, filepath.Join(root, "c.sh"), "q\nw\ne\nr\nt\ny\nu\n")
	added := strings.Replace(rowsClean, "| 4 | gotools/alpha", "| 6 | c.sh (added today) | 7 | x | y | none | NONE | KEEP |\n| 4 | gotools/alpha", 1)
	got, err := Check([]byte(added+totalsClean), sources)
	if err != nil || got.Status != "fail" || !strings.Contains(rules(got), "totals") {
		t.Fatalf("row added with totals untouched must fail: err=%v status=%q rules=%q", err, got.Status, rules(got))
	}
	fixed := strings.NewReplacer("| KEEP | 1 | 3 |", "| KEEP | 2 | 10 |", "| total | 3 | 15 |", "| total | 4 | 22 |").Replace(totalsClean)
	if got, err := Check([]byte(added+fixed), sources); err != nil || got.Status != "pass" {
		t.Fatalf("corrected totals must pass: err=%v status=%q violations=%v", err, got.Status, got.Violations)
	}
	if err := os.Remove(filepath.Join(root, "c.sh")); err != nil {
		t.Fatal(err)
	}

	// A present-marked row must exist at EXACTLY the stated size.
	stale := strings.Replace(rowsClean, "| 1 | a.sh | 3 |", "| 1 | a.sh | 4 |", 1)
	staleTotals := strings.NewReplacer("| KEEP | 1 | 3 |", "| KEEP | 1 | 4 |", "| total | 3 | 15 |", "| total | 3 | 16 |").Replace(totalsClean)
	if got, err := Check([]byte(stale+staleTotals), sources); err != nil || got.Status != "fail" || !strings.Contains(rules(got), "size") {
		t.Fatalf("stale size must fail: err=%v status=%q rules=%q", err, got.Status, rules(got))
	}

	// A retired-marked row must NOT exist.
	writeFile(t, filepath.Join(root, "gone.sh"), "g\n")
	if got, err := Check([]byte(rowsClean+totalsClean), sources); err != nil || got.Status != "fail" || !strings.Contains(rules(got), "retired-present") {
		t.Fatalf("retired file present must fail: err=%v status=%q rules=%q", err, got.Status, rules(got))
	}
	if err := os.Remove(filepath.Join(root, "gone.sh")); err != nil {
		t.Fatal(err)
	}

	// Every file and every dir on disk must have a row.
	writeFile(t, filepath.Join(root, "d.sh"), "z\n")
	if got, err := Check([]byte(rowsClean+totalsClean), sources); err != nil || got.Status != "fail" || !strings.Contains(rules(got), "unrowed") {
		t.Fatalf("unrowed file must fail: err=%v status=%q rules=%q", err, got.Status, rules(got))
	}
	if err := os.Remove(filepath.Join(root, "d.sh")); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(sources[1].Path, "gamma", "g.go"), "package gamma\n")
	if got, err := Check([]byte(rowsClean+totalsClean), sources); err != nil || got.Status != "fail" || !strings.Contains(rules(got), "unrowed") {
		t.Fatalf("unrowed dir must fail: err=%v status=%q rules=%q", err, got.Status, rules(got))
	}
	if err := os.RemoveAll(filepath.Join(sources[1].Path, "gamma")); err != nil {
		t.Fatal(err)
	}

	// Names are unique; a row that matches no source is refused.
	dup := rowsClean + "| 9 | a.sh | 3 | x | y | none | NONE | KEEP |\n"
	if got, err := Check([]byte(dup+totalsClean), sources); err != nil || got.Status != "fail" || !strings.Contains(rules(got), "duplicate-subject") {
		t.Fatalf("duplicate subject must fail: err=%v status=%q rules=%q", err, got.Status, rules(got))
	}
	stray := rowsClean + "| 9 | mystery | 3 | x | y | none | NONE | KEEP |\n"
	if got, err := Check([]byte(stray+totalsClean), sources); err != nil || got.Status != "fail" || !strings.Contains(rules(got), "unassigned-row") {
		t.Fatalf("unassigned row must fail: err=%v status=%q rules=%q", err, got.Status, rules(got))
	}
}

func TestCheckCannotEvaluate(t *testing.T) {
	root, sources := fixtureSources(t)
	if _, err := Check([]byte(rowsClean+totalsClean), nil); err == nil {
		t.Fatal("no sources must be an error")
	}
	if _, err := Check([]byte("# nothing here\n"), sources); err == nil {
		t.Fatal("no rows must be an error")
	}
	missing := []Source{{Path: filepath.Join(root, "nowhere"), Kind: "files"}}
	if _, err := Check([]byte(rowsClean+totalsClean), missing); err == nil {
		t.Fatal("an unreadable source dir must be an error")
	}
	if _, err := Check([]byte(rowsClean+totalsClean), []Source{{Path: sources[1].Path, Kind: "dirs"}}); err == nil {
		t.Fatal("a dirs source without an extension must be an error")
	}
}

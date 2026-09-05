package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const recipeFixtureMakefile = "check:\n\t@bash tools/gate.sh run RECORD \"label\" \\\n\t\t'a=bash tools/a.sh' \\\n\t\t'b=achta --wiki-dir . wiki check'\n"

func TestRecipeCheckCommandRefusesNeutralizedRecipe(t *testing.T) {
	root, _, _ := testWorkspaceRepo(t, "sample")
	writeTestFile(t, filepath.Join(root, "Makefile"), recipeFixtureMakefile)
	writeTestFile(t, filepath.Join(root, "Makefile.neutralized"), strings.Replace(recipeFixtureMakefile, "\t@bash tools/gate.sh", "\t-@bash tools/gate.sh", 1))
	writeTestFile(t, filepath.Join(root, "expected.txt"), "@bash tools/gate.sh run RECORD \"label\" \\\n\t'a=bash tools/a.sh' \\\n\t'b=achta --wiki-dir . wiki check'\n")

	var stdout, stderr bytes.Buffer
	args := []string{"--json", "--workspace", root, "recipe", "check", "--makefile", "Makefile", "--target", "check", "--expect-file", "expected.txt", "--forbid-noop"}
	if code := Run(args, &stdout, &stderr, "v0.1.0"); code != 0 {
		t.Fatalf("clean recipe code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), `"schema_version":"achta.recipe-check.v1"`) || !strings.Contains(stdout.String(), `"status":"pass"`) {
		t.Fatalf("unexpected recipe JSON: %s", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	args = []string{"--json", "--workspace", root, "recipe", "check", "--makefile", "Makefile.neutralized", "--target", "check", "--expect-file", "expected.txt"}
	if code := Run(args, &stdout, &stderr, "v0.1.0"); code != 1 {
		t.Fatalf("neutralized recipe code=%d stdout=%q", code, stdout.String())
	}
	if !strings.Contains(stdout.String(), `"rule":"ignored-exit"`) || !strings.Contains(stdout.String(), `"missing_expected":["@bash tools/gate.sh run RECORD \"label\" \\"]`) {
		t.Fatalf("neutralized recipe did not name the dash and the missing expected line: %s", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	args = []string{"--json", "--workspace", root, "recipe", "check", "--makefile", "Makefile", "--target", "nosuch"}
	if code := Run(args, &stdout, &stderr, "v0.1.0"); code != 2 {
		t.Fatalf("missing target must be cannot_evaluate: code=%d stdout=%q", code, stdout.String())
	}
	if code := Run([]string{"--json", "--workspace", root, "recipe", "check", "--makefile", "../outside/Makefile", "--target", "check"}, &stdout, &stderr, "v0.1.0"); code != 2 {
		t.Fatalf("a makefile outside the workspace must be refused: code=%d", code)
	}
}

func TestLedgerCensusCommandRecountsAgainstDisk(t *testing.T) {
	root, _, _ := testWorkspaceRepo(t, "sample")
	if err := os.MkdirAll(filepath.Join(root, "wiki", "tools"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, filepath.Join(root, "wiki", "tools", "a.sh"), "a\nb\nc\n")
	writeTestFile(t, filepath.Join(root, "wiki", "tools", "b.sh"), "1\n2\n")
	ledger := "| # | script | lines | answers | call sites | rule | achta | mark |\n|---|---|---|---|---|---|---|---|\n" +
		"| 1 | a.sh | 3 | x | y | none | NONE | KEEP |\n| 2 | b.sh | 2 | x | y | none | NONE | KEEP |\n\n" +
		"| mark | scripts | lines |\n|---|---|---|\n| KEEP | 2 | 5 |\n| total | 2 | 5 |\n"
	writeTestFile(t, filepath.Join(root, "wiki", "ledger.md"), ledger)

	var stdout, stderr bytes.Buffer
	args := []string{"--json", "--workspace", root, "ledger", "census", "--file", "wiki/ledger.md", "--dir", "wiki/tools"}
	if code := Run(args, &stdout, &stderr, "v0.1.0"); code != 0 {
		t.Fatalf("clean census code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), `"schema_version":"achta.ledger-census.v1"`) || !strings.Contains(stdout.String(), `"status":"pass"`) {
		t.Fatalf("unexpected census JSON: %s", stdout.String())
	}

	// THE FOUNDING CASE at the command level: a file appears, the ledger does not move.
	writeTestFile(t, filepath.Join(root, "wiki", "tools", "c.sh"), "z\n")
	stdout.Reset()
	if code := Run(args, &stdout, &stderr, "v0.1.0"); code != 1 || !strings.Contains(stdout.String(), `"rule":"unrowed"`) {
		t.Fatalf("unrowed file must fail: code=%d stdout=%q", code, stdout.String())
	}
	if err := os.Remove(filepath.Join(root, "wiki", "tools", "c.sh")); err != nil {
		t.Fatal(err)
	}

	stdout.Reset()
	if code := Run([]string{"--json", "--workspace", root, "ledger", "census", "--file", "wiki/ledger.md", "--tree", "wiki/tools"}, &stdout, &stderr, "v0.1.0"); code != 2 {
		t.Fatalf("--tree without --ext must be invalid input: code=%d stdout=%q", code, stdout.String())
	}
	stdout.Reset()
	if code := Run([]string{"--json", "--workspace", root, "ledger", "census", "--file", "wiki/ledger.md", "--dir", "wiki/nowhere"}, &stdout, &stderr, "v0.1.0"); code != 2 {
		t.Fatalf("an unreadable --dir must be cannot_evaluate: code=%d stdout=%q", code, stdout.String())
	}
}

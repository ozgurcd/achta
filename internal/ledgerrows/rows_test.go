package ledgerrows

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCheckFiresEachStructuralRuleAndPassesCleanRows(t *testing.T) {
	opts := testOptions()
	for _, tc := range []struct {
		name     string
		fixture  string
		status   string
		rule     string
		identity string
	}{
		{name: "clean", fixture: "clean.md", status: "pass"},
		{name: "open completion", fixture: "completion-open.md", status: "fail", rule: RuleOpenCompletion, identity: "CE-SEC-5b"},
		{name: "missing condition quote", fixture: "condition-unquoted.md", status: "fail", rule: RuleConditionQuote, identity: "DOC-2"},
		{name: "paraphrased condition quote", fixture: "condition-paraphrase.md", status: "fail", rule: RuleConditionQuote, identity: "DOC-3"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join("..", "..", "testdata", "ledgerrows", tc.fixture))
			if err != nil {
				t.Fatal(err)
			}
			result, err := Check(data, opts)
			if err != nil {
				t.Fatal(err)
			}
			if result.Status != tc.status {
				t.Fatalf("status=%q want %q; violations=%+v", result.Status, tc.status, result.Violations)
			}
			if tc.rule == "" {
				if len(result.Violations) != 0 {
					t.Fatalf("clean fixture violations=%+v", result.Violations)
				}
				return
			}
			if len(result.Violations) != 1 || result.Violations[0].Rule != tc.rule || result.Violations[0].Identity != tc.identity {
				t.Fatalf("violations=%+v", result.Violations)
			}
		})
	}
}

func TestCheckRefusesMeaningInferenceAndRequiresExplicitVocabulary(t *testing.T) {
	opts := testOptions()
	data := []byte("| ID | Item |\n|---|---|\n| **SEM-1** | IMPLEMENTED at abc1234, but IMPLEMENTED is not a configured completion marker. |\n")
	result, err := Check(data, opts)
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "pass" || len(result.Violations) != 0 {
		t.Fatalf("unconfigured semantic synonym must stay outside the evaluator: %+v", result)
	}
	if len(result.Refused) != 2 {
		t.Fatalf("refused=%v", result.Refused)
	}

	opts.CompletionMarkers = nil
	if _, err := Check(data, opts); err == nil {
		t.Fatal("missing caller vocabulary must be cannot-evaluate")
	}
}

func TestCheckRejectsAmbiguousOrMalformedRows(t *testing.T) {
	opts := testOptions()
	if _, err := Check([]byte("| ID | Item |\n"), opts); err == nil {
		t.Fatal("zero matched rows must be cannot-evaluate")
	}
	if _, err := Check([]byte("| DONE **BROKEN | prose |\n"), opts); err == nil {
		t.Fatal("a state-marked row without the identity end marker must be cannot-evaluate")
	}

	opts.ClosedMarker = "**DONE"
	opts.OpenMarker = "**"
	if _, err := Check([]byte("| **R1** | prose |\n"), opts); err == nil {
		t.Fatal("overlapping state markers must be cannot-evaluate")
	}
}

func testOptions() Options {
	return Options{
		File:              "wiki/contracts/to-do-queue.MD",
		IDCell:            1,
		ProseCell:         2,
		OpenMarker:        "**",
		ClosedMarker:      "DONE **",
		IdentityEndMarker: "**",
		CompletionMarkers: []string{"DONE ", "**SATISFIED"},
		ExemptMarkers:     []string{"RULED-OPEN", "STAYS OPEN"},
		ConditionMarkers:  []string{"Close condition.", "Close condition:"},
		QuoteMarker:       "CLOSE-CONDITION-MET:",
	}
}

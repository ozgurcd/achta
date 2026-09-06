// SPDX-License-Identifier: MIT
package countcheck

import "testing"

var testOptions = Options{
	TotalPattern:    `TOTAL[[:space:]]+(?P<count>[0-9]+)[[:space:]]*[|]`,
	PartPattern:     `(VACUOUS|HAS-CONTROL|SOUND)[[:space:]]+(?P<count>[0-9]+)`,
	ClaimPattern:    `claims[[:space:]]+(?P<count>[0-9]+)[[:space:]]+fixed`,
	CitationPattern: `[[:alnum:]_./-]+[.](go|md):[0-9]+`,
}

func TestCheckReconcilesBreakdownsAndDistinctCitations(t *testing.T) {
	result, err := Check([]Document{{Path: "log/entry.md", Data: []byte("TOTAL 12 | VACUOUS 4 HAS-CONTROL 6 SOUND 4\n\nclaims 2 fixed: internal/a.go:12, internal/a.go:12\n")}}, testOptions)
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "fail" || result.Files != 1 || result.Breakdowns != 1 || result.Claims != 1 || len(result.Violations) != 2 {
		t.Fatalf("result = %+v", result)
	}
	if result.Violations[0].Rule != RuleBreakdownSum || result.Violations[0].Claimed != 12 || result.Violations[0].Observed != 14 {
		t.Fatalf("breakdown violation = %+v", result.Violations[0])
	}
	if result.Violations[1].Rule != RuleClaimCitation || result.Violations[1].Claimed != 2 || result.Violations[1].Observed != 1 {
		t.Fatalf("citation violation = %+v", result.Violations[1])
	}
}

func TestCheckPassesCorrectRelationshipsAndUsesExplicitAliases(t *testing.T) {
	opts := testOptions
	opts.ClaimPattern = `claims[[:space:]]+(?P<count>[[:alpha:]]+)[[:space:]]+fixed`
	opts.CountAliases = []string{"Two=2"}
	result, err := Check([]Document{{Path: "log/entry.md", Data: []byte("TOTAL 12 | VACUOUS 4 HAS-CONTROL 6 SOUND 2\n\nclaims Two fixed: internal/a.go:12, internal/b.go:34\n")}}, opts)
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "pass" || result.Breakdowns != 1 || result.Claims != 1 || len(result.Violations) != 0 {
		t.Fatalf("result = %+v", result)
	}
}

func TestCheckTreatsAStatedTotalWithoutPartsAsZero(t *testing.T) {
	opts := Options{TotalPattern: testOptions.TotalPattern, PartPattern: testOptions.PartPattern}
	result, err := Check([]Document{{Path: "log.md", Data: []byte("TOTAL 3 |\n")}}, opts)
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "fail" || result.Breakdowns != 1 || len(result.Violations) != 1 || result.Violations[0].Observed != 0 {
		t.Fatalf("result = %+v", result)
	}
}

func TestCheckCannotEvaluateIncompleteOrAmbiguousVocabulary(t *testing.T) {
	for name, opts := range map[string]Options{
		"no patterns":       {},
		"half breakdown":    {TotalPattern: `(?P<count>[0-9]+)`},
		"half claim":        {ClaimPattern: `(?P<count>[0-9]+)`},
		"unnamed count":     {TotalPattern: `([0-9]+)`, PartPattern: `(?P<count>[0-9]+)`},
		"empty citation":    {ClaimPattern: `(?P<count>[0-9]+)`, CitationPattern: `.*`},
		"duplicate aliases": {ClaimPattern: `(?P<count>[[:alpha:]]+)`, CitationPattern: `x`, CountAliases: []string{"Two=2", "Two=2"}},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := Check([]Document{{Path: "log.md", Data: []byte("plain\n")}}, opts); err == nil {
				t.Fatal("cannot-evaluate input was accepted")
			}
		})
	}
}

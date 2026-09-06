package countcheck

import "testing"

// RULE: COUNT-RETIREMENT-PARITY-1
func TestCheckCoversCountClaimRetirementMechanics(t *testing.T) {
	proofOptions := Options{
		ProofClaimPatterns: []string{
			`fixed[[:space:]]+(?P<count>[[:alpha:]]+)[[:space:]]+fences`,
			`(?P<count>[[:alpha:]]+)[[:space:]]+fences[[:space:]]+fixed`,
		},
		ProofPatterns:    []string{`controls[[:space:]]+that[[:space:]]+fired:[[:space:]]*(?P<count>[0-9]+)`},
		ProofWithinLines: 12,
		CountAliases:     []string{"Five=5", "Four=4"},
	}

	result, err := Check([]Document{{Path: "log/entry.md", Data: []byte("we fixed Five fences\ncontrols that fired: 4\n")}}, proofOptions)
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "fail" || result.ProofClaims != 1 || len(result.Violations) != 1 || result.Violations[0].Rule != RuleClaimProof || result.Violations[0].Claimed != 5 || result.Violations[0].Observed != 4 {
		t.Fatalf("claim versus proof result = %+v", result)
	}

	proofOptions.ExemptPatterns = []string{`<!--[[:space:]]*count-claim-exempt:[[:space:]]*[^[:space:]>][^>]*-->`}
	exempt, err := Check([]Document{{Path: "log/entry.md", Data: []byte("<!-- count-claim-exempt: quotes a past mismatch -->\nwe fixed Five fences\ncontrols that fired: 4\n")}}, proofOptions)
	if err != nil {
		t.Fatal(err)
	}
	if exempt.Status != "pass" || exempt.ProofClaims != 0 || len(exempt.Violations) != 0 {
		t.Fatalf("exempt result = %+v", exempt)
	}

	citationOptions := Options{
		ClaimPatterns: []string{
			`(?P<count>Five)[[:space:]]+fences[[:space:]]+fixed`,
			`(?P<count>Four)[[:space:]]+sites[[:space:]]+fixed`,
		},
		CitationPattern:     `[[:alnum:]_./-]+[.]md:[0-9]+`,
		ClaimSectionPattern: `^##[[:space:]]+\[[0-9]{4}-[0-9]{2}-[0-9]{2}\]`,
		CountAliases:        []string{"Five=5", "Four=4"},
	}
	scoped, err := Check([]Document{{Path: "log/entry.md", Data: []byte("## [2026-08-01] old\n\nFive fences fixed.\n\n## [2026-09-06] current\n\nnothing claimed here.\n")}}, citationOptions)
	if err != nil {
		t.Fatal(err)
	}
	if scoped.Status != "pass" || scoped.Claims != 0 || len(scoped.Violations) != 0 {
		t.Fatalf("last-section result = %+v", scoped)
	}

	bodyMatchOptions := citationOptions
	bodyMatchOptions.ClaimSectionPattern = `current`
	bodyMatch, err := Check([]Document{{Path: "log/entry.md", Data: []byte("## current\n\nFour sites fixed: a/b.md:12, a/c.md:34, a/d.md:56.\n\ncurrent body text\n")}}, bodyMatchOptions)
	if err != nil {
		t.Fatal(err)
	}
	if bodyMatch.Status != "fail" || bodyMatch.Claims != 1 || len(bodyMatch.Violations) != 1 {
		t.Fatalf("body line changed section scope = %+v", bodyMatch)
	}

	repeated, err := Check([]Document{{Path: "log/entry.md", Data: []byte("## [2026-09-06] current\n\nFour sites fixed: a/b.md:12, a/c.md:34, a/d.md:56.\n")}}, citationOptions)
	if err != nil {
		t.Fatal(err)
	}
	if repeated.Status != "fail" || repeated.Claims != 1 || len(repeated.Violations) != 1 || repeated.Violations[0].Rule != RuleClaimCitation || repeated.Violations[0].Claimed != 4 || repeated.Violations[0].Observed != 3 {
		t.Fatalf("repeatable claim result = %+v", repeated)
	}
}

func TestCheckRefusesIncompleteRetirementVocabulary(t *testing.T) {
	for name, opts := range map[string]Options{
		"proof without window": {
			ProofClaimPatterns: []string{`(?P<count>[0-9]+) fixed`},
			ProofPatterns:      []string{`proved (?P<count>[0-9]+)`},
		},
		"section without citation pair":  {ClaimSectionPattern: `^## `},
		"exemption without relationship": {ExemptPatterns: []string{`exempt`}},
		"duplicate claim pattern": {
			ClaimPatterns:   []string{`(?P<count>[0-9]+) fixed`, `(?P<count>[0-9]+) fixed`},
			CitationPattern: `cite`,
		},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := Check([]Document{{Path: "entry.md", Data: []byte("plain\n")}}, opts); err == nil {
				t.Fatal("incomplete or ambiguous vocabulary was accepted")
			}
		})
	}
}

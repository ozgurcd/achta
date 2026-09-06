package replacement

import (
	"testing"
)

func exit(value int) *int { return &value }

// RULE: REPLACEMENT-PARITY-CITATION-1
func TestCheckRequiresCitedParityForClaimedReplacement(t *testing.T) {
	manifest := Manifest{
		SchemaVersion: ManifestSchema,
		Claims: []Claim{{
			Verb:   "count check",
			Script: "wiki/tools/count-claim-check.sh",
			Status: "replaces",
		}},
	}

	result, err := Check(manifest, []string{"replaces", "retired"})
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if result.Status != "fail" {
		t.Fatalf("status = %q, want fail", result.Status)
	}
	if got := len(result.Violations); got != 3 {
		t.Fatalf("violations = %d, want 3", got)
	}
}

func TestCheckAcceptsCitedReplayWithAgreeingExitCodes(t *testing.T) {
	manifest := Manifest{
		SchemaVersion: ManifestSchema,
		Claims: []Claim{{
			Verb:             "count check",
			Script:           "wiki/tools/count-claim-check.sh",
			Status:           "replaces",
			SelftestCitation: "wiki/tools/count-claim-check.sh --selftest",
			ReplayCitation:   "testdata/replacement/count-check-replay.txt",
			Fixtures: []Fixture{
				{Name: "clean", ScriptExit: exit(0), VerbExit: exit(0)},
				{Name: "mismatch", ScriptExit: exit(1), VerbExit: exit(1)},
			},
		}},
	}

	result, err := Check(manifest, []string{"replaces", "retired"})
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if result.Status != "pass" {
		t.Fatalf("status = %q, want pass; violations = %#v", result.Status, result.Violations)
	}
}

func TestCheckRejectsExitCodeDisagreement(t *testing.T) {
	manifest := Manifest{
		SchemaVersion: ManifestSchema,
		Claims: []Claim{{
			Verb:             "mirror check --digest",
			Script:           "wiki/tools/gate-witness.sh",
			Status:           "retired",
			SelftestCitation: "wiki/tools/gate-witness.sh --selftest",
			ReplayCitation:   "testdata/replacement/mirror-replay.txt",
			Fixtures:         []Fixture{{Name: "tampered", ScriptExit: exit(1), VerbExit: exit(0)}},
		}},
	}

	result, err := Check(manifest, []string{"replaces", "retired"})
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if result.Status != "fail" || len(result.Violations) != 1 || result.Violations[0].Rule != "exit-code-agreement" {
		t.Fatalf("result = %#v, want one exit-code-agreement violation", result)
	}
}

func TestCheckDoesNotInferCandidateAsReplacement(t *testing.T) {
	manifest := Manifest{
		SchemaVersion: ManifestSchema,
		Claims:        []Claim{{Verb: "count check", Script: "wiki/tools/count-claim-check.sh", Status: "candidate"}},
	}
	result, err := Check(manifest, []string{"replaces", "retired"})
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if result.Status != "pass" || result.Claims != 0 {
		t.Fatalf("result = %#v, want unclaimed candidate ignored", result)
	}
}

func TestParseRejectsUnknownFields(t *testing.T) {
	_, err := Parse([]byte(`{"schema_version":"achta.replacement-claims.v1","claims":[],"surprise":true}`))
	if err == nil {
		t.Fatal("Parse() error = nil, want unknown-field error")
	}
}

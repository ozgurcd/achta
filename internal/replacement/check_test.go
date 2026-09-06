package replacement

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
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

	result, err := Check(manifest, []string{"replaces", "retired"}, nil)
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if result.Status != "fail" {
		t.Fatalf("status = %q, want fail", result.Status)
	}
	if got := len(result.Violations); got != 5 {
		t.Fatalf("violations = %d, want 5", got)
	}
	if result.Violations[1].Rule != "vocabulary-citation-required" {
		t.Fatalf("second violation rule = %q, want vocabulary-citation-required", result.Violations[1].Rule)
	}
}

func TestCheckAcceptsCitedReplayWithAgreeingExitCodes(t *testing.T) {
	fixtures := []Fixture{
		{Name: "clean", ScriptExit: exit(0), VerbExit: exit(0)},
		{Name: "mismatch", ScriptExit: exit(1), VerbExit: exit(1)},
	}
	replay := replayBytes(t, ReplayRecord{
		Verb:               "count check",
		Script:             "wiki/tools/count-claim-check.sh",
		SelftestCitation:   "wiki/tools/count-claim-check.sh --selftest",
		VocabularyCitation: "../wiki/contracts/replacement-replay-2026-09-06-v0.5.6.md:57-107",
		Fixtures:           fixtures,
	})
	manifest := Manifest{
		SchemaVersion: ManifestSchema,
		Claims: []Claim{{
			Verb:               "count check",
			Script:             "wiki/tools/count-claim-check.sh",
			Status:             "replaces",
			SelftestCitation:   "wiki/tools/count-claim-check.sh --selftest",
			VocabularyCitation: "../wiki/contracts/replacement-replay-2026-09-06-v0.5.6.md:57-107",
			ReplayCitation:     "evidence.json",
			ReplaySHA256:       digest(replay),
			Fixtures:           fixtures,
		}},
	}

	result, err := Check(manifest, []string{"replaces", "retired"}, evidenceReader("evidence.json", replay))
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if result.Status != "pass" {
		t.Fatalf("status = %q, want pass; violations = %#v", result.Status, result.Violations)
	}
}

// RULE: REPLACEMENT-VOCAB-CITATION-1
func TestCheckRequiresAndReconcilesVocabularyCitation(t *testing.T) {
	fixtures := []Fixture{{Name: "clean", ScriptExit: exit(0), VerbExit: exit(0)}}
	replay := replayBytes(t, ReplayRecord{
		Verb:               "declared-route check",
		Script:             "wiki/tools/rulefloor-install-gate.sh",
		SelftestCitation:   "wiki/tools/rulefloor-install-gate.sh --selftest",
		VocabularyCitation: "../wiki/contracts/replacement-replay-2026-09-06-v0.5.6.md:124-156",
		Fixtures:           fixtures,
	})
	baseClaim := Claim{
		Verb:             "declared-route check",
		Script:           "wiki/tools/rulefloor-install-gate.sh",
		Status:           "replaces",
		SelftestCitation: "wiki/tools/rulefloor-install-gate.sh --selftest",
		ReplayCitation:   "evidence.json",
		ReplaySHA256:     digest(replay),
		Fixtures:         fixtures,
	}

	t.Run("required", func(t *testing.T) {
		result, err := Check(Manifest{SchemaVersion: ManifestSchema, Claims: []Claim{baseClaim}}, []string{"replaces"}, evidenceReader("evidence.json", replay))
		if err != nil {
			t.Fatalf("Check() error = %v", err)
		}
		if result.Status != "fail" || len(result.Violations) != 1 || result.Violations[0].Rule != "vocabulary-citation-required" {
			t.Fatalf("result = %#v, want vocabulary-citation-required", result)
		}
	})

	t.Run("reconciled", func(t *testing.T) {
		claim := baseClaim
		claim.VocabularyCitation = "README.md:wrong-vocabulary"
		result, err := Check(Manifest{SchemaVersion: ManifestSchema, Claims: []Claim{claim}}, []string{"replaces"}, evidenceReader("evidence.json", replay))
		if err != nil {
			t.Fatalf("Check() error = %v", err)
		}
		if result.Status != "fail" || len(result.Violations) != 1 || result.Violations[0].Rule != "vocabulary-citation-mismatch" {
			t.Fatalf("result = %#v, want vocabulary-citation-mismatch", result)
		}
	})
}

func TestCheckRejectsExitCodeDisagreement(t *testing.T) {
	manifest := Manifest{
		SchemaVersion: ManifestSchema,
		Claims: []Claim{{
			Verb:               "mirror check --digest",
			Script:             "wiki/tools/gate-witness.sh",
			Status:             "retired",
			SelftestCitation:   "wiki/tools/gate-witness.sh --selftest",
			VocabularyCitation: "../wiki/contracts/replacement-replay-2026-09-06-v0.5.6.md:158-196",
			ReplayCitation:     "testdata/replacement/mirror-replay.txt",
			ReplaySHA256:       "0000000000000000000000000000000000000000000000000000000000000000",
			Fixtures:           []Fixture{{Name: "tampered", ScriptExit: exit(1), VerbExit: exit(0)}},
		}},
	}

	result, err := Check(manifest, []string{"replaces", "retired"}, nil)
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
	result, err := Check(manifest, []string{"replaces", "retired"}, nil)
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

func TestCheckRequiresAClosedReplayFixtureSet(t *testing.T) {
	replay := replayBytes(t, ReplayRecord{
		Verb:               "count check",
		Script:             "wiki/tools/count-claim-check.sh",
		SelftestCitation:   "wiki/tools/count-claim-check.sh --selftest",
		VocabularyCitation: "../wiki/contracts/replacement-replay-2026-09-06-v0.5.6.md:57-107",
		Fixtures: []Fixture{
			{Name: "same-name", ScriptExit: exit(1), VerbExit: exit(0)},
			{Name: "evidence-only", ScriptExit: exit(0), VerbExit: exit(0)},
		},
	})
	manifest := Manifest{
		SchemaVersion: ManifestSchema,
		Claims: []Claim{{
			Verb:               "count check",
			Script:             "wiki/tools/count-claim-check.sh",
			Status:             "replaces",
			SelftestCitation:   "wiki/tools/count-claim-check.sh --selftest",
			VocabularyCitation: "../wiki/contracts/replacement-replay-2026-09-06-v0.5.6.md:57-107",
			ReplayCitation:     "evidence.json",
			ReplaySHA256:       digest(replay),
			Fixtures: []Fixture{
				{Name: "same-name", ScriptExit: exit(1), VerbExit: exit(1)},
				{Name: "manifest-only", ScriptExit: exit(0), VerbExit: exit(0)},
			},
		}},
	}

	result, err := Check(manifest, []string{"replaces"}, evidenceReader("evidence.json", replay))
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	want := []string{"replay-row-mismatch", "replay-fixture-absent", "replay-fixture-unrecorded"}
	if result.Status != "fail" || len(result.Violations) != len(want) {
		t.Fatalf("result = %#v, want three closed-set violations", result)
	}
	for index, rule := range want {
		if result.Violations[index].Rule != rule {
			t.Fatalf("violation %d rule = %q, want %q", index, result.Violations[index].Rule, rule)
		}
	}
}

func replayBytes(t *testing.T, record ReplayRecord) []byte {
	t.Helper()
	data, err := json.Marshal(ReplayEvidence{
		SchemaVersion: ReplaySchema,
		Source: ReplaySource{
			Citation: "../wiki/contracts/replacement-replay-2026-09-06.md",
			Commit:   "0000000000000000000000000000000000000000",
			SHA256:   "0000000000000000000000000000000000000000000000000000000000000000",
		},
		Records: []ReplayRecord{record},
	})
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func digest(data []byte) string {
	return fmt.Sprintf("%x", sha256.Sum256(data))
}

func evidenceReader(path string, data []byte) EvidenceReader {
	return func(requested string) ([]byte, error) {
		if requested != path {
			return nil, fmt.Errorf("unexpected replay path %q", requested)
		}
		return data, nil
	}
}

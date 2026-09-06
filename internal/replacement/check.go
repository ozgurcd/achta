package replacement

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

const (
	Schema               = "achta.replacement-check.v1"
	ManifestSchema       = "achta.replacement-claims.v2"
	LegacyManifestSchema = "achta.replacement-claims.v1"
	ReplaySchema         = "achta.replacement-replay.v1"
	MaxManifest          = 1 << 20
	MaxReplay            = 4 << 20
)

type Manifest struct {
	SchemaVersion string  `json:"schema_version"`
	Claims        []Claim `json:"claims"`
}

type Claim struct {
	Verb               string    `json:"verb"`
	Script             string    `json:"script"`
	Status             string    `json:"status"`
	SelftestCitation   string    `json:"selftest_citation,omitempty"`
	VocabularyCitation string    `json:"vocabulary_citation,omitempty"`
	ReplayCitation     string    `json:"replay_citation,omitempty"`
	ReplaySHA256       string    `json:"replay_sha256,omitempty"`
	Fixtures           []Fixture `json:"fixtures,omitempty"`
}

type Fixture struct {
	Name       string `json:"name"`
	ScriptExit *int   `json:"script_exit"`
	VerbExit   *int   `json:"verb_exit"`
}

type ReplayEvidence struct {
	SchemaVersion string         `json:"schema_version"`
	Source        ReplaySource   `json:"source"`
	Records       []ReplayRecord `json:"records"`
}

type ReplaySource struct {
	Citation string `json:"citation"`
	Commit   string `json:"commit"`
	SHA256   string `json:"sha256"`
}

type ReplayRecord struct {
	Verb               string    `json:"verb"`
	Script             string    `json:"script"`
	SelftestCitation   string    `json:"selftest_citation"`
	VocabularyCitation string    `json:"vocabulary_citation,omitempty"`
	Fixtures           []Fixture `json:"fixtures"`
}

type EvidenceReader func(path string) ([]byte, error)

type Violation struct {
	Entry   int    `json:"entry"`
	Verb    string `json:"verb"`
	Script  string `json:"script"`
	Status  string `json:"status"`
	Fixture string `json:"fixture,omitempty"`
	Rule    string `json:"rule"`
	Text    string `json:"text"`
}

type Result struct {
	SchemaVersion string      `json:"schema_version"`
	Status        string      `json:"status"`
	Claims        int         `json:"claims"`
	Violations    []Violation `json:"violations"`
}

func Parse(data []byte) (Manifest, error) {
	var manifest Manifest
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&manifest); err != nil {
		return Manifest{}, fmt.Errorf("decode manifest: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return Manifest{}, fmt.Errorf("decode manifest: multiple JSON values")
		}
		return Manifest{}, fmt.Errorf("decode manifest: %w", err)
	}
	if manifest.SchemaVersion != ManifestSchema && manifest.SchemaVersion != LegacyManifestSchema {
		return Manifest{}, fmt.Errorf("schema_version must be %q or %q", ManifestSchema, LegacyManifestSchema)
	}
	for index, claim := range manifest.Claims {
		if strings.TrimSpace(claim.Verb) == "" || strings.TrimSpace(claim.Script) == "" || strings.TrimSpace(claim.Status) == "" {
			return Manifest{}, fmt.Errorf("claim %d requires non-empty verb, script, and status", index+1)
		}
	}
	return manifest, nil
}

func ParseReplay(data []byte) (ReplayEvidence, error) {
	var replay ReplayEvidence
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&replay); err != nil {
		return ReplayEvidence{}, fmt.Errorf("decode replay evidence: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return ReplayEvidence{}, fmt.Errorf("decode replay evidence: multiple JSON values")
		}
		return ReplayEvidence{}, fmt.Errorf("decode replay evidence: %w", err)
	}
	if replay.SchemaVersion != ReplaySchema {
		return ReplayEvidence{}, fmt.Errorf("replay schema_version must be %q", ReplaySchema)
	}
	if strings.TrimSpace(replay.Source.Citation) == "" || !validHex(replay.Source.Commit, 40) || !validHex(replay.Source.SHA256, 64) {
		return ReplayEvidence{}, fmt.Errorf("replay source requires a citation, lowercase full commit, and lowercase sha256")
	}
	if len(replay.Records) == 0 {
		return ReplayEvidence{}, fmt.Errorf("replay evidence contains no records")
	}
	seenRecords := map[string]bool{}
	for recordIndex, record := range replay.Records {
		if strings.TrimSpace(record.Verb) == "" || strings.TrimSpace(record.Script) == "" || strings.TrimSpace(record.SelftestCitation) == "" {
			return ReplayEvidence{}, fmt.Errorf("replay record %d requires verb, script, and selftest citation", recordIndex+1)
		}
		key := record.Verb + "\x00" + record.Script
		if seenRecords[key] {
			return ReplayEvidence{}, fmt.Errorf("replay repeats verb %q and script %q", record.Verb, record.Script)
		}
		seenRecords[key] = true
		if len(record.Fixtures) == 0 {
			return ReplayEvidence{}, fmt.Errorf("replay record %d contains no fixtures", recordIndex+1)
		}
		seenFixtures := map[string]bool{}
		for _, fixture := range record.Fixtures {
			name := strings.TrimSpace(fixture.Name)
			if name == "" || fixture.Name != name {
				return ReplayEvidence{}, fmt.Errorf("replay record %d has an invalid fixture name", recordIndex+1)
			}
			if seenFixtures[name] {
				return ReplayEvidence{}, fmt.Errorf("replay record %d repeats fixture %q", recordIndex+1, name)
			}
			seenFixtures[name] = true
			if err := validateExitPair(fixture); err != nil {
				return ReplayEvidence{}, fmt.Errorf("replay record %d fixture %q: %w", recordIndex+1, name, err)
			}
		}
	}
	return replay, nil
}

func Check(manifest Manifest, claimStatuses []string, readEvidence EvidenceReader) (Result, error) {
	statusSet := make(map[string]bool, len(claimStatuses))
	for _, status := range claimStatuses {
		if strings.TrimSpace(status) == "" {
			return Result{}, fmt.Errorf("claim status must not be empty")
		}
		if statusSet[status] {
			return Result{}, fmt.Errorf("claim status %q is repeated", status)
		}
		statusSet[status] = true
	}
	if len(statusSet) == 0 {
		return Result{}, fmt.Errorf("at least one claim status is required")
	}

	result := Result{SchemaVersion: Schema, Status: "pass", Violations: []Violation{}}
	for index, claim := range manifest.Claims {
		if !statusSet[claim.Status] {
			continue
		}
		result.Claims++
		if manifest.SchemaVersion == LegacyManifestSchema {
			result.Violations = append(result.Violations, violation(index, claim, "", "legacy-replay-attestation", "v1 replay citations are attestations, not checkable evidence; migrate the claim to v2"))
			continue
		}
		claimInvalid := false
		if strings.TrimSpace(claim.SelftestCitation) == "" {
			result.Violations = append(result.Violations, violation(index, claim, "", "selftest-citation-required", "named-script replacement claim does not cite the script's own selftest fixtures"))
			claimInvalid = true
		}
		if strings.TrimSpace(claim.VocabularyCitation) == "" {
			result.Violations = append(result.Violations, violation(index, claim, "", "vocabulary-citation-required", "named-script replacement claim does not cite the exact caller vocabulary used by the replay"))
			claimInvalid = true
		}
		if strings.TrimSpace(claim.ReplayCitation) == "" {
			result.Violations = append(result.Violations, violation(index, claim, "", "dual-replay-citation-required", "named-script replacement claim does not cite a workspace-confined replay artifact"))
			claimInvalid = true
		}
		if strings.TrimSpace(claim.ReplaySHA256) == "" {
			result.Violations = append(result.Violations, violation(index, claim, "", "dual-replay-digest-required", "named-script replacement claim does not pin its replay artifact by sha256"))
			claimInvalid = true
		} else if !validHex(claim.ReplaySHA256, 64) {
			return Result{}, fmt.Errorf("claim %d replay_sha256 must be lowercase sha256", index+1)
		}
		if len(claim.Fixtures) == 0 {
			result.Violations = append(result.Violations, violation(index, claim, "", "fixture-replay-required", "named-script replacement claim records no per-fixture exit-code replay"))
			claimInvalid = true
			continue
		}
		seenFixtures := map[string]bool{}
		for _, fixture := range claim.Fixtures {
			name := strings.TrimSpace(fixture.Name)
			if name == "" {
				result.Violations = append(result.Violations, violation(index, claim, "", "fixture-name-required", "replay fixture has no name"))
				claimInvalid = true
				continue
			}
			if seenFixtures[name] {
				return Result{}, fmt.Errorf("claim %d repeats fixture %q", index+1, name)
			}
			seenFixtures[name] = true
			if fixture.ScriptExit == nil || fixture.VerbExit == nil {
				result.Violations = append(result.Violations, violation(index, claim, name, "exit-codes-required", "replay fixture does not record both script and verb exit codes"))
				claimInvalid = true
				continue
			}
			if *fixture.ScriptExit < 0 || *fixture.ScriptExit > 255 || *fixture.VerbExit < 0 || *fixture.VerbExit > 255 {
				return Result{}, fmt.Errorf("claim %d fixture %q exit codes must be between 0 and 255", index+1, name)
			}
			if *fixture.ScriptExit != *fixture.VerbExit {
				text := fmt.Sprintf("script exit %d does not agree with verb exit %d", *fixture.ScriptExit, *fixture.VerbExit)
				result.Violations = append(result.Violations, violation(index, claim, name, "exit-code-agreement", text))
				claimInvalid = true
			}
		}
		if claimInvalid {
			continue
		}
		if readEvidence == nil {
			return Result{}, fmt.Errorf("claim %d cannot read replay evidence", index+1)
		}
		replayData, err := readEvidence(claim.ReplayCitation)
		if err != nil {
			return Result{}, fmt.Errorf("claim %d read replay evidence %q: %w", index+1, claim.ReplayCitation, err)
		}
		computedDigest := fmt.Sprintf("%x", sha256.Sum256(replayData))
		if computedDigest != claim.ReplaySHA256 {
			text := fmt.Sprintf("recorded sha256 %s does not match replay artifact sha256 %s", claim.ReplaySHA256, computedDigest)
			result.Violations = append(result.Violations, violation(index, claim, "", "dual-replay-digest", text))
			continue
		}
		replay, err := ParseReplay(replayData)
		if err != nil {
			return Result{}, fmt.Errorf("claim %d parse replay evidence %q: %w", index+1, claim.ReplayCitation, err)
		}
		record, err := selectReplayRecord(replay, claim.Verb, claim.Script)
		if err != nil {
			return Result{}, fmt.Errorf("claim %d: %w", index+1, err)
		}
		if claim.SelftestCitation != record.SelftestCitation {
			text := fmt.Sprintf("manifest selftest citation %q does not match replay record %q", claim.SelftestCitation, record.SelftestCitation)
			result.Violations = append(result.Violations, violation(index, claim, "", "selftest-citation-mismatch", text))
		}
		if claim.VocabularyCitation != record.VocabularyCitation {
			text := fmt.Sprintf("manifest vocabulary citation %q does not match replay record %q", claim.VocabularyCitation, record.VocabularyCitation)
			result.Violations = append(result.Violations, violation(index, claim, "", "vocabulary-citation-mismatch", text))
		}
		replayFixtures := make(map[string]Fixture, len(record.Fixtures))
		for _, fixture := range record.Fixtures {
			replayFixtures[fixture.Name] = fixture
		}
		for _, fixture := range claim.Fixtures {
			replayed, ok := replayFixtures[fixture.Name]
			if !ok {
				result.Violations = append(result.Violations, violation(index, claim, fixture.Name, "replay-fixture-absent", "manifest fixture is absent from the cited replay artifact"))
				continue
			}
			delete(replayFixtures, fixture.Name)
			if *fixture.ScriptExit != *replayed.ScriptExit || *fixture.VerbExit != *replayed.VerbExit {
				text := fmt.Sprintf("manifest exits %d/%d contradict cited replay exits %d/%d", *fixture.ScriptExit, *fixture.VerbExit, *replayed.ScriptExit, *replayed.VerbExit)
				result.Violations = append(result.Violations, violation(index, claim, fixture.Name, "replay-row-mismatch", text))
			}
		}
		for _, fixture := range record.Fixtures {
			if _, ok := replayFixtures[fixture.Name]; ok {
				result.Violations = append(result.Violations, violation(index, claim, fixture.Name, "replay-fixture-unrecorded", "cited replay fixture is absent from the manifest claim"))
			}
		}
	}
	if len(result.Violations) > 0 {
		result.Status = "fail"
	}
	return result, nil
}

func selectReplayRecord(replay ReplayEvidence, verb, script string) (ReplayRecord, error) {
	var selected ReplayRecord
	matches := 0
	for _, record := range replay.Records {
		if record.Verb == verb && record.Script == script {
			selected = record
			matches++
		}
	}
	if matches != 1 {
		return ReplayRecord{}, fmt.Errorf("replay evidence has %d records for verb %q and script %q, want 1", matches, verb, script)
	}
	return selected, nil
}

func validateExitPair(fixture Fixture) error {
	if fixture.ScriptExit == nil || fixture.VerbExit == nil {
		return fmt.Errorf("both script and verb exit codes are required")
	}
	if *fixture.ScriptExit < 0 || *fixture.ScriptExit > 255 || *fixture.VerbExit < 0 || *fixture.VerbExit > 255 {
		return fmt.Errorf("exit codes must be between 0 and 255")
	}
	return nil
}

func validHex(value string, length int) bool {
	if len(value) != length {
		return false
	}
	for _, char := range value {
		if (char < '0' || char > '9') && (char < 'a' || char > 'f') {
			return false
		}
	}
	return true
}

func violation(index int, claim Claim, fixture, rule, text string) Violation {
	return Violation{
		Entry:   index + 1,
		Verb:    claim.Verb,
		Script:  claim.Script,
		Status:  claim.Status,
		Fixture: fixture,
		Rule:    rule,
		Text:    text,
	}
}

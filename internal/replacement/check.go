package replacement

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

const (
	Schema         = "achta.replacement-check.v1"
	ManifestSchema = "achta.replacement-claims.v1"
	MaxManifest    = 1 << 20
)

type Manifest struct {
	SchemaVersion string  `json:"schema_version"`
	Claims        []Claim `json:"claims"`
}

type Claim struct {
	Verb             string    `json:"verb"`
	Script           string    `json:"script"`
	Status           string    `json:"status"`
	SelftestCitation string    `json:"selftest_citation,omitempty"`
	ReplayCitation   string    `json:"replay_citation,omitempty"`
	Fixtures         []Fixture `json:"fixtures,omitempty"`
}

type Fixture struct {
	Name       string `json:"name"`
	ScriptExit *int   `json:"script_exit"`
	VerbExit   *int   `json:"verb_exit"`
}

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
	if manifest.SchemaVersion != ManifestSchema {
		return Manifest{}, fmt.Errorf("schema_version must be %q", ManifestSchema)
	}
	for index, claim := range manifest.Claims {
		if strings.TrimSpace(claim.Verb) == "" || strings.TrimSpace(claim.Script) == "" || strings.TrimSpace(claim.Status) == "" {
			return Manifest{}, fmt.Errorf("claim %d requires non-empty verb, script, and status", index+1)
		}
	}
	return manifest, nil
}

func Check(manifest Manifest, claimStatuses []string) (Result, error) {
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
		if strings.TrimSpace(claim.SelftestCitation) == "" {
			result.Violations = append(result.Violations, violation(index, claim, "", "selftest-citation-required", "named-script replacement claim does not cite the script's own selftest fixtures"))
		}
		if strings.TrimSpace(claim.ReplayCitation) == "" {
			result.Violations = append(result.Violations, violation(index, claim, "", "dual-replay-citation-required", "named-script replacement claim does not cite a recorded replay of both implementations"))
		}
		if len(claim.Fixtures) == 0 {
			result.Violations = append(result.Violations, violation(index, claim, "", "fixture-replay-required", "named-script replacement claim records no per-fixture exit-code replay"))
			continue
		}
		seenFixtures := map[string]bool{}
		for _, fixture := range claim.Fixtures {
			name := strings.TrimSpace(fixture.Name)
			if name == "" {
				result.Violations = append(result.Violations, violation(index, claim, "", "fixture-name-required", "replay fixture has no name"))
				continue
			}
			if seenFixtures[name] {
				return Result{}, fmt.Errorf("claim %d repeats fixture %q", index+1, name)
			}
			seenFixtures[name] = true
			if fixture.ScriptExit == nil || fixture.VerbExit == nil {
				result.Violations = append(result.Violations, violation(index, claim, name, "exit-codes-required", "replay fixture does not record both script and verb exit codes"))
				continue
			}
			if *fixture.ScriptExit < 0 || *fixture.ScriptExit > 255 || *fixture.VerbExit < 0 || *fixture.VerbExit > 255 {
				return Result{}, fmt.Errorf("claim %d fixture %q exit codes must be between 0 and 255", index+1, name)
			}
			if *fixture.ScriptExit != *fixture.VerbExit {
				text := fmt.Sprintf("script exit %d does not agree with verb exit %d", *fixture.ScriptExit, *fixture.VerbExit)
				result.Violations = append(result.Violations, violation(index, claim, name, "exit-code-agreement", text))
			}
		}
	}
	if len(result.Violations) > 0 {
		result.Status = "fail"
	}
	return result, nil
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

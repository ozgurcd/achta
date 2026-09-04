package amendments

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"
)

const (
	Schema      = "ledger-amendments.v1"
	MaxManifest = int64(4 << 20)
	MaxReason   = 4096
)

var (
	fullSHA    = regexp.MustCompile(`^[0-9a-f]{40}$`)
	sha256Text = regexp.MustCompile(`^[0-9a-f]{64}$`)
	ruleID     = regexp.MustCompile(`^[A-Z0-9][A-Z0-9._-]*$`)
)

var validClasses = map[string]struct{}{
	"rule_added": {}, "rule_removed": {}, "sentence_changed": {}, "binding_changed": {},
	"proof_changed": {}, "covered_symbols_changed": {}, "test_fingerprint_changed": {},
}

type Manifest struct {
	SchemaVersion string   `json:"schema_version"`
	BaseCommit    string   `json:"base_commit"`
	Changes       []Change `json:"changes"`
}

type Change struct {
	RuleID              string `json:"rule_id"`
	ChangeClass         string `json:"change_class"`
	AfterSentenceSHA256 string `json:"after_sentence_sha256,omitempty"`
	Reason              string `json:"reason"`
}

func Parse(data []byte) (Manifest, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var manifest Manifest
	if err := decoder.Decode(&manifest); err != nil {
		return Manifest{}, fmt.Errorf("decode manifest: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return Manifest{}, errors.New("manifest contains more than one JSON document")
		}
		return Manifest{}, fmt.Errorf("trailing manifest data: %w", err)
	}
	if manifest.SchemaVersion != Schema {
		return Manifest{}, fmt.Errorf("unsupported manifest schema %q", manifest.SchemaVersion)
	}
	if !fullSHA.MatchString(manifest.BaseCommit) {
		return Manifest{}, errors.New("base_commit must be a full lowercase Git SHA")
	}
	seen := make(map[string]struct{})
	for i, change := range manifest.Changes {
		if err := validateChange(change); err != nil {
			return Manifest{}, fmt.Errorf("changes[%d]: %w", i, err)
		}
		key := change.RuleID + "\x00" + change.ChangeClass
		if _, exists := seen[key]; exists {
			return Manifest{}, fmt.Errorf("duplicate amendment %s/%s", change.RuleID, change.ChangeClass)
		}
		seen[key] = struct{}{}
	}
	return manifest, nil
}

func Declare(manifest Manifest, change Change) (Manifest, error) {
	if err := validateChange(change); err != nil {
		return Manifest{}, err
	}
	for _, existing := range manifest.Changes {
		if existing.RuleID == change.RuleID && existing.ChangeClass == change.ChangeClass {
			return Manifest{}, fmt.Errorf("amendment %s/%s already exists", change.RuleID, change.ChangeClass)
		}
	}
	manifest.Changes = append(manifest.Changes, change)
	sort.Slice(manifest.Changes, func(i, j int) bool {
		if manifest.Changes[i].RuleID != manifest.Changes[j].RuleID {
			return manifest.Changes[i].RuleID < manifest.Changes[j].RuleID
		}
		return manifest.Changes[i].ChangeClass < manifest.Changes[j].ChangeClass
	})
	return manifest, nil
}

func Rebase(manifest Manifest, base string) (Manifest, error) {
	if !fullSHA.MatchString(base) {
		return Manifest{}, errors.New("new base must be a full lowercase Git SHA")
	}
	if manifest.BaseCommit == base {
		return Manifest{}, errors.New("manifest already uses selected witness")
	}
	manifest.BaseCommit = base
	return manifest, nil
}

func Encode(manifest Manifest) ([]byte, error) {
	data, err := json.Marshal(manifest)
	if err != nil {
		return nil, err
	}
	if _, err := Parse(data); err != nil {
		return nil, err
	}
	data, err = json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return nil, err
	}
	data = append(data, '\n')
	if _, err := Parse(data); err != nil {
		return nil, fmt.Errorf("encoded manifest validation: %w", err)
	}
	return data, nil
}

func ValidateReason(reason string) error {
	if strings.TrimSpace(reason) == "" {
		return errors.New("reason must not be empty")
	}
	if len(reason) > MaxReason {
		return fmt.Errorf("reason exceeds %d-byte limit", MaxReason)
	}
	if strings.ContainsAny(reason, "\r\n\x00") {
		return errors.New("reason must be one line without NUL")
	}
	return nil
}

func validateChange(change Change) error {
	if !ruleID.MatchString(change.RuleID) {
		return errors.New("invalid rule ID")
	}
	if _, ok := validClasses[change.ChangeClass]; !ok {
		return fmt.Errorf("unsupported change class %q", change.ChangeClass)
	}
	if err := ValidateReason(change.Reason); err != nil {
		return err
	}
	if change.ChangeClass == "sentence_changed" {
		if !sha256Text.MatchString(change.AfterSentenceSHA256) {
			return errors.New("sentence_changed requires a lowercase 64-character after_sentence_sha256")
		}
	} else if change.AfterSentenceSHA256 != "" {
		return errors.New("after_sentence_sha256 is allowed only for sentence_changed")
	}
	return nil
}

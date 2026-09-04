package toolchain

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/ozgurcd/achta/internal/safefile"
)

const (
	ManifestSchema = "achta.toolchain-manifest.v1"
	ResultSchema   = "achta.toolchain-parity.v1"
	MaxManifest    = 1 << 20
	MaxWorkflow    = 4 << 20
)

var (
	namePattern   = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)
	envPattern    = regexp.MustCompile(`^[A-Z][A-Z0-9_]*$`)
	digestPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)
)

type VersionPin struct {
	Name  string `json:"name"`
	Env   string `json:"env"`
	Value string `json:"value"`
}

type DigestPin struct {
	Name  string `json:"name"`
	Env   string `json:"env"`
	Path  string `json:"path"`
	Value string `json:"value"`
}

type Manifest struct {
	SchemaVersion string       `json:"schema_version"`
	Versions      []VersionPin `json:"versions"`
	Digests       []DigestPin  `json:"digests"`
}

type PinCheck struct {
	Name      string `json:"name"`
	Kind      string `json:"kind"`
	Env       string `json:"env"`
	Workspace string `json:"workspace"`
	CI        string `json:"ci,omitempty"`
	Observed  string `json:"observed,omitempty"`
	Status    string `json:"status"`
}

type Result struct {
	SchemaVersion string     `json:"schema_version"`
	Status        string     `json:"status"`
	Pins          []PinCheck `json:"pins"`
}

func Check(repo string, manifestData, workflowData []byte) (Result, error) {
	manifest, err := parseManifest(manifestData)
	if err != nil {
		return Result{}, err
	}
	environment, err := parseWorkflowEnvironment(workflowData)
	if err != nil {
		return Result{}, err
	}
	result := Result{SchemaVersion: ResultSchema, Status: "pass", Pins: []PinCheck{}}
	for _, pin := range manifest.Versions {
		check := PinCheck{Name: pin.Name, Kind: "version", Env: pin.Env, Workspace: pin.Value, CI: environment[pin.Env], Status: "pass"}
		if check.CI == "" {
			check.Status = "missing_ci_pin"
		} else if check.Workspace != check.CI {
			check.Status = "mismatch"
		}
		result.Pins = append(result.Pins, check)
	}
	for _, pin := range manifest.Digests {
		path, err := confinedManifestPath(repo, pin.Path)
		if err != nil {
			return Result{}, fmt.Errorf("digest %s: %w", pin.Name, err)
		}
		snapshot, err := safefile.Read(repo, path, 16<<20)
		if err != nil {
			return Result{}, fmt.Errorf("digest %s: %w", pin.Name, err)
		}
		digest := sha256.Sum256(snapshot.Data)
		observed := hex.EncodeToString(digest[:])
		check := PinCheck{Name: pin.Name, Kind: "sha256", Env: pin.Env, Workspace: pin.Value, CI: environment[pin.Env], Observed: observed, Status: "pass"}
		if check.CI == "" {
			check.Status = "missing_ci_pin"
		} else if check.Workspace != check.CI || check.Workspace != check.Observed {
			check.Status = "mismatch"
		}
		result.Pins = append(result.Pins, check)
	}
	sort.Slice(result.Pins, func(i, j int) bool {
		if result.Pins[i].Name != result.Pins[j].Name {
			return result.Pins[i].Name < result.Pins[j].Name
		}
		return result.Pins[i].Kind < result.Pins[j].Kind
	})
	for _, pin := range result.Pins {
		if pin.Status != "pass" {
			result.Status = "fail"
		}
	}
	return result, nil
}

func parseManifest(data []byte) (Manifest, error) {
	if len(data) == 0 || len(data) > MaxManifest {
		return Manifest{}, errors.New("toolchain manifest size is invalid")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var manifest Manifest
	if err := decoder.Decode(&manifest); err != nil {
		return Manifest{}, fmt.Errorf("decode toolchain manifest: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return Manifest{}, errors.New("toolchain manifest must contain one JSON document")
	}
	if manifest.SchemaVersion != ManifestSchema {
		return Manifest{}, fmt.Errorf("unsupported toolchain manifest schema %q", manifest.SchemaVersion)
	}
	if len(manifest.Versions)+len(manifest.Digests) == 0 {
		return Manifest{}, errors.New("toolchain manifest has no pins")
	}
	seenName := make(map[string]bool)
	seenEnv := make(map[string]bool)
	for _, pin := range manifest.Versions {
		if err := validatePin(pin.Name, pin.Env, pin.Value, seenName, seenEnv); err != nil {
			return Manifest{}, err
		}
	}
	for _, pin := range manifest.Digests {
		if err := validatePin(pin.Name, pin.Env, pin.Value, seenName, seenEnv); err != nil {
			return Manifest{}, err
		}
		if pin.Path == "" || strings.ContainsAny(pin.Path, "\x00\r\n") || !digestPattern.MatchString(pin.Value) {
			return Manifest{}, fmt.Errorf("invalid digest pin %q", pin.Name)
		}
	}
	return manifest, nil
}

func validatePin(name, environment, value string, seenName, seenEnv map[string]bool) error {
	if !namePattern.MatchString(name) || !envPattern.MatchString(environment) || value == "" || len(value) > 512 || strings.ContainsAny(value, "\x00\r\n") {
		return fmt.Errorf("invalid toolchain pin %q", name)
	}
	if seenName[name] || seenEnv[environment] {
		return fmt.Errorf("duplicate toolchain pin %q or environment %q", name, environment)
	}
	seenName[name], seenEnv[environment] = true, true
	return nil
}

func parseWorkflowEnvironment(data []byte) (map[string]string, error) {
	if len(data) == 0 || len(data) > MaxWorkflow {
		return nil, errors.New("workflow size is invalid")
	}
	result := make(map[string]string)
	inEnvironment := false
	for _, raw := range strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n") {
		line := strings.TrimRight(raw, " \t")
		if line == "env:" {
			if inEnvironment || len(result) > 0 {
				return nil, errors.New("workflow has multiple top-level env mappings")
			}
			inEnvironment = true
			continue
		}
		if !inEnvironment {
			continue
		}
		if line == "" || strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}
		if line[0] != ' ' && line[0] != '\t' {
			inEnvironment = false
			continue
		}
		trimmed := strings.TrimSpace(line)
		key, value, ok := strings.Cut(trimmed, ":")
		if !ok || !envPattern.MatchString(key) {
			continue
		}
		value = strings.TrimSpace(value)
		if len(value) >= 2 && ((value[0] == '"' && value[len(value)-1] == '"') || (value[0] == '\'' && value[len(value)-1] == '\'')) {
			value = value[1 : len(value)-1]
		}
		if value == "" || strings.ContainsAny(value, "\x00\r\n") {
			return nil, fmt.Errorf("workflow environment %s has an unsupported value", key)
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("workflow environment %s is duplicated", key)
		}
		result[key] = value
	}
	if len(result) == 0 {
		return nil, errors.New("workflow has no parseable top-level environment pins")
	}
	return result, nil
}

func confinedManifestPath(repo, value string) (string, error) {
	if value == "" || filepath.IsAbs(value) || strings.Contains(value, "\\") {
		return "", errors.New("digest path must be repository-relative")
	}
	clean := filepath.Clean(filepath.FromSlash(value))
	if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", errors.New("digest path escapes repository")
	}
	return filepath.Join(repo, clean), nil
}

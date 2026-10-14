// Package repopolicy separates registered repository policy from checkout claims.
package repopolicy

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"syscall"

	"github.com/ozgurcd/achta/internal/gitstate"
)

const (
	RegistrySchema = "achta.repo-registry.v1"
	PolicySchema   = "achta.repo-policy.v1"
	ReadSchema     = "achta.repo-policy-read.v1"
)

type Checkout struct {
	Path         string `json:"path"`
	GitDir       string `json:"git_dir"`
	PathIdentity string `json:"path_identity"`
	GitIdentity  string `json:"git_identity"`
}

type Registry struct {
	SchemaVersion string     `json:"schema_version"`
	RepoID        string     `json:"repo_id"`
	Checkouts     []Checkout `json:"checkouts"`
}

type Reference struct {
	ID     string `json:"id"`
	SHA256 string `json:"sha256"`
}

type Policy struct {
	SchemaVersion   string    `json:"schema_version"`
	RepoID          string    `json:"repo_id"`
	RegistrySHA256  string    `json:"registry_sha256"`
	DesiredMode     string    `json:"desired_mode"`
	EffectiveMode   string    `json:"effective_mode"`
	Epoch           uint64    `json:"epoch"`
	WriterLimit     uint32    `json:"writer_limit"`
	PathKeys        []string  `json:"path_keys"`
	SemanticKeys    []string  `json:"semantic_keys"`
	EligibleRunners []string  `json:"eligible_runners"`
	Capability      Reference `json:"capability"`
	NativeProfile   Reference `json:"native_profile"`
	OwnerAction     string    `json:"owner_action"`
}

// Verify must authenticate exact bytes using independently protected custody.
// A1 supplies no production verifier, setter or protected-history rollback check.
type Options struct {
	RepoID       string
	CheckoutPath string
	RegistryPath string
	PolicyPath   string
	MinimumEpoch uint64
	Verify       func(schema string, data []byte) error
}

type Projection struct {
	SchemaVersion string  `json:"schema_version"`
	Status        string  `json:"status"`
	RepoID        string  `json:"repo_id"`
	Registered    bool    `json:"registered"`
	PolicyPresent bool    `json:"policy_present"`
	DesiredMode   string  `json:"desired_mode"`
	EffectiveMode string  `json:"effective_mode"`
	Epoch         uint64  `json:"epoch"`
	Policy        *Policy `json:"policy,omitempty"`
}

func Read(opts Options) (Projection, error) {
	r := Projection{SchemaVersion: ReadSchema, Status: "cannot_evaluate"}
	if !validID(opts.RepoID) {
		return r, errors.New("registered repository ID is required")
	}
	if _, err := exactPath(opts.CheckoutPath); err != nil {
		return r, errors.New("unsafe checkout selection")
	}
	if _, err := exactPath(opts.RegistryPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return r, errors.New("unsafe registry selection")
	}
	data, err := readBounded(opts.PolicyPath)
	if errors.Is(err, os.ErrNotExist) {
		return Projection{SchemaVersion: ReadSchema, Status: "pass", RepoID: opts.RepoID, DesiredMode: "off", EffectiveMode: "off"}, nil
	}
	if err != nil {
		return r, errors.New("policy evidence unavailable or unsafe")
	}
	policy, err := ParsePolicy(data)
	if err != nil {
		return r, err
	}
	registryData, err := readBounded(opts.RegistryPath)
	if err != nil {
		return r, errors.New("registry evidence unavailable or unsafe")
	}
	registry, err := ParseRegistry(registryData)
	if err != nil {
		return r, err
	}
	if policy.RepoID != opts.RepoID || registry.RepoID != opts.RepoID {
		return r, errors.New("registered repository mismatch")
	}
	digest := sha256.Sum256(registryData)
	if policy.RegistrySHA256 != hex.EncodeToString(digest[:]) {
		return r, errors.New("registry binding mismatch")
	}
	if policy.Epoch < opts.MinimumEpoch {
		return r, errors.New("epoch below injected minimum")
	}
	if opts.Verify == nil || opts.Verify(RegistrySchema, registryData) != nil || opts.Verify(PolicySchema, data) != nil {
		return r, errors.New("owner authentication unavailable or refused")
	}
	observed, err := ObserveCheckout(opts.CheckoutPath)
	if err != nil {
		return r, err
	}
	found := false
	for _, c := range registry.Checkouts {
		if c == observed {
			found = true
		}
	}
	if !found {
		return r, errors.New("checkout is not exactly registered")
	}
	// Re-observe after authentication/identity checks; no repaired or mixed snapshot.
	currentPolicy, err := readBounded(opts.PolicyPath)
	if err != nil || !bytes.Equal(currentPolicy, data) {
		return r, errors.New("policy changed during read")
	}
	currentRegistry, err := readBounded(opts.RegistryPath)
	if err != nil || !bytes.Equal(currentRegistry, registryData) {
		return r, errors.New("registry changed during read")
	}
	currentCheckout, err := ObserveCheckout(opts.CheckoutPath)
	if err != nil || currentCheckout != observed {
		return r, errors.New("checkout changed during read")
	}
	return Projection{SchemaVersion: ReadSchema, Status: "pass", RepoID: opts.RepoID, Registered: true, PolicyPresent: true, DesiredMode: policy.DesiredMode, EffectiveMode: policy.EffectiveMode, Epoch: policy.Epoch, Policy: &policy}, nil
}

var idPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,127}$`)
var digestPattern = regexp.MustCompile(`^[a-f0-9]{64}$`)
var identityPattern = regexp.MustCompile(`^[0-9]+:[0-9]+$`)

func validID(value string) bool { return idPattern.MatchString(value) }

// ParsePolicy and ParseRegistry validate a closed shape, not authentication.
func ParsePolicy(data []byte) (Policy, error) {
	var p Policy
	if err := decode(data, &p); err != nil {
		return Policy{}, errors.New("malformed closed policy")
	}
	if p.SchemaVersion != PolicySchema || !validID(p.RepoID) || !digestPattern.MatchString(p.RegistrySHA256) || p.Epoch == 0 || p.WriterLimit == 0 || p.WriterLimit > 64 || !validID(p.OwnerAction) {
		return Policy{}, errors.New("unsupported or invalid policy")
	}
	validMode := p.DesiredMode == "off" && (p.EffectiveMode == "off" || p.EffectiveMode == "disabling") || p.DesiredMode == "on" && (p.EffectiveMode == "on" || p.EffectiveMode == "enabling")
	if !validMode || !validKeys(p.PathKeys) || !validKeys(p.SemanticKeys) || !validKeys(p.EligibleRunners) {
		return Policy{}, errors.New("invalid closed policy modes or keys")
	}
	for _, ref := range []Reference{p.Capability, p.NativeProfile} {
		if !validID(ref.ID) || !digestPattern.MatchString(ref.SHA256) {
			return Policy{}, errors.New("invalid capability or native profile reference")
		}
	}
	return p, nil
}

func ParseRegistry(data []byte) (Registry, error) {
	var r Registry
	if err := decode(data, &r); err != nil {
		return Registry{}, errors.New("malformed closed registry")
	}
	if r.SchemaVersion != RegistrySchema || !validID(r.RepoID) || len(r.Checkouts) == 0 || len(r.Checkouts) > 128 {
		return Registry{}, errors.New("unsupported or invalid registry")
	}
	paths, dirs, identities := map[string]bool{}, map[string]bool{}, map[string]bool{}
	for _, c := range r.Checkouts {
		if !lexicalPath(c.Path) || !lexicalPath(c.GitDir) || !identityPattern.MatchString(c.PathIdentity) || !identityPattern.MatchString(c.GitIdentity) || paths[c.Path] || dirs[c.GitDir] || identities[c.PathIdentity] {
			return Registry{}, errors.New("invalid or repeated checkout registration")
		}
		paths[c.Path], dirs[c.GitDir], identities[c.PathIdentity] = true, true, true
	}
	return r, nil
}

func validKeys(keys []string) bool {
	if keys == nil || len(keys) > 128 {
		return false
	}
	seen := map[string]bool{}
	for _, key := range keys {
		if !validID(key) || seen[key] {
			return false
		}
		seen[key] = true
	}
	return true
}

func decode(data []byte, target any) error {
	if len(data) == 0 || len(data) > 65536 {
		return errors.New("input size")
	}
	// encoding/json otherwise accepts duplicate object keys and null scalars.
	tokens := json.NewDecoder(bytes.NewReader(data))
	if err := uniqueValue(tokens); err != nil {
		return err
	}
	if _, err := tokens.Token(); err != io.EOF {
		return errors.New("trailing value")
	}
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if err := exactFields(data, reflect.TypeOf(target).Elem()); err != nil {
		return err
	}
	return d.Decode(target)
}

// Unlike encoding/json's matching, closed field names are case sensitive and
// every member is required, even when its valid value is an empty array.
func exactFields(data []byte, shape reflect.Type) error {
	switch shape.Kind() {
	case reflect.Struct:
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(data, &fields); err != nil {
			return err
		}
		if len(fields) != shape.NumField() {
			return errors.New("closed field set mismatch")
		}
		for i := 0; i < shape.NumField(); i++ {
			field := shape.Field(i)
			value, ok := fields[field.Tag.Get("json")]
			if !ok {
				return errors.New("missing exact field")
			}
			if err := exactFields(value, field.Type); err != nil {
				return err
			}
		}
	case reflect.Slice:
		var values []json.RawMessage
		if err := json.Unmarshal(data, &values); err != nil {
			return err
		}
		for _, value := range values {
			if err := exactFields(value, shape.Elem()); err != nil {
				return err
			}
		}
	}
	return nil
}

func uniqueValue(d *json.Decoder) error {
	token, err := d.Token()
	if err != nil {
		return err
	}
	if token == nil {
		return errors.New("null field")
	}
	delim, ok := token.(json.Delim)
	if !ok {
		return nil
	}
	if delim != '{' && delim != '[' {
		return errors.New("unexpected delimiter")
	}
	seen := map[string]bool{}
	for d.More() {
		if delim == '{' {
			key, err := d.Token()
			if err != nil {
				return err
			}
			name, ok := key.(string)
			if !ok || seen[name] {
				return errors.New("duplicate field")
			}
			seen[name] = true
		}
		if err := uniqueValue(d); err != nil {
			return err
		}
	}
	_, err = d.Token()
	return err
}

func lexicalPath(path string) bool {
	return filepath.IsAbs(path) && filepath.Clean(path) == path && !strings.ContainsAny(path, "\x00\r\n")
}

// exactPath refuses links and case aliases, including existing ancestors of a
// missing file. It never creates directories. Provisioning supplies canonical paths.
func exactPath(path string) (os.FileInfo, error) {
	if !lexicalPath(path) {
		return nil, errors.New("path must be exact absolute")
	}
	current := string(filepath.Separator)
	var info os.FileInfo
	for _, part := range strings.Split(strings.TrimPrefix(path, current), current) {
		if part == "" {
			continue
		}
		entries, err := os.ReadDir(current)
		if err != nil {
			return nil, err
		}
		found := false
		for _, e := range entries {
			if e.Name() == part {
				found = true
				break
			}
		}
		if !found {
			if _, err := os.Lstat(filepath.Join(current, part)); err == nil {
				return nil, errors.New("case-aliased path")
			}
			return nil, os.ErrNotExist
		}
		current = filepath.Join(current, part)
		info, err = os.Lstat(current)
		if err != nil {
			return nil, err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return nil, errors.New("linked path")
		}
	}
	if info == nil {
		return os.Lstat(current)
	}
	return info, nil
}

func readBounded(path string) ([]byte, error) {
	for _, part := range strings.Split(path, string(filepath.Separator)) {
		if strings.HasPrefix(part, ".env") || part == "dev.env" || strings.HasPrefix(part, "dev.env.") {
			return nil, errors.New("credential-like path refused")
		}
	}
	info, err := exactPath(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() > 65536 {
		return nil, errors.New("unsafe input type or size")
	}
	f, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	opened, err := f.Stat()
	if err != nil || !os.SameFile(info, opened) {
		return nil, errors.New("input substituted")
	}
	data, err := io.ReadAll(io.LimitReader(f, 65537))
	if err != nil || len(data) > 65536 {
		return nil, errors.New("input read refused")
	}
	final, err := exactPath(path)
	if err != nil || !os.SameFile(info, final) || final.Size() != int64(len(data)) || final.ModTime() != info.ModTime() {
		return nil, errors.New("input changed during read")
	}
	return data, nil
}

// ObserveCheckout supplies identity facts only; it grants no registration.
func ObserveCheckout(path string) (Checkout, error) {
	root, err := exactPath(path)
	if err != nil || !root.IsDir() {
		return Checkout{}, errors.New("unsafe checkout root")
	}
	dir, err := gitstate.CheckoutGitDir(path)
	if err != nil {
		return Checkout{}, errors.New("checkout Git identity unavailable")
	}
	metadata, err := exactPath(dir)
	if err != nil || !metadata.IsDir() {
		return Checkout{}, errors.New("unsafe checkout Git directory")
	}
	rootStat, rootOK := root.Sys().(*syscall.Stat_t)
	dirStat, dirOK := metadata.Sys().(*syscall.Stat_t)
	if !rootOK || !dirOK {
		return Checkout{}, errors.New("filesystem identity unavailable")
	}
	return Checkout{Path: path, GitDir: dir, PathIdentity: fmt.Sprintf("%d:%d", rootStat.Dev, rootStat.Ino), GitIdentity: fmt.Sprintf("%d:%d", dirStat.Dev, dirStat.Ino)}, nil
}

package cli

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ozgurcd/achta/internal/repopolicy"
)

func TestPolicyReadMissingCreatesNothing(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	policy := filepath.Join(root, "policy.json")
	registry := filepath.Join(root, "registry.json")
	var out, diagnostics bytes.Buffer
	code := Run([]string{"policy", "read", "--repo-id", "repo-a", "--repo", root, "--policy", policy, "--registry", registry, "--json"}, &out, &diagnostics, testVersion)
	if code != 0 || !strings.Contains(out.String(), `"effective_mode":"off"`) {
		t.Fatalf("missing policy must be off: exit %d", code)
	}
	entries, err := os.ReadDir(root)
	if err != nil || len(entries) != 0 {
		t.Fatal("read created state")
	}
}

func TestPolicyReadHelpAndCapability(t *testing.T) {
	var out, diagnostics bytes.Buffer
	if Run([]string{"policy", "read", "--help"}, &out, &diagnostics, testVersion) != 0 || !strings.Contains(out.String(), "no production trust verifier") {
		t.Fatal("missing typed policy help")
	}
	out.Reset()
	if Run([]string{"capabilities", "--json"}, &out, &diagnostics, testVersion) != 0 || !strings.Contains(out.String(), `"name":"policy read","reads":true,"writes":false`) {
		t.Fatal("missing read-only capability")
	}
}

func TestPolicyReadCannotTrustBranchOrNuntiusConfig(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	registry := repopolicy.Registry{SchemaVersion: repopolicy.RegistrySchema, RepoID: "repo-a", Checkouts: []repopolicy.Checkout{{Path: root, GitDir: filepath.Join(root, ".git"), PathIdentity: "1:2", GitIdentity: "1:3"}}}
	data, _ := json.Marshal(registry)
	registryPath := filepath.Join(root, "registry.json")
	if err := os.WriteFile(registryPath, data, 0600); err != nil {
		t.Fatal(err)
	}
	digest := fmt.Sprintf("%x", sha256.Sum256(data))
	policy := repopolicy.Policy{SchemaVersion: repopolicy.PolicySchema, RepoID: "repo-a", RegistrySHA256: digest, DesiredMode: "on", EffectiveMode: "on", Epoch: 3, WriterLimit: 2, PathKeys: []string{}, SemanticKeys: []string{}, EligibleRunners: []string{}, Capability: repopolicy.Reference{ID: "caps", SHA256: digest}, NativeProfile: repopolicy.Reference{ID: "profile", SHA256: digest}, OwnerAction: "fixture-action"}
	data, _ = json.Marshal(policy)
	for _, name := range []string{"branch-policy.json", "nuntius-config.json"} {
		path := filepath.Join(root, name)
		if err := os.WriteFile(path, data, 0600); err != nil {
			t.Fatal(err)
		}
		var out, diagnostics bytes.Buffer
		code := Run([]string{"policy", "read", "--repo-id", "repo-a", "--repo", root, "--registry", registryPath, "--policy", path, "--json"}, &out, &diagnostics, testVersion)
		if code != 2 || !strings.Contains(out.String(), "owner authentication unavailable or refused") || strings.Contains(out.String(), `"effective_mode":"on"`) {
			t.Fatal("caller file became owner authority")
		}
		observed, err := os.ReadFile(path)
		if err != nil || !bytes.Equal(data, observed) {
			t.Fatal("refusal repaired caller input")
		}
	}
}

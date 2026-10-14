package repopolicy

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

type fixture struct {
	root     string
	opts     Options
	registry Registry
	policy   Policy
}

func identity(t *testing.T, path string) string {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	st := info.Sys().(*syscall.Stat_t)
	return fmt.Sprintf("%d:%d", st.Dev, st.Ino)
}

func checkout(t *testing.T, root, name string) Checkout {
	t.Helper()
	path := filepath.Join(root, name)
	if err := os.Mkdir(path, 0700); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("git", "init", "--quiet", path)
	if err := cmd.Run(); err != nil {
		t.Fatal("fixture git init failed")
	}
	dir := filepath.Join(path, ".git")
	return Checkout{Path: path, GitDir: dir, PathIdentity: identity(t, path), GitIdentity: identity(t, dir)}
}

func newFixture(t *testing.T) fixture {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	a, b := checkout(t, root, "clone-a"), checkout(t, root, "clone-b")
	f := fixture{root: root, registry: Registry{SchemaVersion: RegistrySchema, RepoID: "repo-a", Checkouts: []Checkout{a, b}}}
	f.opts = Options{RepoID: "repo-a", CheckoutPath: a.Path, RegistryPath: filepath.Join(root, "registry.json"), PolicyPath: filepath.Join(root, "policy.json"), MinimumEpoch: 3}
	digest := strings.Repeat("a", 64)
	f.policy = Policy{SchemaVersion: PolicySchema, RepoID: "repo-a", DesiredMode: "on", EffectiveMode: "on", Epoch: 3, WriterLimit: 2, PathKeys: []string{"source"}, SemanticKeys: []string{"api"}, EligibleRunners: []string{"local"}, Capability: Reference{ID: "source-v1", SHA256: digest}, NativeProfile: Reference{ID: "native-v1", SHA256: digest}, OwnerAction: "action-3"}
	f.save(t)
	// Exact pinned fixture bytes are trusted only by this injected verifier.
	registryBytes, _ := os.ReadFile(f.opts.RegistryPath)
	policyBytes, _ := os.ReadFile(f.opts.PolicyPath)
	f.opts.Verify = func(schema string, data []byte) error {
		if schema == RegistrySchema && bytes.Equal(data, registryBytes) || schema == PolicySchema && bytes.Equal(data, policyBytes) {
			return nil
		}
		return errors.New("fixture authentication refused")
	}
	return f
}

func (f *fixture) save(t *testing.T) {
	t.Helper()
	data, err := json.Marshal(f.registry)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(f.opts.RegistryPath, data, 0600); err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(data)
	f.policy.RegistrySHA256 = hex.EncodeToString(digest[:])
	data, err = json.Marshal(f.policy)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(f.opts.PolicyPath, data, 0600); err != nil {
		t.Fatal(err)
	}
}

func snapshot(t *testing.T, root string) map[string]string {
	t.Helper()
	result := map[string]string{}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		value := fmt.Sprintf("%v:%d:%d", info.Mode(), info.Size(), info.ModTime().UnixNano())
		if info.Mode().IsRegular() {
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			value += fmt.Sprintf(":%x", sha256.Sum256(data))
		}
		result[path] = value
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func TestRegisteredClonesAndReadOnly(t *testing.T) {
	f := newFixture(t)
	before, _ := json.Marshal(snapshot(t, f.root))
	for _, c := range f.registry.Checkouts {
		opts := f.opts
		opts.CheckoutPath = c.Path
		got, err := Read(opts)
		if err != nil || !got.Registered || got.RepoID != "repo-a" || got.Epoch != 3 || got.Policy == nil || got.Policy.NativeProfile != f.policy.NativeProfile {
			t.Fatal("registered clone valid control failed")
		}
	}
	after, _ := json.Marshal(snapshot(t, f.root))
	if !bytes.Equal(before, after) {
		t.Fatal("policy read changed filesystem state")
	}
}

func TestPolicyRefusals(t *testing.T) {
	cases := []struct {
		name   string
		change func(*testing.T, *fixture)
	}{
		{"malformed", func(t *testing.T, f *fixture) { write(t, f.opts.PolicyPath, `{`) }},
		{"unknown", func(t *testing.T, f *fixture) { appendField(t, f.opts.PolicyPath, `,"extra":true`) }},
		{"nested-unknown", func(t *testing.T, f *fixture) {
			data, _ := os.ReadFile(f.opts.PolicyPath)
			write(t, f.opts.PolicyPath, strings.Replace(string(data), `"id":"source-v1"`, `"id":"source-v1","extra":true`, 1))
		}},
		{"duplicate", func(t *testing.T, f *fixture) { appendField(t, f.opts.PolicyPath, `,"epoch":3`) }},
		{"null", func(t *testing.T, f *fixture) {
			data, _ := os.ReadFile(f.opts.PolicyPath)
			write(t, f.opts.PolicyPath, strings.Replace(string(data), `"epoch":3`, `"epoch":null`, 1))
		}},
		{"trailing", func(t *testing.T, f *fixture) {
			data, _ := os.ReadFile(f.opts.PolicyPath)
			write(t, f.opts.PolicyPath, string(data)+` {}`)
		}},
		{"unsupported", func(t *testing.T, f *fixture) { f.policy.SchemaVersion = "achta.repo-policy.v2"; f.save(t) }},
		{"wrong-repo", func(t *testing.T, f *fixture) { f.opts.RepoID = "repo-b" }},
		{"wrong-registry-repo", func(t *testing.T, f *fixture) { f.registry.RepoID = "repo-b"; f.save(t) }},
		{"old-epoch", func(t *testing.T, f *fixture) { f.opts.MinimumEpoch = 4 }},
		{"untrusted", func(t *testing.T, f *fixture) { f.opts.Verify = nil }},
		{"rejected-trust", func(t *testing.T, f *fixture) {
			f.opts.Verify = func(string, []byte) error { return errors.New("refused") }
		}},
		{"registry-unavailable", func(t *testing.T, f *fixture) { f.opts.RegistryPath += "-absent" }},
		{"unregistered-clone", func(t *testing.T, f *fixture) { f.opts.CheckoutPath = checkout(t, f.root, "clone-c").Path }},
		{"case-alias", func(t *testing.T, f *fixture) {
			f.opts.CheckoutPath = strings.Replace(f.opts.CheckoutPath, "clone-a", "CLONE-A", 1)
		}},
		{"symlink-checkout", func(t *testing.T, f *fixture) {
			alias := filepath.Join(f.root, "alias")
			if err := os.Symlink(f.opts.CheckoutPath, alias); err != nil {
				t.Fatal(err)
			}
			f.opts.CheckoutPath = alias
		}},
		{"checkout-substitution", func(t *testing.T, f *fixture) {
			path := f.opts.CheckoutPath
			if err := os.Rename(path, path+"-old"); err != nil {
				t.Fatal(err)
			}
			checkout(t, f.root, "clone-a")
		}},
		{"gitdir-substitution", func(t *testing.T, f *fixture) {
			path := f.registry.Checkouts[0].GitDir
			if err := os.Rename(path, path+"-old"); err != nil {
				t.Fatal(err)
			}
			if err := exec.Command("git", "init", "--quiet", f.opts.CheckoutPath).Run(); err != nil {
				t.Fatal(err)
			}
		}},
		{"symlink-policy", func(t *testing.T, f *fixture) {
			path := f.opts.PolicyPath
			if err := os.Rename(path, path+"-real"); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(path+"-real", path); err != nil {
				t.Fatal(err)
			}
		}},
		{"symlink-missing", func(t *testing.T, f *fixture) {
			f.opts.PolicyPath += "-link"
			if err := os.Symlink(filepath.Join(f.root, "absent"), f.opts.PolicyPath); err != nil {
				t.Fatal(err)
			}
		}},
		{"directory-policy", func(t *testing.T, f *fixture) { f.opts.PolicyPath = f.root }},
		{"invalid-mode", func(t *testing.T, f *fixture) { f.policy.EffectiveMode = "arbitrary"; f.save(t) }},
		{"inconsistent-mode", func(t *testing.T, f *fixture) { f.policy.DesiredMode = "off"; f.save(t) }},
		{"invalid-reference", func(t *testing.T, f *fixture) { f.policy.NativeProfile.SHA256 = "bad"; f.save(t) }},
		{"zero-limit", func(t *testing.T, f *fixture) { f.policy.WriterLimit = 0; f.save(t) }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(t)
			tc.change(t, &f)
			if tc.name != "untrusted" && tc.name != "rejected-trust" {
				// Authentication cannot mask schema/identity rejection predicates.
				registryBytes, _ := os.ReadFile(f.opts.RegistryPath)
				policyBytes, _ := os.ReadFile(f.opts.PolicyPath)
				f.opts.Verify = func(schema string, data []byte) error {
					if schema == RegistrySchema && bytes.Equal(data, registryBytes) || schema == PolicySchema && bytes.Equal(data, policyBytes) {
						return nil
					}
					return errors.New("fixture authentication refused")
				}
			}
			before, _ := json.Marshal(snapshot(t, f.root))
			if got, err := Read(f.opts); err == nil || got.Status != "cannot_evaluate" || got.Policy != nil || got.Registered {
				t.Fatal("unsafe policy accepted")
			}
			after, _ := json.Marshal(snapshot(t, f.root))
			if !bytes.Equal(before, after) {
				t.Fatal("refusal repaired or wrote input")
			}
		})
	}
}

func write(t *testing.T, path, text string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(text), 0600); err != nil {
		t.Fatal(err)
	}
}
func appendField(t *testing.T, path, field string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	write(t, path, strings.TrimSuffix(string(data), "}")+field+"}")
}

func TestMissingPolicyOff(t *testing.T) {
	f := newFixture(t)
	f.opts.PolicyPath = filepath.Join(f.root, "unprovisioned", "policy.json")
	f.opts.Verify = nil
	before, _ := json.Marshal(snapshot(t, f.root))
	got, err := Read(f.opts)
	if err != nil || got.PolicyPresent || got.Registered || got.Policy != nil || got.DesiredMode != "off" || got.EffectiveMode != "off" || got.Epoch != 0 {
		t.Fatal("missing policy must be off")
	}
	after, _ := json.Marshal(snapshot(t, f.root))
	if !bytes.Equal(before, after) {
		t.Fatal("missing read created state")
	}
}

func TestClosedFieldsRequireExactNamesAndPresence(t *testing.T) {
	f := newFixture(t)
	policy, _ := os.ReadFile(f.opts.PolicyPath)
	registry, _ := os.ReadFile(f.opts.RegistryPath)
	for _, tc := range []struct {
		name     string
		data     string
		registry bool
	}{
		{"case-policy", strings.Replace(string(policy), `"epoch"`, `"EPOCH"`, 1), false},
		{"missing-empty-array", strings.Replace(string(policy), `"semantic_keys":["api"],`, "", 1), false},
		{"case-registry", strings.Replace(string(registry), `"repo_id"`, `"REPO_ID"`, 1), true},
		{"unknown-registry", strings.TrimSuffix(string(registry), "}") + `,"extra":false}`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var err error
			if tc.registry {
				_, err = ParseRegistry([]byte(tc.data))
			} else {
				_, err = ParsePolicy([]byte(tc.data))
			}
			if err == nil {
				t.Fatal("noncanonical field accepted")
			}
		})
	}
}

func TestTransitionsAndInjectedMinimum(t *testing.T) {
	for _, modes := range [][2]string{{"off", "off"}, {"on", "enabling"}, {"on", "on"}, {"off", "disabling"}} {
		f := newFixture(t)
		f.policy.DesiredMode, f.policy.EffectiveMode = modes[0], modes[1]
		f.save(t)
		registryBytes, _ := os.ReadFile(f.opts.RegistryPath)
		policyBytes, _ := os.ReadFile(f.opts.PolicyPath)
		f.opts.Verify = func(schema string, data []byte) error {
			if schema == RegistrySchema && bytes.Equal(data, registryBytes) || schema == PolicySchema && bytes.Equal(data, policyBytes) {
				return nil
			}
			return errors.New("fixture refused")
		}
		got, err := Read(f.opts)
		if err != nil || got.DesiredMode != modes[0] || got.EffectiveMode != modes[1] {
			t.Fatal("valid closed transition refused")
		}
		f.opts.MinimumEpoch = f.policy.Epoch + 1
		if _, err := Read(f.opts); err == nil {
			t.Fatal("injected epoch minimum ignored")
		}
	}
}

func TestSharedCommonDirDoesNotRegisterWorktree(t *testing.T) {
	f := newFixture(t)
	path := filepath.Join(f.root, "worktree")
	cmd := exec.Command("git", "-C", f.opts.CheckoutPath, "-c", "user.name=Fixture", "-c", "user.email=fixture@example.invalid", "-c", "core.hooksPath=/dev/null", "commit", "--quiet", "--allow-empty", "-m", "fixture")
	if err := cmd.Run(); err != nil {
		t.Fatal("fixture commit failed")
	}
	if err := exec.Command("git", "-C", f.opts.CheckoutPath, "-c", "core.hooksPath=/dev/null", "worktree", "add", "--quiet", "-b", "fixture", path).Run(); err != nil {
		t.Fatal("fixture worktree failed")
	}
	f.opts.CheckoutPath = path
	if _, err := Read(f.opts); err == nil {
		t.Fatal("common Git directory created registration")
	}
	observed, err := ObserveCheckout(path)
	if err != nil {
		t.Fatal("worktree identity unavailable")
	}
	f.registry.Checkouts = append(f.registry.Checkouts, observed)
	f.save(t)
	registryBytes, _ := os.ReadFile(f.opts.RegistryPath)
	policyBytes, _ := os.ReadFile(f.opts.PolicyPath)
	f.opts.Verify = func(schema string, data []byte) error {
		if schema == RegistrySchema && bytes.Equal(data, registryBytes) || schema == PolicySchema && bytes.Equal(data, policyBytes) {
			return nil
		}
		return errors.New("fixture refused")
	}
	if got, err := Read(f.opts); err != nil || !got.Registered {
		t.Fatal("explicitly authenticated worktree registration refused")
	}
}

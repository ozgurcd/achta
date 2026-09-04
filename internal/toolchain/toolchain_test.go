package toolchain

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// RULE: TOOLCHAIN-PARITY-1
func TestCheckVersionAndDigestParity(t *testing.T) {
	repo := t.TempDir()
	script := []byte("#!/bin/sh\nexit 0\n")
	if err := os.WriteFile(filepath.Join(repo, "gate.sh"), script, 0o644); err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(script)
	value := hex.EncodeToString(digest[:])
	manifest := []byte(fmt.Sprintf(`{"schema_version":"achta.toolchain-manifest.v1","versions":[{"name":"rulefloor","env":"RULEFLOOR_VERSION","value":"v0.9.0"}],"digests":[{"name":"gate","env":"GATE_SHA256","path":"gate.sh","value":"%s"}]}`, value))
	workflow := []byte(fmt.Sprintf("name: verify\nenv:\n  RULEFLOOR_VERSION: v0.9.0\n  GATE_SHA256: %s\njobs: {}\n", value))
	result, err := Check(repo, manifest, workflow)
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "pass" || len(result.Pins) != 2 {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestCheckFailsOnMissingOrMismatchedCIPin(t *testing.T) {
	repo := t.TempDir()
	manifest := []byte(`{"schema_version":"achta.toolchain-manifest.v1","versions":[{"name":"rulefloor","env":"RULEFLOOR_VERSION","value":"v0.9.0"},{"name":"gograph","env":"GOGRAPH_VERSION","value":"v0.8.0"}],"digests":[]}`)
	workflow := []byte("env:\n  RULEFLOOR_VERSION: v0.8.0\njobs: {}\n")
	result, err := Check(repo, manifest, workflow)
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "fail" || result.Pins[0].Status != "missing_ci_pin" || result.Pins[1].Status != "mismatch" {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestCheckRejectsEmptyManifest(t *testing.T) {
	manifest := []byte(`{"schema_version":"achta.toolchain-manifest.v1","versions":[],"digests":[]}`)
	if _, err := Check(t.TempDir(), manifest, []byte("env:\n  PIN: value\n")); err == nil {
		t.Fatal("empty manifest accepted")
	}
}

package cli

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"
)

const testVersion = "v0.5.3"

func TestVersionJSONIsSingleDocument(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := Run([]string{"version", "--json"}, &stdout, &stderr, testVersion); code != 0 {
		t.Fatalf("code = %d, stderr = %q", code, stderr.String())
	}
	output := stdout.String()
	var doc versionDocument
	decoder := json.NewDecoder(strings.NewReader(output))
	if err := decoder.Decode(&doc); err != nil {
		t.Fatal(err)
	}
	if decoder.More() {
		t.Fatal("more than one JSON document")
	}
	if doc.SchemaVersion != versionSchema || doc.Version != testVersion {
		t.Fatalf("unexpected document: %+v", doc)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q", stderr.String())
	}
	assertGolden(t, "version.json", output)
}

// RULE: MACHINE-CONTRACTS-1
func TestCapabilitiesNeedNoWorkspace(t *testing.T) {
	original, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	foreign := t.TempDir()
	if err := os.WriteFile(foreign+"/workspace-noise", []byte("not a workspace\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(foreign); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(original) })

	var stdout, stderr bytes.Buffer
	if code := Run([]string{"--workspace", "/does/not/exist", "capabilities", "--json"}, &stdout, &stderr, testVersion); code != 0 {
		t.Fatalf("code = %d, stderr = %q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), capabilitiesSchema) {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if err := os.Chdir(original); err != nil {
		t.Fatal(err)
	}
	assertGolden(t, "capabilities.json", stdout.String())
	assertStableMachineContracts(t)
}

func TestVersionAgreementStates(t *testing.T) {
	for _, tc := range []struct {
		name      string
		release   string
		toolchain string
		want      string
	}{
		{name: "agreement", release: "v0.2.0", toolchain: "v0.2.0", want: "pass"},
		{name: "disagreement", release: "v0.2.0", toolchain: "v0.1.0", want: "fail"},
		{name: "unavailable", release: "v0.2.0", toolchain: "unknown", want: "cannot_evaluate"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := versionDocumentFor(tc.release, tc.toolchain); got.VersionAgreement != tc.want {
				t.Fatalf("agreement = %q, want %q", got.VersionAgreement, tc.want)
			}
		})
	}
}

func TestHelpAfterCommandNeedsNoWorkspace(t *testing.T) {
	for _, args := range [][]string{{"wiki", "--help"}, {"decision", "add", "--help"}, {"witness", "summarize", "-h"}} {
		var stdout, stderr bytes.Buffer
		if code := Run(args, &stdout, &stderr, "v0.2.0"); code != 0 {
			t.Fatalf("Run(%q) code=%d stderr=%q", args, code, stderr.String())
		}
		if stdout.String() != helpText || stderr.Len() != 0 {
			t.Fatalf("Run(%q) stdout=%q stderr=%q", args, stdout.String(), stderr.String())
		}
	}
	var stdout, stderr bytes.Buffer
	if code := Run([]string{"decision", "add", "--title", "--help"}, &stdout, &stderr, "v0.2.0"); code != 2 {
		t.Fatalf("help used as a flag value was intercepted: code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
}

func assertGolden(t *testing.T, name, got string) {
	t.Helper()
	want, err := os.ReadFile("../../testdata/machine/" + name)
	if err != nil {
		t.Fatal(err)
	}
	if got != string(want) {
		t.Fatalf("%s mismatch\nwant: %s\ngot:  %s", name, want, got)
	}
}

func TestUnknownCommandJSONUsesExitTwo(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := Run([]string{"--json", "unknown"}, &stdout, &stderr, "dev"); code != 2 {
		t.Fatalf("code = %d", code)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q", stderr.String())
	}
	if !strings.Contains(stdout.String(), "cannot_evaluate") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestTimingAppendsHumanElapsedLine(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := Run([]string{"--timing", "version"}, &stdout, &stderr, testVersion); code != 0 {
		t.Fatalf("code = %d, stderr = %q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "achta "+testVersion) || !strings.HasSuffix(stdout.String(), "ms\n") {
		t.Fatalf("stdout = %q", stdout.String())
	}
	lines := strings.Split(strings.TrimSuffix(stdout.String(), "\n"), "\n")
	if len(lines) != 2 || !strings.HasPrefix(lines[1], "elapsed: ") {
		t.Fatalf("timing is not the final line: %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

// RULE: TIMING-JSON-1
func TestTimingAddsElapsedToSingleJSONDocument(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := Run([]string{"--quiet", "--timing", "version", "--json"}, &stdout, &stderr, testVersion); code != 0 {
		t.Fatalf("code = %d, stderr = %q", code, stderr.String())
	}
	decoder := json.NewDecoder(&stdout)
	var doc map[string]any
	if err := decoder.Decode(&doc); err != nil {
		t.Fatal(err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		t.Fatalf("expected one JSON document, got %v", err)
	}
	if elapsed, ok := doc["elapsed_ms"].(float64); !ok || elapsed < 0 {
		t.Fatalf("elapsed_ms = %#v", doc["elapsed_ms"])
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestQuietSuppressesOnlySuccessfulHumanDetail(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := Run([]string{"--quiet", "version"}, &stdout, &stderr, testVersion); code != 0 {
		t.Fatalf("quiet success code=%d stderr=%q", code, stderr.String())
	}
	if stdout.Len() != 0 || stderr.Len() != 0 {
		t.Fatalf("quiet success stdout=%q stderr=%q", stdout.String(), stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := Run([]string{"--quiet", "--timing", "version"}, &stdout, &stderr, testVersion); code != 0 {
		t.Fatalf("quiet timed success code=%d stderr=%q", code, stderr.String())
	}
	if !strings.HasPrefix(stdout.String(), "elapsed: ") || !strings.HasSuffix(stdout.String(), "ms\n") || stderr.Len() != 0 {
		t.Fatalf("quiet timed stdout=%q stderr=%q", stdout.String(), stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := Run([]string{"--quiet", "unknown"}, &stdout, &stderr, testVersion); code != 2 {
		t.Fatalf("quiet failure code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if stderr.Len() == 0 {
		t.Fatal("quiet suppressed failure diagnostics")
	}
}

func TestTimingAddsElapsedToJSONError(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := Run([]string{"--timing", "--json", "unknown"}, &stdout, &stderr, "v0.1.0"); code != 2 {
		t.Fatalf("code = %d, stderr = %q", code, stderr.String())
	}
	var doc map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &doc); err != nil {
		t.Fatal(err)
	}
	if doc["status"] != "cannot_evaluate" {
		t.Fatalf("status = %#v", doc["status"])
	}
	if _, ok := doc["elapsed_ms"].(float64); !ok {
		t.Fatalf("elapsed_ms = %#v", doc["elapsed_ms"])
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestTimingIsFinalHumanErrorLine(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := Run([]string{"--timing", "unknown"}, &stdout, &stderr, "v0.1.0"); code != 2 {
		t.Fatalf("code = %d", code)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q", stdout.String())
	}
	lines := strings.Split(strings.TrimSuffix(stderr.String(), "\n"), "\n")
	if len(lines) != 2 || !strings.HasPrefix(lines[1], "elapsed: ") || !strings.HasSuffix(lines[1], "ms") {
		t.Fatalf("timing is not the final error line: %q", stderr.String())
	}
}

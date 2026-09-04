package cli

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"
)

func TestVersionJSONIsSingleDocument(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := Run([]string{"version", "--json"}, &stdout, &stderr, "v0.1.0"); code != 0 {
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
	if doc.SchemaVersion != versionSchema || doc.Version != "v0.1.0" {
		t.Fatalf("unexpected document: %+v", doc)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q", stderr.String())
	}
	assertGolden(t, "version.json", output)
}

func TestCapabilitiesNeedNoWorkspace(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := Run([]string{"--workspace", "/does/not/exist", "capabilities", "--json"}, &stdout, &stderr, "v0.1.0"); code != 0 {
		t.Fatalf("code = %d, stderr = %q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), capabilitiesSchema) {
		t.Fatalf("stdout = %q", stdout.String())
	}
	assertGolden(t, "capabilities.json", stdout.String())
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
	if code := Run([]string{"--timing", "version"}, &stdout, &stderr, "v0.1.0"); code != 0 {
		t.Fatalf("code = %d, stderr = %q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "achta v0.1.0") || !strings.HasSuffix(stdout.String(), "ms\n") {
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
	if code := Run([]string{"--timing", "version", "--json"}, &stdout, &stderr, "v0.1.0"); code != 0 {
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

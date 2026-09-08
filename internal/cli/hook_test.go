package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestHookCDPreToolUseContract(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		payload    string
		wantCode   int
		wantStderr string
	}{
		{
			name:       "deny names statement and fix",
			payload:    `{"tool_input":{"command":"make verify"}}`,
			wantCode:   2,
			wantStderr: "statement 1 (make)",
		},
		{
			name:     "leading absolute cd allows",
			payload:  `{"tool_input":{"command":"cd /repo && make verify"}}`,
			wantCode: 0,
		},
		{
			name:       "unparseable warns open",
			payload:    `not json`,
			wantCode:   0,
			wantStderr: "WARNING",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			var stdout, stderr bytes.Buffer
			code := runWithInput([]string{"--workspace", "/does/not/exist", "hook", "cd"}, strings.NewReader(test.payload), &stdout, &stderr, testVersion)
			if code != test.wantCode {
				t.Fatalf("code = %d, want %d; stderr = %q", code, test.wantCode, stderr.String())
			}
			if stdout.Len() != 0 {
				t.Fatalf("stdout = %q, want empty", stdout.String())
			}
			if !strings.Contains(stderr.String(), test.wantStderr) {
				t.Fatalf("stderr = %q, want substring %q", stderr.String(), test.wantStderr)
			}
			if test.wantCode == 2 && !strings.Contains(stderr.String(), "fix:") {
				t.Fatalf("stderr = %q, want fix", stderr.String())
			}
		})
	}
}

func TestHookCDTracksAdvertisedAchtaWriters(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	if code := runCapabilities([]string{"--json"}, &stdout, &stderr, globalOptions{}, testVersion); code != 0 {
		t.Fatalf("capabilities code = %d, stderr = %q", code, stderr.String())
	}
	var document struct {
		Commands []capabilityCommand `json:"commands"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &document); err != nil {
		t.Fatal(err)
	}
	for _, command := range document.Commands {
		if !command.Writes {
			continue
		}
		t.Run(command.Name, func(t *testing.T) {
			payload, err := json.Marshal(map[string]any{"tool_input": map[string]any{"command": "achta " + command.Name}})
			if err != nil {
				t.Fatal(err)
			}
			var out, diagnostics bytes.Buffer
			if code := runWithInput([]string{"hook", "cd"}, bytes.NewReader(payload), &out, &diagnostics, testVersion); code != 2 {
				t.Fatalf("hook code = %d for advertised writer %q; stderr = %q", code, command.Name, diagnostics.String())
			}
		})
	}
}

package cli

import (
	"bytes"
	"strings"
	"testing"
)

// The hook CLI reports the offending variable name, allows the ACH-02 forms,
// and never echoes other command arguments (HOOK-CD-RETRY-1).
func TestHookCDVariableAndFalsePositiveContract(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		command    string
		wantCode   int
		wantStderr string
	}{
		{"deny names statement ordinal", "pwd; mkdir -p $DIR/x", 2, "statement 2 (mkdir)"},
		{"deny names the variable", "pwd; mkdir -p $DIR/x", 2, "$DIR is a shell variable"},
		{"deny suggests the literal form", "printf x > $OUT", 2, "write the literal absolute path"},
		{"read-only git listing allows", "git tag --contains abc123", 0, ""},
		{"scratch redirect under cd allows", "cd /repo && go test ./... > /tmp/out", 0, ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			payload := `{"tool_input":{"command":"` + test.command + `"}}`
			var stdout, stderr bytes.Buffer
			code := runWithInput([]string{"hook", "cd"}, strings.NewReader(payload), &stdout, &stderr, testVersion)
			if code != test.wantCode || stdout.Len() != 0 || !strings.Contains(stderr.String(), test.wantStderr) {
				t.Fatalf("code=%d stdout=%q stderr=%q, want code %d containing %q", code, stdout.String(), stderr.String(), test.wantCode, test.wantStderr)
			}
		})
	}
}

func TestHookCDDenyDoesNotEchoArguments(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	payload := `{"tool_input":{"command":"printf SECRET1 > $OUT"}}`
	if code := runWithInput([]string{"hook", "cd"}, strings.NewReader(payload), &stdout, &stderr, testVersion); code != 2 {
		t.Fatalf("code = %d, want 2", code)
	}
	if strings.Contains(stderr.String(), "SECRET1") {
		t.Fatalf("stderr echoes a command argument: %q", stderr.String())
	}
}

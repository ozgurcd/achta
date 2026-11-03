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
		{"deny explains a relative redirect", "printf x > out.txt", 2, "why: the > target is relative and the command has no leading absolute cd"},
		{"deny explains a target outside the cd", "cd /repo && printf x > /other/out.txt", 2, "why: the > target is outside the leading cd and is not a scratch directory"},
		{"deny explains a missing selector", "git commit -m x", 2, "why: it has no absolute -C selector and the command has no leading absolute cd"},
		{"deny explains a relative selector", "git -C repo commit -m x", 2, "why: the -C selector is not an absolute path inside the leading cd"},
		{"deny explains a variable target", "pwd; mkdir -p $DIR/x", 2, "why: a mkdir target uses a shell variable or glob"},
		{"macOS temp redirect under cd allows", "cd /repo && go list ./... > /var/folders/ab/T/out", 0, ""},
		{"go install into the tmp root allows", "GOBIN=/tmp go install ./cmd/tool", 0, ""},
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

// The explanation line classifies the write without echoing its target.
func TestHookCDDenyExplanationDoesNotEchoTargets(t *testing.T) {
	t.Parallel()

	for _, command := range []string{
		"cd /repo && printf x > /SECRET2/out.txt",
		"cd /repo && cp go.mod /SECRET2/go.mod",
		"git -C SECRET2 commit -m x",
		"printf x > SECRET2.txt",
	} {
		var stdout, stderr bytes.Buffer
		payload := `{"tool_input":{"command":"` + command + `"}}`
		if code := runWithInput([]string{"hook", "cd"}, strings.NewReader(payload), &stdout, &stderr, testVersion); code != 2 {
			t.Fatalf("%q: code = %d, want 2", command, code)
		}
		if !strings.Contains(stderr.String(), "why: ") || strings.Contains(stderr.String(), "SECRET2") {
			t.Fatalf("%q: stderr must explain without echoing the target: %q", command, stderr.String())
		}
	}
}

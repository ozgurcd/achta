package rulefloorclient

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestClientCapabilitiesAndDifferentDiff(t *testing.T) {
	script := fakeExecutable(t, `#!/bin/sh
case "$1" in
  capabilities)
    printf '%s\n' '{"schema_version":"rulefloor.capabilities.v1","machine_interfaces":["rulefloor.ledger-diff.v1"],"ledger_features":["ledger-diff-sentence-sha256"]}'
    exit 0
    ;;
  ledger-diff)
    printf '%s\n' '{"schema_version":"rulefloor.ledger-diff.v1","status":"different","base_commit":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","headers_changed":false,"header_changes":[],"rules":[{"rule_id":"A-1","changes":["sentence_changed"],"after_sentence_sha256":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"}],"total_rule_changes":1,"truncated":false}'
    exit 1
    ;;
esac
exit 2
`)
	client := Client{Path: script}
	if err := client.CheckCompatibility(); err != nil {
		t.Fatal(err)
	}
	diff, raw, err := client.LedgerDiff(strings.Repeat("a", 40), ".")
	if err != nil {
		t.Fatal(err)
	}
	if diff.Status != "different" || len(diff.Rules) != 1 || len(raw) == 0 {
		t.Fatalf("diff = %+v", diff)
	}
}

func TestClientRefusesWrongSchemaNoisyAndExitMismatch(t *testing.T) {
	for name, tc := range map[string]struct {
		body      string
		operation string
	}{
		"wrong schema": {`#!/bin/sh
printf '%s\n' '{"schema_version":"wrong","machine_interfaces":[],"ledger_features":[]}'
`, "capabilities"},
		"noisy stdout": {`#!/bin/sh
printf '%s\n' '{"schema_version":"rulefloor.capabilities.v1","machine_interfaces":["rulefloor.ledger-diff.v1"],"ledger_features":["ledger-diff-sentence-sha256"]}' 'noise'
`, "capabilities"},
		"status mismatch": {`#!/bin/sh
printf '%s\n' '{"schema_version":"rulefloor.ledger-diff.v1","status":"different","base_commit":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","headers_changed":false,"header_changes":[],"rules":[],"total_rule_changes":0,"truncated":false}'
exit 0
`, "diff"},
	} {
		t.Run(name, func(t *testing.T) {
			client := Client{Path: fakeExecutable(t, tc.body)}
			var err error
			if tc.operation == "capabilities" {
				err = client.CheckCompatibility()
			} else {
				_, _, err = client.LedgerDiff(strings.Repeat("a", 40), ".")
			}
			if err == nil {
				t.Fatal("invalid fake output accepted")
			}
		})
	}
}

func TestClientRefusesTimeoutAndOversize(t *testing.T) {
	timeout := Client{Path: fakeExecutable(t, "#!/bin/sh\nsleep 2\n"), Timeout: 20 * time.Millisecond}
	if err := timeout.CheckCompatibility(); err == nil || !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("timeout err = %v", err)
	}
	oversize := Client{Path: fakeExecutable(t, "#!/bin/sh\nhead -c 8400000 /dev/zero\n")}
	if err := oversize.CheckCompatibility(); err == nil || !strings.Contains(err.Error(), "exceeded") {
		t.Fatalf("oversize err = %v", err)
	}
}

func fakeExecutable(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "fake-rulefloor")
	if err := os.WriteFile(path, []byte(body), 0o700); err != nil {
		t.Fatal(err)
	}
	return path
}

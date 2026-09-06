package cli

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCountCheckRetirementGapFlags(t *testing.T) {
	root, _, _ := testWorkspaceRepo(t, "sample")
	writeTestFile(t, filepath.Join(root, "wiki", "entry.md"), "## [2026-08-01] old\n\nFive fences fixed.\n\n## [2026-09-06] current\n\nwe fixed Five fences\ncontrols that fired: 4\n")
	args := []string{
		"--json", "--workspace", root, "count", "check", "--file", "wiki/entry.md",
		"--claim-pattern", `(?P<count>Five)[[:space:]]+fences[[:space:]]+fixed`,
		"--claim-pattern", `(?P<count>Four)[[:space:]]+sites[[:space:]]+fixed`,
		"--citation-pattern", `[[:alnum:]_./-]+[.]md:[0-9]+`,
		"--claim-section-pattern", `^##[[:space:]]+\[[0-9]{4}-[0-9]{2}-[0-9]{2}\]`,
		"--proof-claim-pattern", `fixed[[:space:]]+(?P<count>Five)[[:space:]]+fences`,
		"--proof-pattern", `controls[[:space:]]+that[[:space:]]+fired:[[:space:]]*(?P<count>[0-9]+)`,
		"--proof-within-lines", "12",
		"--exempt-pattern", `<!--[[:space:]]*count-claim-exempt:[[:space:]]*[^[:space:]>][^>]*-->`,
		"--count-alias", "Five=5", "--count-alias", "Four=4",
	}
	var stdout, stderr bytes.Buffer
	if code := Run(args, &stdout, &stderr, "v0.5.3"); code != 1 || !strings.Contains(stdout.String(), `"rule":"claim-proof-count"`) || !strings.Contains(stdout.String(), `"proof_claims":1`) {
		t.Fatalf("proof code/output = %d/%s stderr=%s", code, stdout.String(), stderr.String())
	}

	writeTestFile(t, filepath.Join(root, "wiki", "entry.md"), "## [2026-08-01] old\n\nFive fences fixed.\n\n## [2026-09-06] current\n\n<!-- count-claim-exempt: quotes a past mismatch -->\nwe fixed Five fences\ncontrols that fired: 4\n")
	stdout.Reset()
	stderr.Reset()
	if code := Run(args, &stdout, &stderr, "v0.5.3"); code != 0 || !strings.Contains(stdout.String(), `"status":"pass"`) || !strings.Contains(stdout.String(), `"claims":0`) {
		t.Fatalf("scope/exemption code/output = %d/%s stderr=%s", code, stdout.String(), stderr.String())
	}
}

func TestDeclaredRouteCheckSupportsWorkflowDirectorySet(t *testing.T) {
	root, _, _ := testWorkspaceRepo(t, "sample")
	dir := filepath.Join(root, ".github", "workflows")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, filepath.Join(dir, "ci.yml"), "env:\n  RULEFLOOR_VERSION: v1\njobs:\n  verify:\n    run: derived-route\n  integration:\n    run: derived-route\n")
	writeTestFile(t, filepath.Join(dir, "publish.yml"), "jobs:\n  publish:\n    run: echo publish\n")
	args := []string{
		"--json", "--workspace", root, "declared-route", "check",
		"--dir", ".github/workflows", "--route-cardinality", "per-file-any",
		"--route-pattern", "derived-route", "--ban-pattern", "banned-install",
		"--required-route-pattern", "derived-route", "--required-key", "RULEFLOOR_VERSION", "--required-scope", "/env",
	}
	var stdout, stderr bytes.Buffer
	if code := Run(args, &stdout, &stderr, "v0.5.3"); code != 0 || !strings.Contains(stdout.String(), `"route_cardinality":"per-file-any"`) || !strings.Contains(stdout.String(), `"file":".github/workflows/publish.yml"`) {
		t.Fatalf("set code/output = %d/%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if strings.Index(stdout.String(), `"file":".github/workflows/ci.yml"`) > strings.Index(stdout.String(), `"file":".github/workflows/publish.yml"`) {
		t.Fatalf("directory order is not bytewise: %s", stdout.String())
	}
	writeTestFile(t, filepath.Join(dir, "publish.yml"), "jobs:\n  publish:\n    run: banned-install\n")
	stdout.Reset()
	stderr.Reset()
	if code := Run(args, &stdout, &stderr, "v0.5.3"); code != 1 || !strings.Contains(stdout.String(), `"rule":"banned-pattern"`) {
		t.Fatalf("ban code/output = %d/%s stderr=%s", code, stdout.String(), stderr.String())
	}

	duplicateArgs := append(append([]string{}, args...), "--file", ".github/workflows/ci.yml")
	stdout.Reset()
	stderr.Reset()
	if code := Run(duplicateArgs, &stdout, &stderr, "v0.5.3"); code != 2 || !strings.Contains(stdout.String(), `"status":"cannot_evaluate"`) {
		t.Fatalf("duplicate code/output = %d/%s stderr=%s", code, stdout.String(), stderr.String())
	}
}

func TestMirrorCheckSupportsExplicitDigestRecordName(t *testing.T) {
	root, _, _ := testWorkspaceRepo(t, "sample")
	if err := os.MkdirAll(filepath.Join(root, "wiki", "tools"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "wiki", "contracts"), 0o755); err != nil {
		t.Fatal(err)
	}
	master := []byte("master\n")
	writeTestFile(t, filepath.Join(root, "wiki", "tools", "rulefloor-install-gate.sh"), string(master))
	digest := fmt.Sprintf("%x  tools/rulefloor-install-gate.sh\n", sha256.Sum256(master))
	writeTestFile(t, filepath.Join(root, "wiki", "contracts", "rulefloor-install-gate.master.sha256"), digest)
	args := []string{
		"--json", "--workspace", root, "mirror", "check",
		"--master", "wiki/tools/rulefloor-install-gate.sh",
		"--digest", "wiki/contracts/rulefloor-install-gate.master.sha256",
		"--digest-name", "tools/rulefloor-install-gate.sh",
	}
	var stdout, stderr bytes.Buffer
	if code := Run(args, &stdout, &stderr, "v0.5.3"); code != 0 || !strings.Contains(stdout.String(), `"name":"tools/rulefloor-install-gate.sh"`) {
		t.Fatalf("digest name code/output = %d/%s stderr=%s", code, stdout.String(), stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	withoutDigest := []string{"--json", "--workspace", root, "mirror", "check", "--master", "wiki/tools/rulefloor-install-gate.sh", "--mirror", "wiki/tools/rulefloor-install-gate.sh", "--digest-name", "tools/rulefloor-install-gate.sh"}
	if code := Run(withoutDigest, &stdout, &stderr, "v0.5.3"); code != 2 || !strings.Contains(stdout.String(), `"status":"cannot_evaluate"`) {
		t.Fatalf("digest-name without digest code/output = %d/%s stderr=%s", code, stdout.String(), stderr.String())
	}
}

package releasenotes

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// RULE: RELEASE-PINNED-CHECK-1
func TestPinnedReleaseCheck(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	workflow, err := os.ReadFile(filepath.Join(root, ".github/workflows/release.yml"))
	if err != nil {
		t.Fatal(err)
	}
	pin := regexp.MustCompile(`GORELEASER_VERSION: '(v[0-9]+\.[0-9]+\.[0-9]+)'`).FindSubmatch(workflow)
	if len(pin) != 2 {
		t.Fatal("release workflow must name one exact GoReleaser version")
	}
	cmd := exec.Command("make", "--no-print-directory", "-n", "release-check")
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("release-check unavailable: %v: %s", err, out)
	}
	want := "go run github.com/goreleaser/goreleaser/v2@" + string(pin[1]) + " check"
	if strings.TrimSpace(string(out)) != want {
		t.Fatalf("release-check recipe=%q want=%q", out, want)
	}
	if !strings.Contains(string(workflow), "run: make release-check") {
		t.Fatal("release workflow must run the same release-check recipe")
	}
}

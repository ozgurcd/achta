package releasenotes

import (
	"os"
	"strings"
	"testing"
)

func TestGo1272ToolchainPins(t *testing.T) {
	for _, path := range []string{"../../go.mod"} {
		data, err := os.ReadFile(path)
		if err != nil || !strings.Contains(string(data), "1.27.2") {
			t.Fatalf("Go 1.27.2 missing in %s", path)
		}
	}
	for _, path := range []string{"../../.github/workflows/release.yml", "../../.github/workflows/verify.yml"} {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		want := 1
		if strings.HasSuffix(path, "release.yml") {
			want = 2
		}
		const action = "uses: ozgurcd/lictor/.github/actions/go-toolchain@d35a8ad07dc59bd76153f0ed7d3b1ade22c6cf04"
		if got := strings.Count(string(data), action); got != want {
			t.Errorf("%s has %d shared toolchain steps, want %d", path, got, want)
		}
		for _, duplicate := range []string{"actions/setup-go@", "GO_VERSION:", "STATICCHECK_TAG:", "go install honnef.co/go/tools"} {
			if strings.Contains(string(data), duplicate) {
				t.Errorf("duplicate toolchain setup %q in %s", duplicate, path)
			}
		}
	}
	data, err := os.ReadFile("../../Makefile")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "go install honnef.co/go/tools") {
		t.Fatal("install-tools replaces the export-data-compatible binary")
	}
}

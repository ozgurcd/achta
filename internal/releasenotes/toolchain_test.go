package releasenotes

import (
	"os"
	"strings"
	"testing"
)

func TestGo1272ToolchainPins(t *testing.T) {
	for _, path := range []string{"../../go.mod", "../../.github/workflows/release.yml", "../../.github/workflows/verify.yml"} {
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
		for _, pin := range []string{"2026.2.1", "8d807cd909f4481d6777f7707e5ae75dcc399e14d68ff14a3c814731826e0dfc", "19f123d3f405f779e82a739d3d6e7e3b289659b496ac82bd699586465b06ecc4", "52ce80f83d597020938bb7c463070f3c133802995029faeb632e3fc385fb4d8a", "sha256sum -c -", "./cmd/staticcheck"} {
			if !strings.Contains(string(data), pin) {
				t.Fatalf("missing Staticcheck recipe pin in %s", path)
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

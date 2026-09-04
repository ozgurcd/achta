package amendments

import (
	"os"
	"testing"
)

func TestManifestFixture(t *testing.T) {
	data, err := os.ReadFile("../../testdata/amendments/manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(manifest.Changes) != 1 || manifest.Changes[0].RuleID != "EXAMPLE-1" {
		t.Fatalf("unexpected fixture: %+v", manifest)
	}
}

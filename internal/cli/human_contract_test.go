package cli

import (
	"bytes"
	"os"
	"testing"

	"github.com/ozgurcd/achta/internal/witness"
)

func TestHumanContracts(t *testing.T) {
	for _, fixture := range []struct {
		name string
		args []string
	}{
		{name: "version.txt", args: []string{"version"}},
		{name: "capabilities.txt", args: []string{"capabilities"}},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			if code := Run(fixture.args, &stdout, &stderr, "v0.2.0"); code != 0 {
				t.Fatalf("code=%d stderr=%q", code, stderr.String())
			}
			want, err := os.ReadFile("../../testdata/human/" + fixture.name)
			if err != nil {
				t.Fatal(err)
			}
			if stdout.String() != string(want) {
				t.Fatalf("%s mismatch\nwant: %s\ngot:  %s", fixture.name, want, stdout.String())
			}
		})
	}
}

func TestPhaseOneHumanContracts(t *testing.T) {
	t.Run("wiki-pin.txt", func(t *testing.T) {
		var output bytes.Buffer
		renderWikiPinHuman(&output, wikiPinDocument{
			SchemaVersion: "achta.wiki-pin.v1",
			Status:        "updated",
			Repository:    "sample",
			SHA:           "0123456789abcdef0123456789abcdef01234567",
			Verified:      "2026-09-04",
			Changed:       true,
		})
		assertHumanGolden(t, "wiki-pin.txt", output.String())
	})
	t.Run("witness-summary.txt", func(t *testing.T) {
		var output bytes.Buffer
		renderWitnessSummaryHuman(&output, "GATE-RUN.txt", witness.Summary{
			Status:         "green",
			Completeness:   "complete",
			Freshness:      "current",
			PlannedTargets: 2,
			PassedTargets:  2,
			SlowTargets: []witness.SlowTarget{
				{Name: "test", ElapsedMS: 12},
				{Name: "build", ElapsedMS: 5},
			},
		})
		assertHumanGolden(t, "witness-summary.txt", output.String())
	})
}

func assertHumanGolden(t *testing.T, name, got string) {
	t.Helper()
	want, err := os.ReadFile("../../testdata/human/" + name)
	if err != nil {
		t.Fatal(err)
	}
	if got != string(want) {
		t.Fatalf("%s mismatch\nwant: %s\ngot:  %s", name, want, got)
	}
}

package witness

import (
	"os"
	"testing"
)

func TestWitnessFixtures(t *testing.T) {
	for _, fixture := range []struct {
		name       string
		result     string
		shouldFail bool
	}{
		{name: "green.txt", result: "green"},
		{name: "red.txt", result: "red"},
		{name: "incomplete.txt"},
		{name: "malformed.txt", shouldFail: true},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			data, err := os.ReadFile("../../testdata/witness/" + fixture.name)
			if err != nil {
				t.Fatal(err)
			}
			record, err := Parse(data)
			if fixture.shouldFail {
				if err == nil {
					t.Fatalf("malformed fixture parsed: %+v", record)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if record.Result != fixture.result {
				t.Fatalf("result = %q, want %q", record.Result, fixture.result)
			}
		})
	}
}

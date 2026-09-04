package witness

import (
	"strings"
	"testing"
)

func TestParseGreenAndTiming(t *testing.T) {
	record, err := Parse([]byte(`schema: gate-run.v1
gate: sample
repo-head: aaaaaaa
started: 2026-09-04T10:00:00Z
plan: beta alpha
elapsed: beta 1s
target: beta exit=0
elapsed: alpha 2s
target: alpha exit=0
finished: 2026-09-04T10:00:04Z
tree: commit=aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
result: green
`))
	if err != nil {
		t.Fatal(err)
	}
	if len(record.Targets) != 2 || record.ElapsedMS["alpha"] != 2000 {
		t.Fatalf("record = %+v", record)
	}
}

func TestParseRejectsDuplicateAndImpossibleTime(t *testing.T) {
	base := `schema: gate-run.v1
started: 2026-09-04T10:00:04Z
plan: one
target: one exit=0
finished: 2026-09-04T10:00:00Z
`
	if _, err := Parse([]byte(base)); err == nil || !strings.Contains(err.Error(), "precedes") {
		t.Fatalf("err = %v", err)
	}
	if _, err := Parse([]byte(strings.Replace(base, "plan: one", "plan: one one", 1))); err == nil {
		t.Fatal("duplicate plan accepted")
	}
}

func TestParseIncompleteIsRepresentable(t *testing.T) {
	record, err := Parse([]byte("schema: gate-run.v1\nstarted: 2026-09-04T10:00:00Z\nplan: one two\ntarget: one exit=0\n"))
	if err != nil {
		t.Fatal(err)
	}
	if len(record.Plan) != 2 || len(record.Targets) != 1 {
		t.Fatalf("record = %+v", record)
	}
}

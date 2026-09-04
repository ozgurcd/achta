package witness

import (
	"strings"
	"testing"
	"time"
)

func TestParseSiblingPinsAndCIProvenance(t *testing.T) {
	data := []byte("schema: gate-run.v1\nrepo-head: 0123456789abcdef0123456789abcdef01234567\nstarted: 2026-09-04T10:00:00Z\nplan: verify\ntarget: verify exit=0\nfinished: 2026-09-04T10:00:01Z\nci-run: https://github.com/example/repo/actions/runs/42 attempt=2 sha=0123456789abcdef0123456789abcdef01234567\ntree: commit=0123456789abcdef0123456789abcdef01234567\nxrepo: sibling head=abcdef0123456789abcdef0123456789abcdef01 tree=sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\nresult: green\n")
	record, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	if record.CI == nil || record.CI.Attempt != 2 || len(record.Siblings) != 1 || record.Siblings[0].Name != "sibling" {
		t.Fatalf("metadata was not parsed: %+v", record)
	}
}

func TestParseRejectsDuplicateSiblingAndCI(t *testing.T) {
	base := "schema: gate-run.v1\nplan: verify\n"
	xrepo := "xrepo: sibling head=abcdef0 tree=sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\n"
	if _, err := Parse([]byte(base + xrepo + xrepo)); err == nil || !strings.Contains(err.Error(), "duplicate sibling") {
		t.Fatalf("duplicate sibling error = %v", err)
	}
	ci := "ci-run: https://github.com/example/repo/actions/runs/42 attempt=1 sha=0123456789abcdef0123456789abcdef01234567\n"
	if _, err := Parse([]byte(base + ci + ci)); err == nil || !strings.Contains(err.Error(), "duplicate ci-run") {
		t.Fatalf("duplicate CI error = %v", err)
	}
}

func TestFinalizeWithMetadataWritesDeterministicPins(t *testing.T) {
	started := time.Date(2026, 9, 4, 10, 0, 0, 0, time.UTC)
	data, err := Init("gate", "0123456789abcdef0123456789abcdef01234567", []string{"verify"}, started)
	if err != nil {
		t.Fatal(err)
	}
	data, err = Step(data, "verify", 0, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	data, err = FinalizeWithMetadata(data, "commit", "0123456789abcdef0123456789abcdef01234567", started.Add(time.Second), FinalizeOptions{
		Siblings: []SiblingPin{
			{Name: "zeta", Head: "abcdef0123456789abcdef0123456789abcdef01", TreeValue: strings.Repeat("b", 64)},
			{Name: "alpha", Head: "abcdef0123456789abcdef0123456789abcdef01", TreeValue: strings.Repeat("a", 64)},
		},
		CI: &CIProvenance{RunURL: "https://github.com/example/repo/actions/runs/42", Attempt: 1, SHA: "0123456789abcdef0123456789abcdef01234567"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Index(string(data), "xrepo: alpha") > strings.Index(string(data), "xrepo: zeta") {
		t.Fatal("sibling pins are not sorted")
	}
	if _, err := Parse(data); err != nil {
		t.Fatal(err)
	}
}

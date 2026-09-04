package amendments

import (
	"strings"
	"testing"
)

func testManifest() Manifest {
	return Manifest{SchemaVersion: Schema, BaseCommit: strings.Repeat("a", 40), Changes: []Change{}}
}

func TestDeclareValidatesAndSorts(t *testing.T) {
	manifest := testManifest()
	var err error
	manifest, err = Declare(manifest, Change{RuleID: "Z-1", ChangeClass: "rule_added", Reason: "Adds Z."})
	if err != nil {
		t.Fatal(err)
	}
	manifest, err = Declare(manifest, Change{RuleID: "A-1", ChangeClass: "sentence_changed", AfterSentenceSHA256: strings.Repeat("b", 64), Reason: "Clarifies A."})
	if err != nil {
		t.Fatal(err)
	}
	if manifest.Changes[0].RuleID != "A-1" {
		t.Fatalf("changes not sorted: %+v", manifest.Changes)
	}
	if _, err := Declare(manifest, manifest.Changes[0]); err == nil {
		t.Fatal("duplicate declaration accepted")
	}
}

func TestSentenceDigestContract(t *testing.T) {
	manifest := testManifest()
	if _, err := Declare(manifest, Change{RuleID: "A-1", ChangeClass: "sentence_changed", Reason: "No digest."}); err == nil {
		t.Fatal("sentence change without digest accepted")
	}
	if _, err := Declare(manifest, Change{RuleID: "A-1", ChangeClass: "proof_changed", AfterSentenceSHA256: strings.Repeat("b", 64), Reason: "Wrong digest."}); err == nil {
		t.Fatal("digest on non-sentence change accepted")
	}
}

func TestEncodeRoundTrip(t *testing.T) {
	data, err := Encode(testManifest())
	if err != nil {
		t.Fatal(err)
	}
	if data[len(data)-1] != '\n' {
		t.Fatal("missing final newline")
	}
	if _, err := Parse(data); err != nil {
		t.Fatal(err)
	}
}

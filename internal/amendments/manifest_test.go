package amendments

import (
	"errors"
	"reflect"
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
	if _, err := Declare(manifest, manifest.Changes[0]); !errors.Is(err, ErrDuplicateDeclaration) {
		t.Fatalf("duplicate declaration err = %v", err)
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

func TestEveryChangeClassAndRebasePreservation(t *testing.T) {
	classes := []string{"rule_added", "rule_removed", "sentence_changed", "binding_changed", "proof_changed", "covered_symbols_changed", "test_fingerprint_changed"}
	manifest := testManifest()
	for i, class := range classes {
		change := Change{RuleID: "A-" + string(rune('1'+i)), ChangeClass: class, Reason: "Measured change."}
		if class == "sentence_changed" {
			change.AfterSentenceSHA256 = strings.Repeat("b", 64)
		}
		var err error
		manifest, err = Declare(manifest, change)
		if err != nil {
			t.Fatalf("class %s: %v", class, err)
		}
	}
	if len(manifest.Changes) != len(classes) {
		t.Fatalf("changes = %d, want %d", len(manifest.Changes), len(classes))
	}
	if _, err := Declare(testManifest(), Change{RuleID: "A-1", ChangeClass: "unknown", Reason: "Invalid class."}); err == nil {
		t.Fatal("unknown change class accepted")
	}

	before := append([]Change(nil), manifest.Changes...)
	rebased, err := Rebase(manifest, strings.Repeat("c", 40))
	if err != nil {
		t.Fatal(err)
	}
	if rebased.BaseCommit != strings.Repeat("c", 40) || !reflect.DeepEqual(rebased.Changes, before) {
		t.Fatalf("rebase changed declarations: before=%+v after=%+v", before, rebased.Changes)
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

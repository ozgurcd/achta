package reachability

import "testing"

// RULE: REACHABILITY-FAIL-CLOSED-1
func TestClassifyRecordsExactExcludedPaths(t *testing.T) {
	result, err := Classify(
		[]string{"docs/guide.md", "README.md"},
		[]Exclusion{{Pattern: "docs/**", Why: "documentation cannot reach the gate"}, {Pattern: "README.md", Why: "documentation only"}},
	)
	if err != nil {
		t.Fatal(err)
	}
	if result.Decision != "SKIPPABLE" || len(result.Excluded) != 2 || result.Excluded[0].Path != "README.md" || result.Excluded[1].Path != "docs/guide.md" {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestClassifyFailsClosedOnUnmatchedPath(t *testing.T) {
	result, err := Classify([]string{"docs/guide.md", "internal/gate.go"}, []Exclusion{{Pattern: "docs/**", Why: "documentation only"}})
	if err != nil {
		t.Fatal(err)
	}
	if result.Decision != "REQUIRED" || len(result.Unknown) != 1 || result.Unknown[0] != "internal/gate.go" {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestClassifyRejectsCatchAllPatterns(t *testing.T) {
	for _, pattern := range []string{"*", "**", "**/*", "./**"} {
		if _, err := Classify([]string{"README.md"}, []Exclusion{{Pattern: pattern, Why: "too broad"}}); err == nil {
			t.Fatalf("catch-all pattern %q accepted", pattern)
		}
	}
}

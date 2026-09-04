package wiki

import (
	"strings"
	"testing"
)

func FuzzRenderPin(f *testing.F) {
	f.Add([]byte(pageFixture))
	sha := strings.Repeat("b", 40)
	f.Fuzz(func(t *testing.T, data []byte) {
		_, _ = RenderPin(data, "sample", sha, "2026-09-04", "main")
	})
}

func FuzzDerivedBlockDelimiters(f *testing.F) {
	f.Add([]byte("before\n<!-- BEGIN DERIVED: sample -->\nold\n<!-- END DERIVED -->\nafter\n"))
	f.Add([]byte("<!-- BEGIN DERIVED: sample -->\n<!-- BEGIN DERIVED: sample -->\n<!-- END DERIVED -->\n"))
	f.Fuzz(func(t *testing.T, data []byte) {
		repository, marked, err := derivedRepository(data)
		if err != nil || !marked {
			return
		}
		if repository == "" {
			t.Fatal("marked page has no repository")
		}
		if _, err := spliceDerived(data, repository, []byte("generated\n")); err != nil {
			t.Fatalf("validated delimiters failed to splice: %v", err)
		}
	})
}

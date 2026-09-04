package wiki

import (
	"os"
	"testing"
)

func TestRepositoryPageFixture(t *testing.T) {
	data, err := os.ReadFile("../../testdata/wiki/repository-page.md")
	if err != nil {
		t.Fatal(err)
	}
	fields, err := frontmatter(data)
	if err != nil {
		t.Fatal(err)
	}
	if fields["category"] != "repo" || fields["verified_against"] != "sample @ 0123456789abcdef0123456789abcdef01234567 (main)" {
		t.Fatalf("unexpected fixture fields: %v", fields)
	}
}

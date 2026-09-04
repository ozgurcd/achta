package slicecheck

import "testing"

func TestSecretPathClassification(t *testing.T) {
	if !secretPath([]string{"config/.env.local"}) {
		t.Fatal("secret-like path passed")
	}
	if secretPath([]string{"dev.env.example", "docs/key.pem.example"}) {
		t.Fatal("example path failed")
	}
}

func TestLogHeadings(t *testing.T) {
	got := logHeadings([]byte("# Log\r\n## [2026-01-01] one\r\nbody\r\n## unrelated\r\n## [2026-01-02] two\r\n"))
	if len(got) != 2 || got[0] != "## [2026-01-01] one" || got[1] != "## [2026-01-02] two" {
		t.Fatalf("headings = %#v", got)
	}
}

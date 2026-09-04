package releasenotes

import "testing"

func TestExtractOneVersion(t *testing.T) {
	data := []byte("# Notes\n\n## v0.2.0 — 2026-10-01\n\nSecond.\n\n## v0.1.0 — 2026-09-04\n\nFirst.\n")
	got, err := Extract(data, "v0.2.0")
	if err != nil {
		t.Fatal(err)
	}
	want := "## v0.2.0 — 2026-10-01\n\nSecond.\n"
	if string(got) != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

// RULE: RELEASE-NOTES-1
func TestExtractRefusesMissingDuplicateAndInvalid(t *testing.T) {
	for _, tc := range []struct {
		version string
		data    string
	}{
		{"0.1.0", "## v0.1.0 — 2026-09-04\nbody\n"},
		{"v0.2.0", "## v0.1.0 — 2026-09-04\nbody\n"},
		{"v0.1.0", "## v0.1.0 — 2026-09-04\nbody\n## v0.1.0 — 2026-09-05\nbody\n"},
		{"v0.1.0", "## v0.1.0 — 2026-02-30\nbody\n"},
		{"v0.1.0", "## v0.1.0 — 2026-09-04 trailing\nbody\n"},
		{"v01.0.0", "## v01.0.0 — 2026-09-04\nbody\n"},
	} {
		if _, err := Extract([]byte(tc.data), tc.version); err == nil {
			t.Fatalf("version %q data %q passed", tc.version, tc.data)
		}
	}
}

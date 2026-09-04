package wiki

import (
	"strings"
	"testing"
)

const pageFixture = `---
title: sample
category: repo
updated: 2026-09-01
verified: 2026-09-01
verified_against: sample @ aaaaaaa (main). Earlier prose stays.
---

# sample

<!-- BEGIN DERIVED: sample -->
| Derived fact | Value |
| --- | --- |
| Repo HEAD | aaaaaaa (main) |
<!-- END DERIVED -->
`

func TestRenderPinPreservesSuffixAndUnrelatedBytes(t *testing.T) {
	sha := strings.Repeat("b", 40)
	got, err := RenderPin([]byte(pageFixture), "sample", sha, "2026-09-04", "main")
	if err != nil {
		t.Fatal(err)
	}
	text := string(got.Data)
	if !strings.Contains(text, "verified_against: sample @ "+sha+" (main). Earlier prose stays.") {
		t.Fatalf("suffix not preserved:\n%s", text)
	}
	if !strings.Contains(text, "| Repo HEAD | bbbbbbb (main) |") {
		t.Fatalf("head row not updated:\n%s", text)
	}
}

func TestRenderPinRefusesMalformedStructure(t *testing.T) {
	sha := strings.Repeat("b", 40)
	for name, input := range map[string]string{
		"duplicate field": strings.Replace(pageFixture, "verified: 2026-09-01", "verified: 2026-09-01\nverified: 2026-09-02", 1),
		"missing marker":  strings.Replace(pageFixture, "<!-- END DERIVED -->", "", 1),
		"wrong repo":      strings.Replace(pageFixture, "title: sample", "title: other", 1),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := RenderPin([]byte(input), "sample", sha, "2026-09-04", "main"); err == nil {
				t.Fatal("accepted malformed page")
			}
		})
	}
}

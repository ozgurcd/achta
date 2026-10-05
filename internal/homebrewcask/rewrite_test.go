package homebrewcask

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

// RULE: HOMEBREW-PRIVATE-ASSET-1
func TestRewriteUsesPublicAssetURLs(t *testing.T) {
	cask := testCask()

	got, err := Rewrite([]byte(cask))
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range archiveNames {
		if count := strings.Count(string(got), browserURLPrefix+name); count != 1 {
			t.Errorf("public cask has %d URLs for %q, want 1", count, name)
		}
	}
	for _, forbidden := range []string{"Authorization", "Accept:", "api.github.com", "HOMEBREW_GITHUB_API_TOKEN"} {
		if strings.Contains(string(got), forbidden) {
			t.Errorf("public cask contains forbidden marker %q", forbidden)
		}
	}
	// Reprocessing an already prepared cask must leave the exact bytes intact.
	again, err := Rewrite(got)
	if err != nil || !bytes.Equal(got, again) {
		t.Fatal("public cask preparation is not idempotent")
	}
}

// RULE: HOMEBREW-STRUCTURED-POSTFLIGHT-1
func TestRewriteUsesStructuredPostflight(t *testing.T) {
	got, err := Rewrite([]byte(testCask()))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(got, legacyPostflight) {
		t.Fatal("rewritten cask retains deprecated postflight hook")
	}
	if bytes.Contains(got, []byte("  postflight do\n")) {
		t.Fatal("rewritten cask contains a legacy postflight stanza")
	}
	if count := bytes.Count(got, structuredPostflight); count != 1 {
		t.Fatalf("rewritten cask contains structured postflight_steps %d times, want 1", count)
	}
	for _, required := range [][]byte{
		[]byte("  postflight_steps do\n"),
		[]byte("run \"/usr/bin/xattr\""),
		[]byte("{{staged_path}}/achta"),
	} {
		if !bytes.Contains(got, required) {
			t.Fatalf("rewritten cask is missing %q", required)
		}
	}
}

func TestRewriteRefusesIncompleteOrUnsafeInputs(t *testing.T) {
	tests := []struct {
		name string
		cask string
	}{
		{name: "authorization header", cask: testCask() + "Authorization: Bearer"},
		{name: "accept header", cask: testCask() + "Accept: application/octet-stream"},
		{name: "API URL", cask: testCask() + "https://api.github.com/repos/ozgurcd/achta/releases/assets/101"},
		{name: "installer token lookup", cask: testCask() + "HOMEBREW_GITHUB_API_TOKEN"},
		{name: "headers option", cask: testCask() + "headers: []"},
		{name: "missing cask URL", cask: strings.Replace(testCask(), browserURLPrefix+archiveNames[0], "", 1)},
		{name: "duplicate cask URL", cask: testCask() + "url \"" + browserURLPrefix + archiveNames[0] + "\"\n"},
		{name: "extra cask URL", cask: testCask() + "url \"https://example.invalid/archive\"\n"},
		{name: "missing postflight", cask: strings.Replace(testCask(), string(legacyPostflight), "", 1)},
		{name: "duplicate postflight", cask: testCask() + string(legacyPostflight)},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := Rewrite([]byte(test.cask)); err == nil {
				t.Fatal("Rewrite succeeded for unsafe input")
			}
		})
	}
}

func testCask() string {
	var builder strings.Builder
	for _, name := range archiveNames {
		fmt.Fprintf(&builder, "url %q\n", browserURLPrefix+name)
	}
	builder.Write(legacyPostflight)
	builder.WriteByte('\n')
	return builder.String()
}

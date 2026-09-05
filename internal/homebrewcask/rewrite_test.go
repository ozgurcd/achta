package homebrewcask

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

// RULE: HOMEBREW-PRIVATE-ASSET-1
func TestRewriteUsesAuthenticatedAssetAPIURLs(t *testing.T) {
	cask := testCask()
	release := testRelease()

	got, err := Rewrite([]byte(cask), []byte(release))
	if err != nil {
		t.Fatal(err)
	}
	for i, name := range archiveNames {
		apiURL := fmt.Sprintf("%s%d", assetAPIURLPrefix, i+101)
		if !strings.Contains(string(got), apiURL) {
			t.Errorf("rewritten cask is missing %q", apiURL)
		}
		if strings.Contains(string(got), browserURLPrefix+name) {
			t.Errorf("rewritten cask retained browser URL for %q", name)
		}
	}
}

// RULE: HOMEBREW-STRUCTURED-POSTFLIGHT-1
func TestRewriteUsesStructuredPostflight(t *testing.T) {
	got, err := Rewrite([]byte(testCask()), []byte(testRelease()))
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
		name    string
		cask    string
		release string
	}{
		{name: "missing header", cask: strings.Replace(testCask(), "Accept: application/octet-stream", "", 1), release: testRelease()},
		{name: "missing asset", cask: testCask(), release: strings.Replace(testRelease(), assetJSON(archiveNames[0], 101)+",", "", 1)},
		{name: "duplicate asset", cask: testCask(), release: strings.Replace(testRelease(), "]}", ","+assetJSON(archiveNames[0], 999)+"]}", 1)},
		{name: "browser API value", cask: testCask(), release: strings.Replace(testRelease(), assetAPIURLPrefix+"101", browserURLPrefix+archiveNames[0], 1)},
		{name: "non-numeric asset ID", cask: testCask(), release: strings.Replace(testRelease(), assetAPIURLPrefix+"101", assetAPIURLPrefix+"not-an-id", 1)},
		{name: "missing cask URL", cask: strings.Replace(testCask(), browserURLPrefix+archiveNames[0], "", 1), release: testRelease()},
		{name: "missing postflight", cask: strings.Replace(testCask(), string(legacyPostflight), "", 1), release: testRelease()},
		{name: "duplicate postflight", cask: testCask() + string(legacyPostflight), release: testRelease()},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := Rewrite([]byte(test.cask), []byte(test.release)); err == nil {
				t.Fatal("Rewrite succeeded for unsafe input")
			}
		})
	}
}

func testCask() string {
	var builder strings.Builder
	for _, name := range archiveNames {
		builder.WriteString("Accept: application/octet-stream\n")
		builder.WriteString("Authorization: Bearer #{ENV.fetch(\"HOMEBREW_GITHUB_API_TOKEN\", \"\")}\n")
		fmt.Fprintf(&builder, "url %q\n", browserURLPrefix+name)
	}
	builder.Write(legacyPostflight)
	builder.WriteByte('\n')
	return builder.String()
}

func testRelease() string {
	assets := make([]string, 0, len(archiveNames))
	for i, name := range archiveNames {
		assets = append(assets, assetJSON(name, i+101))
	}
	return `{"assets":[` + strings.Join(assets, ",") + `]}`
}

func assetJSON(name string, id int) string {
	return fmt.Sprintf(`{"name":%q,"url":%q}`, name, fmt.Sprintf("%s%d", assetAPIURLPrefix, id))
}

package releasenotes

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// RULE: HOMEBREW-RELEASE-1
func TestReleaseUsesCurrentHomebrewCaskPublishing(t *testing.T) {
	repositoryRoot, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		path      string
		required  []string
		forbidden []string
	}{
		{
			path: ".goreleaser.yaml",
			required: []string{
				"homebrew_casks:",
				"skip_upload: true",
				"name: homebrew-tap",
				"HOMEBREW_TAP_GITHUB_TOKEN",
				"binaries:",
				"Authorization: Bearer #{ENV.fetch",
				"com.apple.quarantine",
			},
			forbidden: []string{"\nbrews:", "directory: Formula"},
		},
		{
			path: ".github/workflows/release.yml",
			required: []string{
				"- name: Verify Homebrew tap credential",
				".permissions.push",
				"dist/homebrew/Casks/achta.rb",
				"secrets.HOMEBREW_TAP_GITHUB_TOKEN",
				"SOURCE_GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}",
				"repos/ozgurcd/achta/releases/tags/$GITHUB_REF_NAME",
				"go run ./cmd/homebrew-cask",
				"https://api.github.com/repos/ozgurcd/achta/releases/assets/",
				"retains a private release browser URL",
				"gh auth setup-git",
				"gh repo clone ozgurcd/homebrew-tap",
				"refusing to downgrade Homebrew",
				"git -C \"$tap_dir\" diff --cached --quiet",
				"repos/ozgurcd/homebrew-tap/contents/Casks/achta.rb",
			},
			forbidden: []string{"dist/homebrew/Formula/achta.rb"},
		},
	}
	for _, test := range tests {
		data, err := os.ReadFile(filepath.Join(repositoryRoot, test.path))
		if err != nil {
			t.Fatal(err)
		}
		contents := string(data)
		for _, required := range test.required {
			if !strings.Contains(contents, required) {
				t.Errorf("%s is missing %q", test.path, required)
			}
		}
		for _, forbidden := range test.forbidden {
			if strings.Contains(contents, forbidden) {
				t.Errorf("%s contains obsolete Homebrew configuration %q", test.path, forbidden)
			}
		}
	}
	workflow, err := os.ReadFile(filepath.Join(repositoryRoot, ".github/workflows/release.yml"))
	if err != nil {
		t.Fatal(err)
	}
	githubRelease := strings.Index(string(workflow), "- name: Publish private GitHub release")
	credentialCheck := strings.Index(string(workflow), "- name: Verify Homebrew tap credential")
	homebrewRelease := strings.Index(string(workflow), "- name: Publish and verify Homebrew cask")
	if credentialCheck < 0 || githubRelease < 0 || homebrewRelease < 0 || credentialCheck >= githubRelease || githubRelease >= homebrewRelease {
		t.Fatal("tap credential verification, private GitHub release, and Homebrew publication are out of order")
	}
}

func TestVerificationInstallsAndBuildsGograph(t *testing.T) {
	repositoryRoot, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(repositoryRoot, "Makefile"))
	if err != nil {
		t.Fatal(err)
	}
	contents := string(data)
	if !strings.Contains(contents, "go install github.com/ozgurcd/gograph/cmd/gograph@v1.6.10") {
		t.Fatal("install-tools does not install the pinned Gograph required by exact Rulefloor reach")
	}
	buildGraph := strings.Index(contents, "gograph build . --precise")
	checkFloor := strings.Index(contents, "rulefloor check --repo . --run-profile unit --timings")
	if buildGraph < 0 || checkFloor < 0 || buildGraph >= checkFloor {
		t.Fatal("verify does not build a precise graph before Rulefloor exact-reach evaluation")
	}
}

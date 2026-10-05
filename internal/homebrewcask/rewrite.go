// Package homebrewcask checks public downloads and prepares Homebrew postflight steps.
package homebrewcask

import (
	"bytes"
	"fmt"
	"strings"
)

var archiveNames = [...]string{
	"achta_Darwin_arm64.tar.gz",
	"achta_Darwin_x86_64.tar.gz",
	"achta_Linux_arm64.tar.gz",
	"achta_Linux_x86_64.tar.gz",
}

const browserURLPrefix = "https://github.com/ozgurcd/achta/releases/download/v#{version}/"

var (
	legacyPostflight = []byte(`  postflight do
    if OS.mac?
      system_command "/usr/bin/xattr", args: ["-dr", "com.apple.quarantine", "#{staged_path}/achta"]
    end
  end`)
	structuredPostflight = []byte(`  postflight_steps do
    on_macos do
      run "/usr/bin/xattr", args: ["-dr", "com.apple.quarantine", "{{staged_path}}/achta"]
    end
  end`)
)

// Rewrite refuses authenticated or unexpected downloads, keeps the public URLs,
// and converts GoReleaser's legacy postflight block to Homebrew's current form.
func Rewrite(cask []byte) ([]byte, error) {
	for _, forbidden := range []string{"authorization", "accept:", "api.github.com", "homebrew_github_api_token", "headers:"} {
		if bytes.Contains(bytes.ToLower(cask), []byte(forbidden)) {
			return nil, fmt.Errorf("generated cask contains forbidden download marker %q", forbidden)
		}
	}
	wanted := make(map[string]bool, len(archiveNames))
	for _, name := range archiveNames {
		wanted[`url "`+browserURLPrefix+name+`"`] = false
	}
	for _, line := range strings.Split(string(cask), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "url ") {
			continue
		}
		seen, ok := wanted[line]
		if !ok || seen {
			return nil, fmt.Errorf("generated cask contains an unexpected or duplicate download URL")
		}
		wanted[line] = true
	}
	for _, name := range archiveNames {
		if !wanted[`url "`+browserURLPrefix+name+`"`] {
			return nil, fmt.Errorf("generated cask is missing public download for %q", name)
		}
	}
	return rewritePostflight(cask)
}

func rewritePostflight(cask []byte) ([]byte, error) {
	legacyCount := bytes.Count(cask, legacyPostflight)
	structuredCount := bytes.Count(cask, structuredPostflight)
	switch {
	case legacyCount == 1 && structuredCount == 0:
		return bytes.Replace(cask, legacyPostflight, structuredPostflight, 1), nil
	case legacyCount == 0 && structuredCount == 1:
		return bytes.Clone(cask), nil
	default:
		return nil, fmt.Errorf("generated cask contains legacy postflight %d times and structured postflight_steps %d times; want exactly one supported form", legacyCount, structuredCount)
	}
}

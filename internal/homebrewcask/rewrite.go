// Package homebrewcask prepares a GoReleaser-generated cask for private assets.
package homebrewcask

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

var archiveNames = [...]string{
	"achta_Darwin_arm64.tar.gz",
	"achta_Darwin_x86_64.tar.gz",
	"achta_Linux_arm64.tar.gz",
	"achta_Linux_x86_64.tar.gz",
}

const (
	browserURLPrefix  = "https://github.com/ozgurcd/achta/releases/download/v#{version}/"
	assetAPIURLPrefix = "https://api.github.com/repos/ozgurcd/achta/releases/assets/"
)

// Rewrite replaces each browser download URL with the corresponding release
// asset API URL. GitHub requires the API endpoint for authenticated private
// release downloads.
func Rewrite(cask, releaseJSON []byte) ([]byte, error) {
	for _, header := range []string{
		"Accept: application/octet-stream",
		"Authorization: Bearer #{ENV.fetch(\"HOMEBREW_GITHUB_API_TOKEN\", \"\")}",
	} {
		if count := bytes.Count(cask, []byte(header)); count != len(archiveNames) {
			return nil, fmt.Errorf("generated cask contains required header %q %d times, want %d", header, count, len(archiveNames))
		}
	}

	var release struct {
		Assets []struct {
			Name string `json:"name"`
			URL  string `json:"url"`
		} `json:"assets"`
	}
	if err := json.Unmarshal(releaseJSON, &release); err != nil {
		return nil, fmt.Errorf("decode release metadata: %w", err)
	}

	wanted := make(map[string]bool, len(archiveNames))
	for _, name := range archiveNames {
		wanted[name] = true
	}
	assetURLs := make(map[string]string, len(archiveNames))
	for _, asset := range release.Assets {
		if !wanted[asset.Name] {
			continue
		}
		if _, duplicate := assetURLs[asset.Name]; duplicate {
			return nil, fmt.Errorf("release metadata contains duplicate asset %q", asset.Name)
		}
		assetID := strings.TrimPrefix(asset.URL, assetAPIURLPrefix)
		if assetID == asset.URL || assetID == "" {
			return nil, fmt.Errorf("asset %q has unexpected API URL %q", asset.Name, asset.URL)
		}
		if strings.Trim(assetID, "0123456789") != "" {
			return nil, fmt.Errorf("asset %q has invalid API URL %q", asset.Name, asset.URL)
		}
		if _, err := strconv.ParseUint(assetID, 10, 64); err != nil {
			return nil, fmt.Errorf("asset %q has invalid API URL %q", asset.Name, asset.URL)
		}
		assetURLs[asset.Name] = asset.URL
	}

	rewritten := bytes.Clone(cask)
	for _, name := range archiveNames {
		apiURL, ok := assetURLs[name]
		if !ok {
			return nil, fmt.Errorf("release metadata is missing asset %q", name)
		}
		browserURL := []byte(browserURLPrefix + name)
		if count := bytes.Count(rewritten, browserURL); count != 1 {
			return nil, fmt.Errorf("generated cask contains %d browser URLs for %q, want 1", count, name)
		}
		rewritten = bytes.Replace(rewritten, browserURL, []byte(apiURL), 1)
	}
	if bytes.Contains(rewritten, []byte(browserURLPrefix)) {
		return nil, fmt.Errorf("generated cask retains a private release browser URL")
	}
	return rewritten, nil
}

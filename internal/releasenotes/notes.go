package releasenotes

import (
	"bytes"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

const MaxFile = 1 << 20

var versionPattern = regexp.MustCompile(`^v(?:0|[1-9][0-9]*)\.(?:0|[1-9][0-9]*)\.(?:0|[1-9][0-9]*)$`)

var headingPattern = regexp.MustCompile(`^## (v(?:0|[1-9][0-9]*)\.(?:0|[1-9][0-9]*)\.(?:0|[1-9][0-9]*)) — ([0-9]{4}-[0-9]{2}-[0-9]{2})$`)

const unreleasedHeading = "## Unreleased"

// Extract returns exactly one version section, including its heading and final
// newline, from the persistent newest-first release notes document.
func Extract(data []byte, version string) ([]byte, error) {
	if !versionPattern.MatchString(version) {
		return nil, errors.New("release version must be vMAJOR.MINOR.PATCH")
	}
	if len(data) == 0 || len(data) > MaxFile {
		return nil, fmt.Errorf("release notes must contain 1-%d bytes", MaxFile)
	}
	normalized := bytes.ReplaceAll(data, []byte("\r\n"), []byte("\n"))
	lines := strings.Split(string(normalized), "\n")
	start, end, matches := -1, len(lines), 0
	unreleasedStart, unreleasedEnd, unreleasedMatches := -1, len(lines), 0
	seenRelease := false
	for index, line := range lines {
		if !strings.HasPrefix(line, "## ") {
			continue
		}
		if line == unreleasedHeading {
			unreleasedMatches++
			if unreleasedMatches > 1 {
				return nil, errors.New("release notes contain duplicate Unreleased sections")
			}
			if seenRelease {
				return nil, errors.New("release notes Unreleased section must precede version sections")
			}
			unreleasedStart = index
			continue
		}
		if unreleasedStart >= 0 && unreleasedEnd == len(lines) {
			unreleasedEnd = index
		}
		if start >= 0 && end == len(lines) {
			end = index
		}
		match := headingPattern.FindStringSubmatch(line)
		if match == nil {
			return nil, fmt.Errorf("invalid release notes heading on line %d", index+1)
		}
		if _, err := time.Parse("2006-01-02", match[2]); err != nil {
			return nil, fmt.Errorf("invalid release notes date on line %d", index+1)
		}
		seenRelease = true
		if match[1] == version {
			matches++
			if start < 0 {
				start = index
			}
		}
	}
	if unreleasedStart >= 0 {
		hasContent := false
		for _, line := range lines[unreleasedStart+1 : unreleasedEnd] {
			if strings.TrimSpace(line) != "" {
				hasContent = true
				break
			}
		}
		if !hasContent {
			return nil, errors.New("release notes Unreleased section is empty")
		}
	}
	if matches != 1 {
		return nil, fmt.Errorf("release notes contain %d sections for %s", matches, version)
	}
	for end > start && lines[end-1] == "" {
		end--
	}
	if end <= start+1 {
		return nil, fmt.Errorf("release notes section %s is empty", version)
	}
	return []byte(strings.Join(lines[start:end], "\n") + "\n"), nil
}

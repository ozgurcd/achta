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
	for index, line := range lines {
		if !strings.HasPrefix(line, "## ") {
			continue
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
		if match[1] == version {
			matches++
			if start < 0 {
				start = index
			}
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

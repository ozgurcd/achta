package reachability

import (
	"errors"
	"fmt"
	"path"
	"sort"
	"strings"
)

const Schema = "achta.reachability.v1"

type Exclusion struct {
	Pattern string `json:"pattern"`
	Why     string `json:"why"`
}

type ExcludedPath struct {
	Path    string `json:"path"`
	Pattern string `json:"pattern"`
	Why     string `json:"why"`
}

type Result struct {
	SchemaVersion string         `json:"schema_version"`
	Decision      string         `json:"decision"`
	Changed       []string       `json:"changed"`
	Excluded      []ExcludedPath `json:"excluded"`
	Reaching      []string       `json:"reaching"`
	Unknown       []string       `json:"unknown"`
}

func Classify(changed []string, exclusions []Exclusion) (Result, error) {
	result := Result{
		SchemaVersion: Schema,
		Decision:      "SKIPPABLE",
		Changed:       append([]string(nil), changed...),
		Excluded:      []ExcludedPath{},
		Reaching:      []string{},
		Unknown:       []string{},
	}
	sort.Strings(result.Changed)
	for i, exclusion := range exclusions {
		if err := validateExclusion(exclusion); err != nil {
			return Result{}, fmt.Errorf("exclusion %d: %w", i+1, err)
		}
	}
	for _, changedPath := range result.Changed {
		if err := validatePath(changedPath); err != nil {
			return Result{}, err
		}
		matched := false
		for _, exclusion := range exclusions {
			if match(exclusion.Pattern, changedPath) {
				result.Excluded = append(result.Excluded, ExcludedPath{Path: changedPath, Pattern: exclusion.Pattern, Why: exclusion.Why})
				matched = true
				break
			}
		}
		if !matched {
			result.Reaching = append(result.Reaching, changedPath)
			result.Unknown = append(result.Unknown, changedPath)
		}
	}
	if len(result.Reaching) > 0 {
		result.Decision = "REQUIRED"
	}
	return result, nil
}

func validateExclusion(exclusion Exclusion) error {
	if exclusion.Pattern == "" || exclusion.Why == "" {
		return errors.New("pattern and why are required")
	}
	if strings.ContainsAny(exclusion.Pattern+exclusion.Why, "\x00\r\n") {
		return errors.New("pattern and why must be single-line values")
	}
	clean := path.Clean(exclusion.Pattern)
	if strings.HasPrefix(exclusion.Pattern, "/") || clean == ".." || strings.HasPrefix(clean, "../") || strings.Contains(exclusion.Pattern, "\\") {
		return errors.New("pattern must be repository-relative")
	}
	if isCatchAll(exclusion.Pattern) {
		return errors.New("catch-all no-reach patterns are forbidden")
	}
	if _, err := path.Match(exclusion.Pattern, "validation/path"); err != nil {
		return fmt.Errorf("invalid pattern: %w", err)
	}
	return nil
}

func validatePath(value string) error {
	clean := path.Clean(value)
	if value == "" || value != clean || strings.HasPrefix(value, "/") || clean == ".." || strings.HasPrefix(clean, "../") || strings.Contains(value, "\\") || strings.ContainsAny(value, "\x00\r\n") {
		return fmt.Errorf("invalid changed path %q", value)
	}
	return nil
}

func isCatchAll(pattern string) bool {
	switch strings.TrimPrefix(pattern, "./") {
	case "*", "**", "**/*", "**/**":
		return true
	default:
		return false
	}
}

func match(pattern, changedPath string) bool {
	if strings.HasSuffix(pattern, "/**") {
		prefix := strings.TrimSuffix(pattern, "**")
		return strings.HasPrefix(changedPath, prefix)
	}
	if strings.HasPrefix(pattern, "**/") {
		suffix := strings.TrimPrefix(pattern, "**/")
		return changedPath == suffix || strings.HasSuffix(changedPath, "/"+suffix)
	}
	matched, _ := path.Match(pattern, changedPath)
	return matched
}

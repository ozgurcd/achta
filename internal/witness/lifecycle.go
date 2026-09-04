package witness

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"time"
)

var ErrOperationRefused = errors.New("witness operation refused")

func Init(label, repoHead string, targets []string, started time.Time) ([]byte, error) {
	if err := validateOneLine("gate label", label, 512); err != nil {
		return nil, err
	}
	if !shaPattern.MatchString(repoHead) {
		return nil, errors.New("repository HEAD must be a full lowercase Git SHA")
	}
	if len(targets) == 0 || len(targets) > 1000 {
		return nil, errors.New("target plan must contain 1-1000 names")
	}
	seen := make(map[string]bool)
	for _, target := range targets {
		if !namePattern.MatchString(target) || seen[target] {
			return nil, fmt.Errorf("invalid or duplicate target %q", target)
		}
		seen[target] = true
	}
	rendered := []byte(fmt.Sprintf("schema: %s\ngate: %s\nrepo-head: %s\nstarted: %s\nplan: %s\n", Schema, label, repoHead, started.UTC().Format(time.RFC3339), strings.Join(targets, " ")))
	if _, err := Parse(rendered); err != nil {
		return nil, fmt.Errorf("rendered witness validation: %w", err)
	}
	return rendered, nil
}

func Step(source []byte, target string, exitCode int, elapsedMS *int64, evidence []string) ([]byte, error) {
	record, err := Parse(source)
	if err != nil {
		return nil, err
	}
	if record.Finished != nil || record.Result != "" || record.TreeKind != "" {
		return nil, fmt.Errorf("%w: witness record is already finalized", ErrOperationRefused)
	}
	planned := false
	for _, name := range record.Plan {
		if name == target {
			planned = true
			break
		}
	}
	if !planned {
		return nil, fmt.Errorf("%w: target %q is not planned", ErrOperationRefused, target)
	}
	if _, exists := record.Targets[target]; exists {
		return nil, fmt.Errorf("%w: target %q is already recorded", ErrOperationRefused, target)
	}
	if elapsedMS != nil && *elapsedMS < 0 {
		return nil, errors.New("elapsed milliseconds must be non-negative")
	}
	var addition strings.Builder
	for _, line := range evidence {
		if err := validateOneLine("evidence", line, 4096); err != nil {
			return nil, err
		}
		fmt.Fprintf(&addition, "evidence: [%s] %s\n", target, line)
	}
	if elapsedMS != nil {
		fmt.Fprintf(&addition, "elapsed: %s %dms\n", target, *elapsedMS)
	}
	fmt.Fprintf(&addition, "target: %s exit=%d\n", target, exitCode)
	rendered := appendRecord(source, addition.String())
	if _, err := Parse(rendered); err != nil {
		return nil, fmt.Errorf("rendered witness validation: %w", err)
	}
	return rendered, nil
}

func Finalize(source []byte, treeKind, treeValue string, finished time.Time) ([]byte, error) {
	record, err := Parse(source)
	if err != nil {
		return nil, err
	}
	if record.Finished != nil || record.Result != "" || record.TreeKind != "" {
		return nil, fmt.Errorf("%w: witness record is already finalized", ErrOperationRefused)
	}
	if record.Started != nil && finished.Before(*record.Started) {
		return nil, errors.New("finished timestamp precedes started timestamp")
	}
	green := true
	for _, target := range record.Plan {
		exitCode, ok := record.Targets[target]
		if !ok || exitCode != 0 {
			green = false
		}
	}
	var addition strings.Builder
	fmt.Fprintf(&addition, "finished: %s\n", finished.UTC().Format(time.RFC3339))
	if green {
		switch treeKind {
		case "sha256":
			if !digestPattern.MatchString(treeValue) {
				return nil, errors.New("invalid tree digest")
			}
		case "commit":
			if !shaPattern.MatchString(treeValue) {
				return nil, errors.New("invalid tree commit")
			}
		default:
			return nil, errors.New("tree kind must be sha256 or commit")
		}
		fmt.Fprintf(&addition, "tree: %s=%s\nresult: green\n", treeKind, treeValue)
	} else {
		addition.WriteString("result: red\n")
	}
	rendered := appendRecord(source, addition.String())
	if _, err := Parse(rendered); err != nil {
		return nil, fmt.Errorf("rendered witness validation: %w", err)
	}
	return rendered, nil
}

func Evidence(data []byte) ([]string, error) {
	if len(data) == 0 {
		return nil, nil
	}
	normalized := bytes.ReplaceAll(data, []byte("\r\n"), []byte("\n"))
	lines := strings.Split(strings.TrimSuffix(string(normalized), "\n"), "\n")
	if len(lines) > 1000 {
		return nil, errors.New("evidence exceeds 1000 lines")
	}
	for _, line := range lines {
		if err := validateOneLine("evidence", line, 4096); err != nil {
			return nil, err
		}
	}
	return lines, nil
}

func appendRecord(source []byte, addition string) []byte {
	base := bytes.TrimSuffix(source, []byte("\n"))
	result := make([]byte, 0, len(base)+1+len(addition))
	result = append(result, base...)
	result = append(result, '\n')
	result = append(result, addition...)
	return result
}

func validateOneLine(name, value string, limit int) error {
	if value == "" || len(value) > limit || strings.TrimSpace(value) != value || strings.ContainsAny(value, "\r\n\x00") {
		return fmt.Errorf("%s must be one trimmed line of 1-%d bytes", name, limit)
	}
	return nil
}

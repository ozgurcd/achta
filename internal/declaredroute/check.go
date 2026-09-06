package declaredroute

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"

	"go.yaml.in/yaml/v3"
)

const (
	Schema       = "achta.declared-route-check.v1"
	MaxDocument  = 8 << 20
	MaxDocuments = 256
	MaxPattern   = 4096
	MaxPatterns  = 64
	MaxMatches   = 4096

	RuleRouteCount  = "route-count"
	RuleBanned      = "banned-pattern"
	RuleRequiredKey = "required-key-scope"

	RefusedLineScope  = "line-level scope inference; required scope is resolved from YAML mapping nodes"
	RefusedVocabulary = "default route, banned-pattern, required-key, and scope vocabulary"
)

type Document struct {
	Path string
	Data []byte
}

type Options struct {
	RoutePatterns        []string
	BanPatterns          []string
	RequiredRoutePattern string
	RequiredKey          string
	RequiredScope        string
}

type PatternCount struct {
	Pattern string `json:"pattern"`
	Count   int    `json:"count"`
	Lines   []int  `json:"lines"`
}

type KeyCount struct {
	RoutePattern string `json:"route_pattern"`
	Required     bool   `json:"required"`
	Scope        string `json:"scope"`
	Key          string `json:"key"`
	Count        int    `json:"count"`
	Lines        []int  `json:"lines"`
}

type DocumentResult struct {
	File        string         `json:"file"`
	Status      string         `json:"status"`
	Routes      []PatternCount `json:"routes"`
	Banned      []PatternCount `json:"banned"`
	RequiredKey KeyCount       `json:"required_key"`
}

type Violation struct {
	File     string `json:"file"`
	Line     int    `json:"line"`
	Rule     string `json:"rule"`
	Expected int    `json:"expected"`
	Observed int    `json:"observed"`
	Text     string `json:"text"`
}

type Result struct {
	SchemaVersion string           `json:"schema_version"`
	Status        string           `json:"status"`
	Documents     []DocumentResult `json:"documents"`
	Violations    []Violation      `json:"violations"`
	Refused       []string         `json:"refused"`
}

type compiledPattern struct {
	text string
	re   *regexp.Regexp
}

type scalarLine struct {
	line int
	text string
}

func Check(documents []Document, opts Options) (Result, error) {
	result := Result{
		SchemaVersion: Schema,
		Status:        "pass",
		Documents:     []DocumentResult{},
		Violations:    []Violation{},
		Refused:       []string{RefusedLineScope, RefusedVocabulary},
	}
	if len(documents) == 0 {
		return result, errors.New("at least one --file is required")
	}
	if len(documents) > MaxDocuments {
		return result, fmt.Errorf("document count exceeds %d", MaxDocuments)
	}
	routes, err := compilePatterns("route", opts.RoutePatterns)
	if err != nil {
		return result, err
	}
	banned, err := compilePatterns("ban", opts.BanPatterns)
	if err != nil {
		return result, err
	}
	if opts.RequiredRoutePattern == "" {
		return result, errors.New("--required-route-pattern is required")
	}
	if len(opts.RequiredRoutePattern) > MaxPattern {
		return result, fmt.Errorf("--required-route-pattern exceeds %d bytes", MaxPattern)
	}
	requiredRouteMatches := 0
	for _, route := range routes {
		if route.text == opts.RequiredRoutePattern {
			requiredRouteMatches++
		}
	}
	if requiredRouteMatches != 1 {
		return result, errors.New("--required-route-pattern must equal exactly one --route-pattern")
	}
	if opts.RequiredKey == "" {
		return result, errors.New("--required-key is required")
	}
	if len(opts.RequiredKey) > MaxPattern {
		return result, fmt.Errorf("--required-key exceeds %d bytes", MaxPattern)
	}
	if len(opts.RequiredScope) > MaxPattern {
		return result, fmt.Errorf("--required-scope exceeds %d bytes", MaxPattern)
	}
	scope, err := parsePointer(opts.RequiredScope)
	if err != nil {
		return result, fmt.Errorf("--required-scope: %w", err)
	}

	seen := map[string]bool{}
	for _, document := range documents {
		if document.Path == "" {
			return result, errors.New("document path is required")
		}
		if seen[document.Path] {
			return result, fmt.Errorf("document %q is repeated", document.Path)
		}
		seen[document.Path] = true
		if len(document.Data) > MaxDocument {
			return result, fmt.Errorf("document %q exceeds the size limit", document.Path)
		}
		root, err := decodeOne(document.Data)
		if err != nil {
			return result, fmt.Errorf("document %q: %w", document.Path, err)
		}
		lines := collectScalarLines(root)
		routeCounts, err := countPatterns(routes, lines)
		if err != nil {
			return result, fmt.Errorf("document %q route patterns: %w", document.Path, err)
		}
		banCounts, err := countPatterns(banned, lines)
		if err != nil {
			return result, fmt.Errorf("document %q ban patterns: %w", document.Path, err)
		}
		documentResult := DocumentResult{
			File:   document.Path,
			Status: "pass",
			Routes: routeCounts,
			Banned: banCounts,
			RequiredKey: KeyCount{
				RoutePattern: opts.RequiredRoutePattern,
				Scope:        opts.RequiredScope,
				Key:          opts.RequiredKey,
				Lines:        []int{},
			},
		}

		routeCount := 0
		routeLine := root.Line
		for _, count := range documentResult.Routes {
			routeCount += count.Count
			if len(count.Lines) > 0 {
				routeLine = count.Lines[0]
			}
		}
		if routeCount != 1 {
			documentResult.Status = "fail"
			result.Violations = append(result.Violations, Violation{
				File: document.Path, Line: routeLine, Rule: RuleRouteCount,
				Expected: 1, Observed: routeCount,
				Text: "configured route alternatives must appear exactly once across the workflow",
			})
		}
		for _, count := range documentResult.Banned {
			for _, line := range count.Lines {
				documentResult.Status = "fail"
				result.Violations = append(result.Violations, Violation{
					File: document.Path, Line: line, Rule: RuleBanned,
					Expected: 0, Observed: count.Count,
					Text: fmt.Sprintf("configured banned pattern %q appears in a live YAML scalar", count.Pattern),
				})
			}
		}

		for _, count := range documentResult.Routes {
			if count.Pattern == opts.RequiredRoutePattern && count.Count > 0 {
				documentResult.RequiredKey.Required = true
			}
		}
		if documentResult.RequiredKey.Required {
			scopeNode, scopeLine, err := mappingAt(root, scope)
			if err != nil {
				return result, fmt.Errorf("document %q required scope %q: %w", document.Path, opts.RequiredScope, err)
			}
			keyLines := []int{}
			if scopeNode != nil {
				keyLines = mappingKeyLines(scopeNode, opts.RequiredKey)
			}
			documentResult.RequiredKey.Count = len(keyLines)
			documentResult.RequiredKey.Lines = keyLines
			if len(keyLines) != 1 {
				documentResult.Status = "fail"
				line := scopeLine
				if len(keyLines) > 0 {
					line = keyLines[0]
				}
				result.Violations = append(result.Violations, Violation{
					File: document.Path, Line: line, Rule: RuleRequiredKey,
					Expected: 1, Observed: len(keyLines),
					Text: fmt.Sprintf("required key %q must appear exactly once in YAML mapping %q for route %q", opts.RequiredKey, opts.RequiredScope, opts.RequiredRoutePattern),
				})
			}
		}
		result.Documents = append(result.Documents, documentResult)
	}
	if len(result.Violations) > 0 {
		result.Status = "fail"
	}
	return result, nil
}

func compilePatterns(kind string, values []string) ([]compiledPattern, error) {
	if len(values) == 0 {
		return nil, fmt.Errorf("at least one --%s-pattern is required", kind)
	}
	if len(values) > MaxPatterns {
		return nil, fmt.Errorf("--%s-pattern count exceeds %d", kind, MaxPatterns)
	}
	seen := map[string]bool{}
	result := make([]compiledPattern, 0, len(values))
	for _, value := range values {
		if value == "" {
			return nil, fmt.Errorf("--%s-pattern must not be empty", kind)
		}
		if len(value) > MaxPattern {
			return nil, fmt.Errorf("--%s-pattern exceeds %d bytes", kind, MaxPattern)
		}
		if seen[value] {
			return nil, fmt.Errorf("--%s-pattern %q is repeated", kind, value)
		}
		seen[value] = true
		re, err := regexp.Compile(value)
		if err != nil {
			return nil, fmt.Errorf("--%s-pattern %q: %w", kind, value, err)
		}
		if re.MatchString("") {
			return nil, fmt.Errorf("--%s-pattern %q matches empty text", kind, value)
		}
		result = append(result, compiledPattern{text: value, re: re})
	}
	return result, nil
}

func decodeOne(data []byte) (*yaml.Node, error) {
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	var document yaml.Node
	if err := decoder.Decode(&document); err != nil {
		if errors.Is(err, io.EOF) {
			return nil, errors.New("empty YAML document")
		}
		return nil, fmt.Errorf("parse YAML: %w", err)
	}
	if document.Kind != yaml.DocumentNode || len(document.Content) != 1 {
		return nil, errors.New("YAML must contain one non-empty document")
	}
	var extra yaml.Node
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err != nil {
			return nil, fmt.Errorf("parse trailing YAML: %w", err)
		}
		return nil, errors.New("multiple YAML documents are not supported")
	}
	root := document.Content[0]
	if root.Kind != yaml.MappingNode {
		return nil, errors.New("workflow document root must be a YAML mapping")
	}
	return root, nil
}

func collectScalarLines(root *yaml.Node) []scalarLine {
	lines := []scalarLine{}
	var walk func(*yaml.Node)
	walk = func(node *yaml.Node) {
		switch node.Kind {
		case yaml.MappingNode:
			for i := 1; i < len(node.Content); i += 2 {
				walk(node.Content[i])
			}
		case yaml.SequenceNode, yaml.DocumentNode:
			for _, child := range node.Content {
				walk(child)
			}
		case yaml.ScalarNode:
			for offset, line := range strings.Split(node.Value, "\n") {
				if strings.HasPrefix(strings.TrimSpace(line), "#") {
					continue
				}
				lines = append(lines, scalarLine{line: node.Line + offset, text: line})
			}
		}
	}
	walk(root)
	return lines
}

func countPatterns(patterns []compiledPattern, lines []scalarLine) ([]PatternCount, error) {
	result := make([]PatternCount, 0, len(patterns))
	totalMatches := 0
	for _, pattern := range patterns {
		count := PatternCount{Pattern: pattern.text, Lines: []int{}}
		for _, line := range lines {
			matches := pattern.re.FindAllStringIndex(line.text, MaxMatches+1-totalMatches)
			count.Count += len(matches)
			totalMatches += len(matches)
			if totalMatches > MaxMatches {
				return nil, fmt.Errorf("match count exceeds %d", MaxMatches)
			}
			for range matches {
				count.Lines = append(count.Lines, line.line)
			}
		}
		result = append(result, count)
	}
	return result, nil
}

func parsePointer(value string) ([]string, error) {
	if value == "" || !strings.HasPrefix(value, "/") {
		return nil, errors.New("must be a non-empty JSON Pointer beginning with '/'")
	}
	parts := strings.Split(value[1:], "/")
	for i, part := range parts {
		var decoded strings.Builder
		for j := 0; j < len(part); j++ {
			if part[j] != '~' {
				decoded.WriteByte(part[j])
				continue
			}
			if j+1 >= len(part) || (part[j+1] != '0' && part[j+1] != '1') {
				return nil, fmt.Errorf("segment %q has an invalid escape", part)
			}
			j++
			if part[j] == '0' {
				decoded.WriteByte('~')
			} else {
				decoded.WriteByte('/')
			}
		}
		parts[i] = decoded.String()
	}
	return parts, nil
}

func mappingAt(root *yaml.Node, path []string) (*yaml.Node, int, error) {
	current := root
	line := root.Line
	for _, segment := range path {
		if current.Kind == yaml.AliasNode {
			if current.Alias == nil {
				return nil, line, errors.New("unresolved YAML alias")
			}
			current = current.Alias
		}
		if current.Kind != yaml.MappingNode {
			return nil, line, fmt.Errorf("path segment %q is not inside a mapping", segment)
		}
		matches := []*yaml.Node{}
		for i := 0; i+1 < len(current.Content); i += 2 {
			key := current.Content[i]
			if key.Kind == yaml.ScalarNode && key.Value == segment {
				matches = append(matches, current.Content[i+1])
				line = key.Line
			}
		}
		if len(matches) == 0 {
			return nil, line, nil
		}
		if len(matches) != 1 {
			return nil, line, fmt.Errorf("path segment %q is declared %d times", segment, len(matches))
		}
		current = matches[0]
	}
	if current.Kind == yaml.AliasNode {
		if current.Alias == nil {
			return nil, line, errors.New("unresolved YAML alias")
		}
		current = current.Alias
	}
	if current.Kind != yaml.MappingNode {
		return nil, line, errors.New("selected scope is not a YAML mapping")
	}
	return current, line, nil
}

func mappingKeyLines(mapping *yaml.Node, keyValue string) []int {
	lines := []int{}
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		key := mapping.Content[i]
		if key.Kind == yaml.ScalarNode && key.Value == keyValue {
			lines = append(lines, key.Line)
		}
	}
	return lines
}

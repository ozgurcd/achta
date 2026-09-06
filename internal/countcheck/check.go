// SPDX-License-Identifier: MIT
package countcheck

import (
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"
)

const (
	// Schema is the stable count-check machine interface.
	Schema = "achta.count-check.v1"
	// MaxDocument bounds every Markdown document read by count check.
	MaxDocument = 8 << 20
	// MaxDocuments bounds one invocation's input set.
	MaxDocuments = 256
	// MaxDirectoryEntries bounds each explicitly selected input directory.
	MaxDirectoryEntries = 4096

	RuleBreakdownSum  = "breakdown-sum"
	RuleClaimCitation = "claim-citation-count"
	RuleClaimProof    = "claim-proof-count"

	RefusedClaimMeaning    = "whether unconfigured prose makes a count claim"
	RefusedCitationMeaning = "whether a matched citation proves a disposition"
	RefusedProofMeaning    = "whether a matched proof count proves a claim"
)

// Document is one explicitly selected Markdown input.
type Document struct {
	Path string
	Data []byte
}

// Options contains caller-owned vocabulary. Count patterns identify their
// numeric token through one named capture group called "count".
type Options struct {
	TotalPattern           string
	PartPattern            string
	ClaimPatterns          []string
	ClaimTargetPatterns    []string
	ClaimAssertionPatterns []string
	CitationPattern        string
	ProofClaimPatterns     []string
	ProofPatterns          []string
	ProofWithinLines       int
	ClaimSectionPattern    string
	ExemptPatterns         []string
	CountAliases           []string
}

// Violation records one evaluated arithmetic disagreement.
type Violation struct {
	File     string `json:"file"`
	Line     int    `json:"line"`
	Rule     string `json:"rule"`
	Claimed  int    `json:"claimed"`
	Observed int    `json:"observed"`
	Text     string `json:"text"`
}

// Result is the stable count-check result.
type Result struct {
	SchemaVersion string      `json:"schema_version"`
	Status        string      `json:"status"`
	Files         int         `json:"files"`
	Breakdowns    int         `json:"breakdowns"`
	Claims        int         `json:"claims"`
	ProofClaims   int         `json:"proof_claims,omitempty"`
	Violations    []Violation `json:"violations"`
	Refused       []string    `json:"refused"`
}

type countPattern struct {
	re         *regexp.Regexp
	countIndex int
}

// Check reconciles caller-declared count relationships without inferring prose
// meaning. Documents are evaluated in caller order and violations in source
// order.
func Check(documents []Document, opts Options) (Result, error) {
	result := Result{
		SchemaVersion: Schema,
		Status:        "pass",
		Violations:    []Violation{},
		Refused:       []string{RefusedClaimMeaning, RefusedCitationMeaning},
	}
	if len(documents) == 0 {
		return result, errors.New("at least one document is required")
	}
	if len(documents) > MaxDocuments {
		return result, fmt.Errorf("document count exceeds %d", MaxDocuments)
	}

	breakdownEnabled := opts.TotalPattern != "" || opts.PartPattern != ""
	composedClaimEnabled := len(opts.ClaimTargetPatterns) > 0 || len(opts.ClaimAssertionPatterns) > 0
	claimEnabled := len(opts.ClaimPatterns) > 0 || composedClaimEnabled || opts.CitationPattern != ""
	proofEnabled := len(opts.ProofClaimPatterns) > 0 || len(opts.ProofPatterns) > 0 || opts.ProofWithinLines != 0
	if proofEnabled {
		result.Refused = append(result.Refused, RefusedProofMeaning)
	}
	if !breakdownEnabled && !claimEnabled && !proofEnabled {
		return result, errors.New("at least one pattern pair is required")
	}
	if (opts.TotalPattern == "") != (opts.PartPattern == "") {
		return result, errors.New("--total-pattern and --part-pattern must be supplied together")
	}
	if (len(opts.ClaimTargetPatterns) == 0) != (len(opts.ClaimAssertionPatterns) == 0) {
		return result, errors.New("--claim-target-pattern and --claim-assertion-pattern must be supplied together")
	}
	if (len(opts.ClaimPatterns) == 0 && !composedClaimEnabled) != (opts.CitationPattern == "") {
		return result, errors.New("a claim pattern relationship and --citation-pattern must be supplied together")
	}
	if proofEnabled && (len(opts.ProofClaimPatterns) == 0 || len(opts.ProofPatterns) == 0 || opts.ProofWithinLines < 1) {
		return result, errors.New("--proof-claim-pattern, --proof-pattern, and a positive --proof-within-lines must be supplied together")
	}
	if opts.ClaimSectionPattern != "" && !claimEnabled {
		return result, errors.New("--claim-section-pattern requires a claim pattern relationship and --citation-pattern")
	}
	if len(opts.ExemptPatterns) > 0 && !claimEnabled && !proofEnabled {
		return result, errors.New("--exempt-pattern requires a claim or proof relationship")
	}

	aliases, err := parseAliases(opts.CountAliases)
	if err != nil {
		return result, err
	}
	var total, part *countPattern
	var claims, claimTargets, proofClaims, proofs []*countPattern
	var citation *regexp.Regexp
	var claimSection *regexp.Regexp
	var claimAssertions, exemptions []*regexp.Regexp
	if breakdownEnabled {
		if total, err = compileCountPattern("total", opts.TotalPattern); err != nil {
			return result, err
		}
		if part, err = compileCountPattern("part", opts.PartPattern); err != nil {
			return result, err
		}
	}
	if claimEnabled {
		if len(opts.ClaimPatterns) > 0 {
			if claims, err = compileCountPatterns("claim", opts.ClaimPatterns); err != nil {
				return result, err
			}
		}
		if composedClaimEnabled {
			if claimTargets, err = compileCountPatterns("claim target", opts.ClaimTargetPatterns); err != nil {
				return result, err
			}
			if claimAssertions, err = compileTextPatterns("claim assertion", opts.ClaimAssertionPatterns); err != nil {
				return result, err
			}
		}
		if citation, err = regexp.Compile(opts.CitationPattern); err != nil {
			return result, fmt.Errorf("citation pattern: %w", err)
		}
		if citation.MatchString("") {
			return result, errors.New("citation pattern must not match empty text")
		}
		if opts.ClaimSectionPattern != "" {
			if claimSection, err = compileTextPattern("claim section", opts.ClaimSectionPattern); err != nil {
				return result, err
			}
		}
	}
	if proofEnabled {
		if proofClaims, err = compileCountPatterns("proof claim", opts.ProofClaimPatterns); err != nil {
			return result, err
		}
		if proofs, err = compileCountPatterns("proof", opts.ProofPatterns); err != nil {
			return result, err
		}
	}
	if len(opts.ExemptPatterns) > 0 {
		if exemptions, err = compileTextPatterns("exempt", opts.ExemptPatterns); err != nil {
			return result, err
		}
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
		if !utf8.Valid(document.Data) {
			return result, fmt.Errorf("document %q is not valid UTF-8", document.Path)
		}
		result.Files++
		lines := strings.Split(strings.ReplaceAll(string(document.Data), "\r\n", "\n"), "\n")
		exempt := exemptedLines(lines, exemptions)
		var documentViolations []Violation
		if breakdownEnabled {
			count, violations, checkErr := checkBreakdowns(document.Path, lines, total, part, aliases)
			if checkErr != nil {
				return result, checkErr
			}
			result.Breakdowns += count
			documentViolations = append(documentViolations, violations...)
		}
		if claimEnabled {
			count, violations, checkErr := checkClaims(document.Path, lines, claims, claimTargets, claimAssertions, citation, claimSection, exempt, aliases)
			if checkErr != nil {
				return result, checkErr
			}
			result.Claims += count
			documentViolations = append(documentViolations, violations...)
		}
		if proofEnabled {
			count, violations, checkErr := checkProofClaims(document.Path, lines, proofClaims, proofs, opts.ProofWithinLines, exempt, aliases)
			if checkErr != nil {
				return result, checkErr
			}
			result.ProofClaims += count
			documentViolations = append(documentViolations, violations...)
		}
		sort.SliceStable(documentViolations, func(i, j int) bool {
			return documentViolations[i].Line < documentViolations[j].Line
		})
		result.Violations = append(result.Violations, documentViolations...)
	}
	if len(result.Violations) > 0 {
		result.Status = "fail"
	}
	return result, nil
}

func compileCountPattern(name, value string) (*countPattern, error) {
	if len(value) > 4096 {
		return nil, fmt.Errorf("%s pattern exceeds 4096 bytes", name)
	}
	re, err := regexp.Compile(value)
	if err != nil {
		return nil, fmt.Errorf("%s pattern: %w", name, err)
	}
	index := -1
	for candidate, subname := range re.SubexpNames() {
		if subname != "count" {
			continue
		}
		if index >= 0 {
			return nil, fmt.Errorf("%s pattern repeats the named count capture", name)
		}
		index = candidate
	}
	if index < 1 {
		return nil, fmt.Errorf("%s pattern needs exactly one named count capture", name)
	}
	return &countPattern{re: re, countIndex: index}, nil
}

func compileCountPatterns(name string, values []string) ([]*countPattern, error) {
	seen := map[string]bool{}
	patterns := make([]*countPattern, 0, len(values))
	for _, value := range values {
		if value == "" {
			return nil, fmt.Errorf("%s pattern must not be empty", name)
		}
		if seen[value] {
			return nil, fmt.Errorf("%s pattern %q is repeated", name, value)
		}
		seen[value] = true
		pattern, err := compileCountPattern(name, value)
		if err != nil {
			return nil, err
		}
		patterns = append(patterns, pattern)
	}
	return patterns, nil
}

func compileTextPattern(name, value string) (*regexp.Regexp, error) {
	if len(value) > 4096 {
		return nil, fmt.Errorf("%s pattern exceeds 4096 bytes", name)
	}
	re, err := regexp.Compile(value)
	if err != nil {
		return nil, fmt.Errorf("%s pattern: %w", name, err)
	}
	if re.MatchString("") {
		return nil, fmt.Errorf("%s pattern must not match empty text", name)
	}
	return re, nil
}

func compileTextPatterns(name string, values []string) ([]*regexp.Regexp, error) {
	seen := map[string]bool{}
	patterns := make([]*regexp.Regexp, 0, len(values))
	for _, value := range values {
		if seen[value] {
			return nil, fmt.Errorf("%s pattern %q is repeated", name, value)
		}
		seen[value] = true
		pattern, err := compileTextPattern(name, value)
		if err != nil {
			return nil, err
		}
		patterns = append(patterns, pattern)
	}
	return patterns, nil
}

func parseAliases(values []string) (map[string]int, error) {
	aliases := map[string]int{}
	for _, value := range values {
		index := strings.LastIndex(value, "=")
		if index < 1 || index == len(value)-1 {
			return nil, fmt.Errorf("invalid count alias %q; want TOKEN=N", value)
		}
		token, number := value[:index], value[index+1:]
		if strings.ContainsAny(token, "\r\n") {
			return nil, fmt.Errorf("invalid count alias token %q", token)
		}
		parsed, err := decimal(number)
		if err != nil {
			return nil, fmt.Errorf("invalid count alias %q: %w", value, err)
		}
		if _, exists := aliases[token]; exists {
			return nil, fmt.Errorf("count alias token %q is repeated", token)
		}
		aliases[token] = parsed
	}
	return aliases, nil
}

func checkBreakdowns(path string, lines []string, total, part *countPattern, aliases map[string]int) (int, []Violation, error) {
	var violations []Violation
	breakdowns := 0
	for index, line := range lines {
		totals := total.re.FindAllStringSubmatch(line, -1)
		if len(totals) == 0 {
			continue
		}
		if len(totals) > 1 {
			return 0, nil, fmt.Errorf("document %q line %d has more than one stated total", path, index+1)
		}
		parts := part.re.FindAllStringSubmatch(line, -1)
		stated, err := countFromMatch(totals[0], total.countIndex, aliases)
		if err != nil {
			return 0, nil, fmt.Errorf("document %q line %d total: %w", path, index+1, err)
		}
		sum := 0
		for _, match := range parts {
			value, parseErr := countFromMatch(match, part.countIndex, aliases)
			if parseErr != nil {
				return 0, nil, fmt.Errorf("document %q line %d part: %w", path, index+1, parseErr)
			}
			if value > int(^uint(0)>>1)-sum {
				return 0, nil, fmt.Errorf("document %q line %d part sum overflows", path, index+1)
			}
			sum += value
		}
		breakdowns++
		if sum != stated {
			violations = append(violations, Violation{File: path, Line: index + 1, Rule: RuleBreakdownSum, Claimed: stated, Observed: sum, Text: fmt.Sprintf("breakdown parts sum to %d, stated total is %d", sum, stated)})
		}
	}
	return breakdowns, violations, nil
}

func checkClaims(path string, lines []string, claims, targets []*countPattern, assertions []*regexp.Regexp, citation, section *regexp.Regexp, exempt map[int]bool, aliases map[string]int) (int, []Violation, error) {
	var violations []Violation
	claimCount := 0
	scopeStart := 0
	if section != nil {
		scopeStart = -1
		for index, line := range lines {
			if strings.HasPrefix(line, "## ") && section.MatchString(line) {
				scopeStart = index
			}
		}
		if scopeStart < 0 {
			return 0, violations, nil
		}
	}
	for start := scopeStart; start < len(lines); {
		for start < len(lines) && strings.TrimSpace(lines[start]) == "" {
			start++
		}
		if start >= len(lines) {
			break
		}
		end := start
		for end < len(lines) && strings.TrimSpace(lines[end]) != "" {
			end++
		}
		paragraph := strings.Join(lines[start:end], "\n")
		masked, citations, err := maskCitations(paragraph, citation)
		if err != nil {
			return 0, nil, fmt.Errorf("document %q paragraph at line %d: %w", path, start+1, err)
		}
		for _, match := range countMatches(masked, claims) {
			lineIndex := start + strings.Count(masked[:match.indexes[0]], "\n")
			if paragraphIsExempt(start, end, exempt) {
				continue
			}
			claimed, parseErr := countFromIndexes(masked, match.indexes, match.pattern.countIndex, aliases)
			if parseErr != nil {
				return 0, nil, fmt.Errorf("document %q line %d claim: %w", path, lineIndex+1, parseErr)
			}
			claimCount++
			if len(citations) < claimed {
				violations = append(violations, Violation{File: path, Line: lineIndex + 1, Rule: RuleClaimCitation, Claimed: claimed, Observed: len(citations), Text: fmt.Sprintf("claim requires %d distinct citation(s), found %d in its paragraph", claimed, len(citations))})
			}
		}
		asserted := false
		for _, assertion := range assertions {
			if assertion.MatchString(masked) {
				asserted = true
				break
			}
		}
		if asserted && !paragraphIsExempt(start, end, exempt) {
			minimum := 0
			minimumLine := start
			for _, match := range countMatches(masked, targets) {
				claimed, parseErr := countFromIndexes(masked, match.indexes, match.pattern.countIndex, aliases)
				if parseErr != nil {
					lineIndex := start + strings.Count(masked[:match.indexes[0]], "\n")
					return 0, nil, fmt.Errorf("document %q line %d claim target: %w", path, lineIndex+1, parseErr)
				}
				if claimed > 0 && (minimum == 0 || claimed < minimum) {
					minimum = claimed
					minimumLine = start + strings.Count(masked[:match.indexes[0]], "\n")
				}
			}
			if minimum > 0 {
				claimCount++
				if len(citations) < minimum {
					violations = append(violations, Violation{File: path, Line: minimumLine + 1, Rule: RuleClaimCitation, Claimed: minimum, Observed: len(citations), Text: fmt.Sprintf("claim requires %d distinct citation(s), found %d in its paragraph", minimum, len(citations))})
				}
			}
		}
		start = end + 1
	}
	return claimCount, violations, nil
}

type locatedCountMatch struct {
	pattern *countPattern
	indexes []int
}

func countMatches(text string, patterns []*countPattern) []locatedCountMatch {
	matches := []locatedCountMatch{}
	seen := map[string]bool{}
	for _, pattern := range patterns {
		for _, indexes := range pattern.re.FindAllStringSubmatchIndex(text, -1) {
			key := fmt.Sprintf("%d:%d", indexes[0], indexes[1])
			if seen[key] {
				continue
			}
			seen[key] = true
			matches = append(matches, locatedCountMatch{pattern: pattern, indexes: indexes})
		}
	}
	sort.SliceStable(matches, func(i, j int) bool {
		if matches[i].indexes[0] != matches[j].indexes[0] {
			return matches[i].indexes[0] < matches[j].indexes[0]
		}
		return matches[i].indexes[1] < matches[j].indexes[1]
	})
	return matches
}

func exemptedLines(lines []string, patterns []*regexp.Regexp) map[int]bool {
	exempt := map[int]bool{}
	for index, line := range lines {
		matched := false
		for _, pattern := range patterns {
			if pattern.MatchString(line) {
				matched = true
				break
			}
		}
		if !matched {
			continue
		}
		start := index + 1
		for start < len(lines) && strings.TrimSpace(lines[start]) == "" {
			start++
		}
		for start < len(lines) && strings.TrimSpace(lines[start]) != "" {
			exempt[start] = true
			start++
		}
	}
	return exempt
}

func paragraphIsExempt(start, end int, exempt map[int]bool) bool {
	for index := start; index < end; index++ {
		if exempt[index] {
			return true
		}
	}
	return false
}

type proofCount struct {
	line  int
	value int
}

func checkProofClaims(path string, lines []string, claims, proofs []*countPattern, within int, exempt map[int]bool, aliases map[string]int) (int, []Violation, error) {
	proofCounts := []proofCount{}
	for index, line := range lines {
		for _, match := range countMatches(line, proofs) {
			value, err := countFromIndexes(line, match.indexes, match.pattern.countIndex, aliases)
			if err != nil {
				return 0, nil, fmt.Errorf("document %q line %d proof: %w", path, index+1, err)
			}
			proofCounts = append(proofCounts, proofCount{line: index, value: value})
		}
	}

	claimCount := 0
	violations := []Violation{}
	for index, line := range lines {
		if exempt[index] {
			continue
		}
		for _, match := range countMatches(line, claims) {
			claimed, err := countFromIndexes(line, match.indexes, match.pattern.countIndex, aliases)
			if err != nil {
				return 0, nil, fmt.Errorf("document %q line %d proof claim: %w", path, index+1, err)
			}
			claimCount++
			nearest := -1
			nearestDistance := within + 1
			for candidate, proof := range proofCounts {
				distance := index - proof.line
				if distance < 0 {
					distance = -distance
				}
				if distance <= within && distance < nearestDistance {
					nearest = candidate
					nearestDistance = distance
				}
			}
			if nearest < 0 {
				continue
			}
			proved := proofCounts[nearest].value
			if claimed != proved {
				violations = append(violations, Violation{File: path, Line: index + 1, Rule: RuleClaimProof, Claimed: claimed, Observed: proved, Text: fmt.Sprintf("claim count is %d, nearest proof count within %d line(s) is %d", claimed, within, proved)})
			}
		}
	}
	return claimCount, violations, nil
}

func maskCitations(text string, pattern *regexp.Regexp) (string, []string, error) {
	masked := []byte(text)
	unique := map[string]bool{}
	for _, match := range pattern.FindAllStringIndex(text, -1) {
		if match[0] == match[1] {
			return "", nil, errors.New("citation pattern matched empty text")
		}
		unique[text[match[0]:match[1]]] = true
		for index := match[0]; index < match[1]; index++ {
			if masked[index] != '\n' {
				masked[index] = ' '
			}
		}
	}
	citations := make([]string, 0, len(unique))
	for value := range unique {
		citations = append(citations, value)
	}
	sort.Strings(citations)
	return string(masked), citations, nil
}

func countFromMatch(match []string, index int, aliases map[string]int) (int, error) {
	if index >= len(match) {
		return 0, errors.New("named count capture did not participate")
	}
	return parseCount(match[index], aliases)
}

func countFromIndexes(text string, match []int, index int, aliases map[string]int) (int, error) {
	position := index * 2
	if position+1 >= len(match) || match[position] < 0 || match[position+1] < 0 {
		return 0, errors.New("named count capture did not participate")
	}
	return parseCount(text[match[position]:match[position+1]], aliases)
}

func parseCount(token string, aliases map[string]int) (int, error) {
	if value, err := decimal(token); err == nil {
		return value, nil
	}
	if value, ok := aliases[token]; ok {
		return value, nil
	}
	return 0, fmt.Errorf("count token %q is neither unsigned decimal nor a declared alias", token)
}

func decimal(value string) (int, error) {
	if value == "" {
		return 0, errors.New("count is empty")
	}
	for _, char := range value {
		if char < '0' || char > '9' {
			return 0, errors.New("count is not unsigned decimal")
		}
	}
	number, err := strconv.ParseInt(value, 10, 0)
	if err != nil {
		return 0, errors.New("count is out of range")
	}
	return int(number), nil
}

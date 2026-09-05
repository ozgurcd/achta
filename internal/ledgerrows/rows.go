package ledgerrows

import (
	"errors"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	Schema    = "achta.ledger-rows.v1"
	MaxLedger = 16 << 20
	maxMarker = 256
)

const (
	RuleOpenCompletion    = "open-completion-claim"
	RuleConditionQuote    = "close-condition-quote"
	RefusedClaimMeaning   = "natural-language completion inference; only explicit completion-marker literals are evaluated"
	RefusedConditionTruth = "condition satisfaction; only verbatim repetition after the quote marker is evaluated"
)

type Options struct {
	File              string
	IDCell            int
	ProseCell         int
	OpenMarker        string
	ClosedMarker      string
	IdentityEndMarker string
	CompletionMarkers []string
	ExemptMarkers     []string
	ConditionMarkers  []string
	QuoteMarker       string
}

type Violation struct {
	Line     int    `json:"line"`
	Rule     string `json:"rule"`
	Identity string `json:"identity"`
	Text     string `json:"text"`
}

type Result struct {
	SchemaVersion string      `json:"schema_version"`
	Status        string      `json:"status"`
	File          string      `json:"file"`
	Rows          int         `json:"rows"`
	OpenRows      int         `json:"open_rows"`
	ClosedRows    int         `json:"closed_rows"`
	Violations    []Violation `json:"violations"`
	Refused       []string    `json:"refused"`
}

func Check(data []byte, opts Options) (Result, error) {
	result := Result{
		SchemaVersion: Schema,
		Status:        "pass",
		File:          opts.File,
		Violations:    []Violation{},
		Refused:       []string{RefusedClaimMeaning, RefusedConditionTruth},
	}
	if len(data) > MaxLedger {
		return result, errors.New("ledger exceeds the size limit")
	}
	if err := validateOptions(opts); err != nil {
		return result, err
	}

	seen := map[string]int{}
	lines := strings.Split(string(data), "\n")
	for i, line := range lines {
		line = strings.TrimSuffix(line, "\r")
		cells, ok := splitTableRow(line)
		if !ok || len(cells) < opts.IDCell {
			continue
		}

		idCell := strings.TrimSpace(cells[opts.IDCell-1])
		open := strings.HasPrefix(idCell, opts.OpenMarker)
		closed := strings.HasPrefix(idCell, opts.ClosedMarker)
		if !open && !closed {
			continue
		}
		if open && closed {
			return result, fmt.Errorf("line %d matches both open and closed markers", i+1)
		}
		if len(cells) < opts.ProseCell {
			return result, fmt.Errorf("line %d has a state-marked ID cell but no prose cell %d", i+1, opts.ProseCell)
		}

		stateMarker := opts.OpenMarker
		if closed {
			stateMarker = opts.ClosedMarker
		}
		identityPart := strings.TrimSpace(strings.TrimPrefix(idCell, stateMarker))
		if !strings.HasSuffix(identityPart, opts.IdentityEndMarker) {
			return result, fmt.Errorf("line %d state-marked ID cell has no identity end marker", i+1)
		}
		identity := strings.TrimSpace(strings.TrimSuffix(identityPart, opts.IdentityEndMarker))
		if identity == "" || len(identity) > maxMarker || !utf8.ValidString(identity) || hasControl(identity) {
			return result, fmt.Errorf("line %d has an invalid row identity", i+1)
		}
		if first, duplicate := seen[identity]; duplicate {
			return result, fmt.Errorf("line %d repeats row identity %q first seen at line %d", i+1, identity, first)
		}
		seen[identity] = i + 1
		result.Rows++
		if open {
			result.OpenRows++
		} else {
			result.ClosedRows++
		}

		prose := strings.TrimSpace(cells[opts.ProseCell-1])
		if open && containsAny(prose, opts.CompletionMarkers) && !containsAny(prose, opts.ExemptMarkers) {
			result.Violations = append(result.Violations, Violation{
				Line:     i + 1,
				Rule:     RuleOpenCompletion,
				Identity: identity,
				Text:     "open ID cell conflicts with an explicit completion marker in the prose cell",
			})
		}
		if closed && containsAny(prose, opts.ConditionMarkers) {
			if ok, reason := hasVerbatimQuote(prose, opts.QuoteMarker); !ok {
				result.Violations = append(result.Violations, Violation{
					Line:     i + 1,
					Rule:     RuleConditionQuote,
					Identity: identity,
					Text:     reason,
				})
			}
		}
	}
	if result.Rows == 0 {
		return result, errors.New("no ledger rows matched the supplied state markers")
	}
	if len(result.Violations) > 0 {
		result.Status = "fail"
	}
	return result, nil
}

func validateOptions(opts Options) error {
	if opts.IDCell < 1 || opts.ProseCell < 1 {
		return errors.New("ID and prose cells must be positive one-based indexes")
	}
	if opts.IDCell == opts.ProseCell {
		return errors.New("ID and prose cells must be different")
	}
	for name, marker := range map[string]string{
		"open marker":         opts.OpenMarker,
		"closed marker":       opts.ClosedMarker,
		"identity end marker": opts.IdentityEndMarker,
		"quote marker":        opts.QuoteMarker,
	} {
		if err := validateMarker(name, marker); err != nil {
			return err
		}
	}
	if strings.HasPrefix(opts.OpenMarker, opts.ClosedMarker) || strings.HasPrefix(opts.ClosedMarker, opts.OpenMarker) {
		return errors.New("open and closed markers must not overlap as prefixes")
	}
	if err := validateMarkers("completion marker", opts.CompletionMarkers, true); err != nil {
		return err
	}
	if err := validateMarkers("condition marker", opts.ConditionMarkers, true); err != nil {
		return err
	}
	return validateMarkers("exempt marker", opts.ExemptMarkers, false)
}

func validateMarkers(name string, markers []string, required bool) error {
	if required && len(markers) == 0 {
		return fmt.Errorf("at least one %s is required", name)
	}
	seen := map[string]bool{}
	for _, marker := range markers {
		if err := validateMarker(name, marker); err != nil {
			return err
		}
		if seen[marker] {
			return fmt.Errorf("%s %q is repeated", name, marker)
		}
		seen[marker] = true
	}
	return nil
}

func validateMarker(name, marker string) error {
	if marker == "" {
		return fmt.Errorf("%s must not be empty", name)
	}
	if len(marker) > maxMarker || !utf8.ValidString(marker) || hasControl(marker) {
		return fmt.Errorf("%s is invalid", name)
	}
	return nil
}

func hasControl(value string) bool {
	return strings.IndexFunc(value, unicode.IsControl) >= 0
}

func containsAny(value string, markers []string) bool {
	for _, marker := range markers {
		if strings.Contains(value, marker) {
			return true
		}
	}
	return false
}

func hasVerbatimQuote(prose, marker string) (bool, string) {
	if strings.Count(prose, marker) != 1 {
		return false, "closed row stating a condition requires exactly one quote marker"
	}
	start := strings.Index(prose, marker)
	afterMarker := start + len(marker)
	quoteStart := afterMarker
	for quoteStart < len(prose) {
		r, size := utf8.DecodeRuneInString(prose[quoteStart:])
		if !unicode.IsSpace(r) {
			break
		}
		quoteStart += size
	}
	if quoteStart >= len(prose) || prose[quoteStart] != '"' {
		return false, "quote marker is not followed by a double-quoted string"
	}
	quoteEndOffset := strings.IndexByte(prose[quoteStart+1:], '"')
	if quoteEndOffset < 0 {
		return false, "quote marker has no closing double quote"
	}
	quoteEnd := quoteStart + 1 + quoteEndOffset
	quoted := prose[quoteStart+1 : quoteEnd]
	if quoted == "" {
		return false, "quote marker carries an empty quoted string"
	}
	rest := prose[:start] + prose[quoteEnd+1:]
	if !strings.Contains(rest, quoted) {
		return false, "quoted string does not appear verbatim elsewhere in the prose cell"
	}
	return true, ""
}

func splitTableRow(line string) ([]string, bool) {
	if len(line) < 2 || line[0] != '|' || line[len(line)-1] != '|' {
		return nil, false
	}
	inner := line[1 : len(line)-1]
	var cells []string
	start := 0
	for i := 0; i < len(inner); i++ {
		if inner[i] != '|' || escaped(inner, i) {
			continue
		}
		cells = append(cells, inner[start:i])
		start = i + 1
	}
	cells = append(cells, inner[start:])
	return cells, true
}

func escaped(value string, at int) bool {
	backslashes := 0
	for i := at - 1; i >= 0 && value[i] == '\\'; i-- {
		backslashes++
	}
	return backslashes%2 == 1
}

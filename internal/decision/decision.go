package decision

import (
	"bytes"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

const (
	MaxRegister = 16 << 20
	MaxBody     = 64 << 10
	MaxTitle    = 512
)

var (
	prefixPattern  = regexp.MustCompile(`^[A-Z][A-Z0-9]{0,7}$`)
	headingPattern = regexp.MustCompile(`^### ([A-Z][A-Z0-9]{0,7})-([0-9]{3,}) — (.+)$`)
)

type Result struct {
	Data    []byte
	ID      string
	Changed bool
	Line    int
}

// Add allocates the next ID for prefix and appends one decision to the same
// second-level section as the existing decisions carrying that prefix.
func Add(source []byte, prefix, title string, body []byte) (Result, error) {
	if len(source) == 0 {
		return Result{}, errors.New("empty decision register")
	}
	if len(source) > MaxRegister {
		return Result{}, fmt.Errorf("decision register exceeds %d-byte limit", MaxRegister)
	}
	if !prefixPattern.MatchString(prefix) {
		return Result{}, errors.New("decision prefix must be 1-8 uppercase letters or digits and start with a letter")
	}
	if err := validateTitle(title); err != nil {
		return Result{}, err
	}
	if err := validateBody(body); err != nil {
		return Result{}, err
	}
	if err := validateRegisterStructure(source); err != nil {
		return Result{}, err
	}

	newline := []byte("\n")
	if bytes.Contains(source, []byte("\r\n")) {
		if bytes.Contains(bytes.ReplaceAll(source, []byte("\r\n"), nil), []byte("\n")) {
			return Result{}, errors.New("mixed line endings in decision register")
		}
		newline = []byte("\r\n")
	}
	hadFinalNewline := bytes.HasSuffix(source, newline)
	normalized := bytes.ReplaceAll(source, []byte("\r\n"), []byte("\n"))
	lines := strings.Split(string(normalized), "\n")

	sectionStart, sectionEnd := -1, -1
	maxID := 0
	seenIDs := make(map[string]bool)
	currentSection := -1
	for i, line := range lines {
		if strings.HasPrefix(line, "## ") && !strings.HasPrefix(line, "### ") {
			currentSection = i
		}
		match := headingPattern.FindStringSubmatch(line)
		if strings.HasPrefix(line, "### ") && match == nil {
			return Result{}, fmt.Errorf("malformed decision heading on line %d", i+1)
		}
		if match == nil {
			continue
		}
		value, err := strconv.Atoi(match[2])
		if err != nil || value < 1 {
			return Result{}, fmt.Errorf("invalid decision ID on line %d", i+1)
		}
		key := fmt.Sprintf("%s-%d", match[1], value)
		if seenIDs[key] {
			return Result{}, fmt.Errorf("duplicate decision ID %s-%s", match[1], match[2])
		}
		seenIDs[key] = true
		if match[1] != prefix {
			continue
		}
		if value > maxID {
			maxID = value
		}
		if sectionStart == -1 {
			sectionStart = currentSection
		} else if sectionStart != currentSection {
			return Result{}, fmt.Errorf("decision prefix %s appears in multiple sections", prefix)
		}
	}
	if sectionStart < 0 || maxID == 0 {
		return Result{}, fmt.Errorf("decision prefix %s has no measured section", prefix)
	}
	for i := sectionStart + 1; i < len(lines); i++ {
		if strings.HasPrefix(lines[i], "## ") && !strings.HasPrefix(lines[i], "### ") {
			sectionEnd = i
			break
		}
	}
	if sectionEnd < 0 {
		sectionEnd = len(lines)
	}

	insertAt := sectionEnd
	for insertAt > sectionStart+1 && lines[insertAt-1] == "" {
		insertAt--
	}
	nextID := fmt.Sprintf("%s-%03d", prefix, maxID+1)
	block := []string{"", "### " + nextID + " — " + title}
	block = append(block, strings.Split(string(bytes.TrimSuffix(body, []byte("\n"))), "\n")...)
	block = append(block, "")
	lines = append(lines[:insertAt], append(block, lines[sectionEnd:]...)...)
	rendered := []byte(strings.Join(lines, "\n"))
	if bytes.Equal(newline, []byte("\r\n")) {
		rendered = bytes.ReplaceAll(rendered, []byte("\n"), newline)
	}
	if !hadFinalNewline {
		rendered = bytes.TrimSuffix(rendered, newline)
	}
	if err := validateRegisterStructure(rendered); err != nil {
		return Result{}, fmt.Errorf("rendered decision register validation: %w", err)
	}
	return Result{Data: rendered, ID: nextID, Changed: !bytes.Equal(source, rendered), Line: insertAt + 2}, nil
}

func validateRegisterStructure(source []byte) error {
	if len(source) == 0 || len(source) > MaxRegister {
		return fmt.Errorf("decision register must contain 1-%d bytes", MaxRegister)
	}
	normalized := bytes.ReplaceAll(source, []byte("\r\n"), []byte("\n"))
	if bytes.Contains(normalized, []byte("\r")) {
		return errors.New("decision register contains unsupported carriage returns")
	}
	seen := make(map[string]bool)
	insideSection := false
	for i, line := range strings.Split(string(normalized), "\n") {
		if strings.HasPrefix(line, "## ") && !strings.HasPrefix(line, "### ") {
			insideSection = true
		}
		if !strings.HasPrefix(line, "### ") {
			continue
		}
		match := headingPattern.FindStringSubmatch(line)
		if match == nil {
			return fmt.Errorf("malformed decision heading on line %d", i+1)
		}
		value, err := strconv.Atoi(match[2])
		if err != nil || value < 1 {
			return fmt.Errorf("invalid decision ID on line %d", i+1)
		}
		if !insideSection {
			return fmt.Errorf("decision heading outside a second-level section on line %d", i+1)
		}
		key := fmt.Sprintf("%s-%d", match[1], value)
		if seen[key] {
			return fmt.Errorf("duplicate decision ID %s-%s", match[1], match[2])
		}
		seen[key] = true
	}
	return nil
}

func validateTitle(title string) error {
	if title == "" || len(title) > MaxTitle {
		return fmt.Errorf("decision title must contain 1-%d bytes", MaxTitle)
	}
	if strings.TrimSpace(title) != title || strings.ContainsAny(title, "\r\n\x00") {
		return errors.New("decision title must be one trimmed line without control characters")
	}
	return nil
}

func validateBody(body []byte) error {
	if len(body) == 0 || len(body) > MaxBody {
		return fmt.Errorf("decision body must contain 1-%d bytes", MaxBody)
	}
	if bytes.ContainsAny(body, "\x00\r") {
		return errors.New("decision body contains a forbidden control character or carriage return")
	}
	trimmed := bytes.TrimSuffix(body, []byte("\n"))
	if len(trimmed) == 0 || len(bytes.TrimSpace(trimmed)) == 0 {
		return errors.New("decision body must contain non-whitespace text")
	}
	for _, line := range strings.Split(string(trimmed), "\n") {
		if strings.HasPrefix(line, "## ") || strings.HasPrefix(line, "### ") {
			return errors.New("decision body must not introduce register headings")
		}
	}
	return nil
}

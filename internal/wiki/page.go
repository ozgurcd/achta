package wiki

import (
	"bytes"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

const MaxPageSize int64 = 16 << 20

var (
	datePattern    = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
	repoPattern    = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)
	shaPattern     = regexp.MustCompile(`^[0-9a-f]{40}$`)
	verifiedPrefix = regexp.MustCompile(`^([A-Za-z0-9][A-Za-z0-9._-]*) @ ([0-9a-f]{7,40})(.*)$`)
)

type PinResult struct {
	Data       []byte
	Repository string
	SHA        string
	Verified   string
	Changed    bool
}

type parsedPage struct {
	lines       [][]byte
	newline     []byte
	fields      map[string]int
	derivedHead int
}

func RenderPin(source []byte, repository, fullSHA, verified, branch string) (PinResult, error) {
	if !repoPattern.MatchString(repository) {
		return PinResult{}, errors.New("invalid repository name")
	}
	if !shaPattern.MatchString(fullSHA) {
		return PinResult{}, errors.New("SHA must be a full lowercase 40-character Git SHA")
	}
	if !datePattern.MatchString(verified) {
		return PinResult{}, errors.New("verified date must be exactly YYYY-MM-DD")
	}
	if _, err := time.Parse("2006-01-02", verified); err != nil {
		return PinResult{}, errors.New("verified date is not a calendar date")
	}
	page, err := parsePage(source, repository)
	if err != nil {
		return PinResult{}, err
	}
	page.lines[page.fields["verified"]] = []byte("verified: " + verified)
	lineIndex := page.fields["verified_against"]
	value := strings.TrimPrefix(string(page.lines[lineIndex]), "verified_against: ")
	match := verifiedPrefix.FindStringSubmatch(value)
	if match == nil || match[1] != repository {
		return PinResult{}, errors.New("verified_against repository identity mismatch")
	}
	page.lines[lineIndex] = []byte("verified_against: " + repository + " @ " + fullSHA + match[3])
	headLine := string(page.lines[page.derivedHead])
	parts := strings.Split(headLine, "|")
	if len(parts) != 4 || strings.TrimSpace(parts[1]) != "Repo HEAD" {
		return PinResult{}, errors.New("malformed Repo HEAD derived row")
	}
	valueSuffix := ""
	if open := strings.Index(parts[2], "("); open >= 0 {
		valueSuffix = " " + strings.TrimSpace(parts[2][open:])
	} else if branch != "" {
		valueSuffix = " (" + branch + ")"
	}
	page.lines[page.derivedHead] = []byte("| Repo HEAD | " + fullSHA[:7] + valueSuffix + " |")
	rendered := bytes.Join(page.lines, page.newline)
	if _, err := parsePage(rendered, repository); err != nil {
		return PinResult{}, fmt.Errorf("rendered page validation: %w", err)
	}
	return PinResult{Data: rendered, Repository: repository, SHA: fullSHA, Verified: verified, Changed: !bytes.Equal(source, rendered)}, nil
}

func parsePage(source []byte, repository string) (parsedPage, error) {
	if len(source) == 0 {
		return parsedPage{}, errors.New("empty page")
	}
	newline := []byte("\n")
	if bytes.Contains(source, []byte("\r\n")) {
		newline = []byte("\r\n")
		withoutCRLF := bytes.ReplaceAll(source, []byte("\r\n"), nil)
		if bytes.Contains(withoutCRLF, []byte("\n")) {
			return parsedPage{}, errors.New("mixed line endings")
		}
	}
	lines := bytes.Split(source, newline)
	if len(lines) < 4 || string(lines[0]) != "---" {
		return parsedPage{}, errors.New("missing frontmatter opener")
	}
	end := -1
	fields := make(map[string]int)
	for i := 1; i < len(lines); i++ {
		line := string(lines[i])
		if line == "---" {
			end = i
			break
		}
		colon := strings.IndexByte(line, ':')
		if colon <= 0 {
			return parsedPage{}, fmt.Errorf("malformed frontmatter line %d", i+1)
		}
		key := line[:colon]
		if _, exists := fields[key]; exists {
			return parsedPage{}, fmt.Errorf("duplicate frontmatter field %q", key)
		}
		fields[key] = i
	}
	if end < 0 {
		return parsedPage{}, errors.New("missing frontmatter closer")
	}
	for _, field := range []string{"title", "category", "verified", "verified_against"} {
		if _, ok := fields[field]; !ok {
			return parsedPage{}, fmt.Errorf("missing frontmatter field %q", field)
		}
	}
	if strings.TrimSpace(strings.TrimPrefix(string(lines[fields["title"]]), "title:")) != repository {
		return parsedPage{}, errors.New("page title does not match repository")
	}
	if strings.TrimSpace(strings.TrimPrefix(string(lines[fields["category"]]), "category:")) != "repo" {
		return parsedPage{}, errors.New("page category must be repo")
	}
	begin := "<!-- BEGIN DERIVED: " + repository + " -->"
	beginCount, endCount, headIndex := 0, 0, -1
	inDerived := false
	for i := end + 1; i < len(lines); i++ {
		line := string(lines[i])
		switch line {
		case begin:
			beginCount++
			if inDerived {
				return parsedPage{}, errors.New("nested derived marker")
			}
			inDerived = true
		case "<!-- END DERIVED -->":
			endCount++
			if !inDerived {
				return parsedPage{}, errors.New("derived end marker without start")
			}
			inDerived = false
		default:
			if inDerived && strings.HasPrefix(line, "| Repo HEAD |") {
				if headIndex >= 0 {
					return parsedPage{}, errors.New("duplicate Repo HEAD row")
				}
				headIndex = i
			}
		}
	}
	if beginCount != 1 || endCount != 1 || inDerived {
		return parsedPage{}, errors.New("missing, duplicate, or mismatched derived markers")
	}
	if headIndex < 0 {
		return parsedPage{}, errors.New("missing Repo HEAD row in derived block")
	}
	return parsedPage{lines: lines, newline: newline, fields: fields, derivedHead: headIndex}, nil
}

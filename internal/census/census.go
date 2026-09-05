// Package census recounts a markdown ledger table against itself and against
// the filesystem. Measured cause (identuum, 2026-09-05): a row added to a
// tools ledger with the totals untouched passed every gate the workspace had.
package census

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// Schema is the stable machine interface for ledger censuses.
const Schema = "achta.ledger-census.v1"

// MaxLedger bounds the ledger size this package will read.
const MaxLedger = 8 << 20

// Retired is the one mark with disk semantics of its own: the subject must be gone.
const Retired = "RETIRED"

// Source is one place on disk the ledger censuses. Kind is "files" (every
// regular file directly under Path has a row; size = line count) or "dirs"
// (every immediate subdirectory has a row; size = summed line count of files
// with Ext under it, recursive). Label prefixes the totals rows: "" means
// `KEEP` / `total`; "GO" means `GO-KEEP` / `GO-total`. Rows are assigned to a
// files source when their subject ends with a file name that exists or could
// exist there (its extension is one the source counts), and to a dirs source
// when the subject starts with the source directory's base name and a slash.
type Source struct {
	Label string `json:"label"`
	Path  string `json:"path"`
	Kind  string `json:"kind"`
	Ext   string `json:"ext,omitempty"`
}

// Violation names one disagreement.
type Violation struct {
	Line int    `json:"line"`
	Rule string `json:"rule"`
	Text string `json:"text"`
}

// SourceSummary reports what one source contributed.
type SourceSummary struct {
	Label  string         `json:"label"`
	Path   string         `json:"path"`
	Kind   string         `json:"kind"`
	Rows   int            `json:"rows"`
	OnDisk int            `json:"on_disk"`
	Marks  map[string]int `json:"marks"`
}

// Result is the achta.ledger-census.v1 document.
type Result struct {
	SchemaVersion string          `json:"schema_version"`
	Status        string          `json:"status"`
	Rows          int             `json:"rows"`
	Sources       []SourceSummary `json:"sources"`
	Violations    []Violation     `json:"violations"`
}

type row struct {
	line    int
	number  int
	subject string
	size    int
	mark    string
	source  int
}

var (
	intCell   = regexp.MustCompile(`^\d+$`)
	labelWord = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_-]*`)
)

// Check recounts the table in data against the sources. An error is a
// cannot-evaluate condition (no sources, an unreadable source, no rows).
func Check(data []byte, sources []Source) (Result, error) {
	result := Result{SchemaVersion: Schema, Status: "pass", Sources: []SourceSummary{}, Violations: []Violation{}}
	if len(sources) == 0 {
		return result, errors.New("at least one source is required")
	}
	if len(data) > MaxLedger {
		return result, errors.New("ledger exceeds the size limit")
	}
	for i, s := range sources {
		if s.Kind != "files" && s.Kind != "dirs" {
			return result, fmt.Errorf("source %d: kind must be files or dirs", i+1)
		}
		if s.Kind == "dirs" && s.Ext == "" {
			return result, fmt.Errorf("source %d: a dirs source needs an extension to count", i+1)
		}
		if s.Label != "" && !labelWord.MatchString(s.Label) {
			return result, fmt.Errorf("source %d: invalid label %q", i+1, s.Label)
		}
		info, err := os.Stat(s.Path)
		if err != nil || !info.IsDir() {
			return result, fmt.Errorf("source %d: %q is not a readable directory", i+1, s.Path)
		}
	}
	lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")

	// rows: | # | subject | size | ... | mark |  (4+ cells; # and size integers)
	var rows []row
	totals := map[string][3]int{} // key -> count, size, line
	for i, l := range lines {
		if !strings.HasPrefix(l, "|") {
			continue
		}
		cells := splitRow(l)
		if len(cells) >= 4 && intCell.MatchString(cells[0]) && intCell.MatchString(cells[2]) {
			num, _ := strconv.Atoi(cells[0])
			size, _ := strconv.Atoi(cells[2])
			markWords := strings.Fields(cells[len(cells)-1])
			mark := ""
			if len(markWords) > 0 {
				mark = markWords[0]
			}
			src := assign(cells[1], sources)
			if src < 0 {
				result.Violations = append(result.Violations, Violation{Line: i + 1, Rule: "unassigned-row", Text: fmt.Sprintf("row %d names no subject any source censuses: %q", num, strings.TrimSpace(cells[1]))})
				continue
			}
			if mark == "" || !labelWord.MatchString(mark) {
				result.Violations = append(result.Violations, Violation{Line: i + 1, Rule: "mark", Text: fmt.Sprintf("row %d has no mark word", num)})
				continue
			}
			rows = append(rows, row{line: i + 1, number: num, subject: subjectName(cells[1], sources[src]), size: size, mark: strings.ToUpper(mark), source: src})
			continue
		}
		if len(cells) == 3 && !intCell.MatchString(cells[0]) && intCell.MatchString(cells[1]) && intCell.MatchString(cells[2]) {
			key := strings.ToUpper(labelWord.FindString(strings.TrimSpace(cells[0])))
			if key == "" {
				continue
			}
			if _, dup := totals[key]; dup {
				result.Violations = append(result.Violations, Violation{Line: i + 1, Rule: "totals", Text: fmt.Sprintf("totals row %s is repeated", key)})
				continue
			}
			c, _ := strconv.Atoi(cells[1])
			s, _ := strconv.Atoi(cells[2])
			totals[key] = [3]int{c, s, i + 1}
		}
	}
	if len(rows) == 0 {
		return result, errors.New("no ledger rows found")
	}
	result.Rows = len(rows)

	seenNum := map[int]int{}
	seenSubject := map[string]int{}
	for _, r := range rows {
		if first, ok := seenNum[r.number]; ok {
			result.Violations = append(result.Violations, Violation{Line: r.line, Rule: "duplicate-number", Text: fmt.Sprintf("row number %d is repeated (first at line %d)", r.number, first)})
		} else {
			seenNum[r.number] = r.line
		}
		key := fmt.Sprintf("%d:%s", r.source, r.subject)
		if first, ok := seenSubject[key]; ok {
			result.Violations = append(result.Violations, Violation{Line: r.line, Rule: "duplicate-subject", Text: fmt.Sprintf("%s has two rows (first at line %d)", r.subject, first)})
		} else {
			seenSubject[key] = r.line
		}
	}

	usedTotals := map[string]bool{}
	for si, s := range sources {
		prefix := ""
		if s.Label != "" {
			prefix = strings.ToUpper(s.Label) + "-"
		}
		summary := SourceSummary{Label: s.Label, Path: s.Path, Kind: s.Kind, Marks: map[string]int{}}
		counts := map[string][2]int{}
		var mine []row
		for _, r := range rows {
			if r.source != si {
				continue
			}
			mine = append(mine, r)
			c := counts[r.mark]
			c[0]++
			c[1] += r.size
			counts[r.mark] = c
			summary.Marks[r.mark]++
		}
		summary.Rows = len(mine)
		marks := make([]string, 0, len(counts))
		for m := range counts {
			marks = append(marks, m)
		}
		sort.Strings(marks)
		for _, m := range marks {
			key := prefix + m
			usedTotals[key] = true
			t, ok := totals[key]
			if !ok {
				result.Violations = append(result.Violations, Violation{Line: 0, Rule: "totals", Text: fmt.Sprintf("Totals have no %s row, but the table has %d %s row(s) totalling %d", key, counts[m][0], m, counts[m][1])})
				continue
			}
			if t[0] != counts[m][0] || t[1] != counts[m][1] {
				result.Violations = append(result.Violations, Violation{Line: t[2], Rule: "totals", Text: fmt.Sprintf("Totals state %s %d / %d, but the rows count %d / %d", key, t[0], t[1], counts[m][0], counts[m][1])})
			}
		}
		tkey := prefix + "TOTAL"
		usedTotals[tkey] = true
		if t, ok := totals[tkey]; !ok {
			result.Violations = append(result.Violations, Violation{Line: 0, Rule: "totals", Text: fmt.Sprintf("Totals have no %s row for %d row(s)", tkey, len(mine))})
		} else {
			sum := 0
			for _, r := range mine {
				sum += r.size
			}
			if t[0] != len(mine) || t[1] != sum {
				result.Violations = append(result.Violations, Violation{Line: t[2], Rule: "totals", Text: fmt.Sprintf("Totals state %s %d / %d, but the rows count %d / %d", tkey, t[0], t[1], len(mine), sum)})
			}
		}
		// totals rows for marks with zero rows are fine only when they say zero
		for key, t := range totals {
			if strings.HasPrefix(key, prefix) && (prefix != "" || !strings.Contains(key, "-")) && !usedTotals[key] && (t[0] != 0 || t[1] != 0) {
				result.Violations = append(result.Violations, Violation{Line: t[2], Rule: "totals", Text: fmt.Sprintf("Totals state %s %d / %d, but no row carries that mark", key, t[0], t[1])})
			}
		}

		onDisk, sizes, err := scan(s)
		if err != nil {
			return result, err
		}
		summary.OnDisk = len(onDisk)
		rowed := map[string]bool{}
		for _, r := range mine {
			rowed[r.subject] = true
			_, exists := onDisk[r.subject]
			if r.mark == Retired {
				if exists {
					result.Violations = append(result.Violations, Violation{Line: r.line, Rule: "retired-present", Text: fmt.Sprintf("row %d says %s is %s, but it still exists under %s", r.number, r.subject, Retired, s.Path)})
				}
				continue
			}
			if !exists {
				result.Violations = append(result.Violations, Violation{Line: r.line, Rule: "missing-on-disk", Text: fmt.Sprintf("row %d (%s, %s) names a subject that is not under %s", r.number, r.subject, r.mark, s.Path)})
				continue
			}
			if sizes[r.subject] != r.size {
				result.Violations = append(result.Violations, Violation{Line: r.line, Rule: "size", Text: fmt.Sprintf("row %d states %s has %d lines; on disk it has %d", r.number, r.subject, r.size, sizes[r.subject])})
			}
		}
		names := make([]string, 0, len(onDisk))
		for n := range onDisk {
			names = append(names, n)
		}
		sort.Strings(names)
		for _, n := range names {
			if !rowed[n] {
				result.Violations = append(result.Violations, Violation{Line: 0, Rule: "unrowed", Text: fmt.Sprintf("%s under %s has no row", n, s.Path)})
			}
		}
		result.Sources = append(result.Sources, summary)
	}
	sort.Slice(result.Violations, func(i, j int) bool {
		if result.Violations[i].Line != result.Violations[j].Line {
			return result.Violations[i].Line < result.Violations[j].Line
		}
		return result.Violations[i].Text < result.Violations[j].Text
	})
	if len(result.Violations) > 0 {
		result.Status = "fail"
	}
	return result, nil
}

func splitRow(l string) []string {
	trimmed := strings.TrimSpace(l)
	trimmed = strings.TrimPrefix(trimmed, "|")
	trimmed = strings.TrimSuffix(trimmed, "|")
	parts := strings.Split(trimmed, "|")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

// assign picks the first source whose naming shape the subject cell matches.
func assign(cell string, sources []Source) int {
	for token := range strings.FieldsSeq(cell) {
		for i, s := range sources {
			switch s.Kind {
			case "files":
				if s.Ext == "" || strings.HasSuffix(token, s.Ext) {
					if !strings.Contains(token, "/") && strings.Contains(token, ".") {
						return i
					}
				}
			case "dirs":
				if strings.HasPrefix(token, filepath.Base(s.Path)+"/") {
					return i
				}
			}
		}
	}
	return -1
}

func subjectName(cell string, s Source) string {
	for token := range strings.FieldsSeq(cell) {
		switch s.Kind {
		case "files":
			if !strings.Contains(token, "/") && strings.Contains(token, ".") && (s.Ext == "" || strings.HasSuffix(token, s.Ext)) {
				return token
			}
		case "dirs":
			if rest, ok := strings.CutPrefix(token, filepath.Base(s.Path)+"/"); ok {
				return strings.TrimSuffix(rest, "/")
			}
		}
	}
	return ""
}

// scan lists the source's subjects on disk with their sizes.
func scan(s Source) (map[string]bool, map[string]int, error) {
	entries, err := os.ReadDir(s.Path)
	if err != nil {
		return nil, nil, fmt.Errorf("read %s: %w", s.Path, err)
	}
	onDisk := map[string]bool{}
	sizes := map[string]int{}
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		switch s.Kind {
		case "files":
			if !e.Type().IsRegular() || (s.Ext != "" && !strings.HasSuffix(name, s.Ext)) {
				continue
			}
			n, err := countLines(filepath.Join(s.Path, name))
			if err != nil {
				return nil, nil, err
			}
			onDisk[name] = true
			sizes[name] = n
		case "dirs":
			if !e.IsDir() {
				continue
			}
			total := 0
			err := filepath.WalkDir(filepath.Join(s.Path, name), func(path string, d os.DirEntry, walkErr error) error {
				if walkErr != nil {
					return walkErr
				}
				if d.Type().IsRegular() && strings.HasSuffix(d.Name(), s.Ext) {
					n, err := countLines(path)
					if err != nil {
						return err
					}
					total += n
				}
				return nil
			})
			if err != nil {
				return nil, nil, fmt.Errorf("walk %s: %w", name, err)
			}
			onDisk[name] = true
			sizes[name] = total
		}
	}
	return onDisk, sizes, nil
}

func countLines(path string) (int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, fmt.Errorf("read %s: %w", path, err)
	}
	if len(data) == 0 {
		return 0, nil
	}
	n := bytes.Count(data, []byte{'\n'})
	if data[len(data)-1] != '\n' {
		n++
	}
	return n, nil
}

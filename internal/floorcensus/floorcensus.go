// Package floorcensus recounts a fenced completeness census against itself and
// against Rulefloor's covers map (rulefloor.covers.v1). Measured cause
// (identuum, 2026-08..09): the floor-completeness census drifted from the
// rule ledger three times in one day, and a shell gate grew eight violation
// classes to hold it; achta takes the seven classes that need only the census
// and the covers document, and REFUSES the one that needs RULE-FLOOR.md's
// columns (armed state) — that is Rulefloor's format, outside this boundary.
//
// Nothing here is identuum's vocabulary: the bucket tokens, the covered
// bucket, the fence heading, the frozen marker, the stated-count patterns and
// the allowlist prefix are all Options the caller names.
package floorcensus

import (
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// Schema is the stable machine interface for floor censuses.
const Schema = "achta.floor-census.v1"

// MaxCensus bounds the census size this package will read.
const MaxCensus = 8 << 20

// Options name the census's own vocabulary. Buckets, Covered and FenceHeading
// are required. Marker (a trailing rune on a citation, e.g. "~") enables the
// frozen/plain split. Each Count* pattern is a regexp with exactly one (\d+)
// capture; when empty, that stated count is not checked. AllowPrefix is the
// line prefix of allowlisted covers absences; when empty, no absence is
// allowlisted.
type Options struct {
	Buckets      []string
	Covered      string
	FenceHeading string
	Marker       string
	CountSum     string
	CountFrozen  string
	CountPlain   string
	AllowPrefix  string
}

// Violation names one disagreement.
type Violation struct {
	Line  int    `json:"line"`
	Class int    `json:"class"`
	Text  string `json:"text"`
}

// CoversStats summarizes the covers document against the census.
type CoversStats struct {
	Rules       int `json:"rules"`
	Mapped      int `json:"mapped"`
	Absences    int `json:"absences"`
	Allowlisted int `json:"allowlisted"`
	New         int `json:"new"`
}

// Result is the achta.floor-census.v1 document.
type Result struct {
	SchemaVersion       string         `json:"schema_version"`
	Status              string         `json:"status"`
	Rows                int            `json:"rows"`
	Buckets             map[string]int `json:"buckets"`
	Frozen              int            `json:"frozen"`
	Plain               int            `json:"plain"`
	Covers              CoversStats    `json:"covers"`
	RulefloorExecutable string         `json:"rulefloor_executable,omitempty"`
	Violations          []Violation    `json:"violations"`
	Refused             []string       `json:"refused"`
}

// RefusedArmed is the one half of the old gate achta will not take.
const RefusedArmed = "whether a cited rule is ARMED: that needs RULE-FLOOR.md column parsing, Rulefloor's format, outside this boundary"

// ruleID is Rulefloor's rule identifier grammar, as its ledger rows carry it.
var ruleID = regexp.MustCompile(`^[A-Z][A-Z0-9-]*[0-9]+$`)

var bucketToken = regexp.MustCompile(`^[A-Z][A-Z0-9-]*$`)

// Check evaluates the census in data against covers (rule ID -> qualified
// file:Symbol entries). An error is a cannot-evaluate condition.
func Check(data []byte, covers map[string][]string, opts Options) (Result, error) {
	result := Result{SchemaVersion: Schema, Status: "pass", Buckets: map[string]int{}, Violations: []Violation{}, Refused: []string{RefusedArmed}}
	if len(data) > MaxCensus {
		return result, errors.New("census exceeds the size limit")
	}
	if len(opts.Buckets) == 0 {
		return result, errors.New("at least one bucket token is required")
	}
	seenBucket := map[string]bool{}
	for _, b := range opts.Buckets {
		if !bucketToken.MatchString(b) {
			return result, fmt.Errorf("invalid bucket token %q", b)
		}
		if seenBucket[b] {
			return result, fmt.Errorf("bucket token %q repeated", b)
		}
		seenBucket[b] = true
		result.Buckets[b] = 0
	}
	if !seenBucket[opts.Covered] {
		return result, fmt.Errorf("covered bucket %q is not one of the buckets", opts.Covered)
	}
	if strings.TrimSpace(opts.FenceHeading) == "" {
		return result, errors.New("a fence heading is required")
	}
	if len(opts.Marker) > 1 {
		return result, errors.New("the marker must be a single character")
	}
	if opts.Marker == "" && (opts.CountFrozen != "" || opts.CountPlain != "") {
		return result, errors.New("frozen and plain counts need a marker")
	}
	counts := map[string]*regexp.Regexp{}
	for name, pattern := range map[string]string{"sum": opts.CountSum, "frozen": opts.CountFrozen, "plain": opts.CountPlain} {
		if pattern == "" {
			continue
		}
		re, err := regexp.Compile(pattern)
		if err != nil || re.NumSubexp() != 1 {
			return result, fmt.Errorf("count pattern %s must be a valid regexp with exactly one capture group", name)
		}
		counts[name] = re
	}
	if covers == nil {
		return result, errors.New("a covers document is required")
	}

	lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
	rowRe := regexp.MustCompile(`^(` + strings.Join(escapeAll(opts.Buckets), "|") + `)\s+(\d+)\s+(\S+)\s+(\S+)\s*(.*)$`)
	leadRe := regexp.MustCompile(`^(` + strings.Join(escapeAll(opts.Buckets), "|") + `)\b`)

	// locate the fenced table under the heading
	start, end := -1, -1
	inHeading := false
	for i, l := range lines {
		if strings.HasPrefix(l, opts.FenceHeading) {
			inHeading = true
			continue
		}
		if inHeading && strings.TrimSpace(l) == "```" {
			if start < 0 {
				start = i
			} else {
				end = i
				break
			}
		}
	}
	if start < 0 || end < 0 {
		return result, fmt.Errorf("no fenced table under %q", opts.FenceHeading)
	}

	// covers: rule set, identities
	rules := make([]string, 0, len(covers))
	for r := range covers {
		rules = append(rules, r)
	}
	sort.Strings(rules)
	result.Covers.Rules = len(rules)
	knownRule := map[string]bool{}
	for _, r := range rules {
		knownRule[r] = true
		if len(covers[r]) > 0 {
			result.Covers.Mapped++
		}
	}

	type identity struct{ file, symbol string }
	rowIDs := map[identity]bool{}
	citedBy := map[identity]map[string]bool{}
	tokenOf := func(t string) (string, bool, bool) { // rule id, frozen marker present, is a citation
		frozen := false
		if opts.Marker != "" && strings.HasSuffix(t, opts.Marker) {
			t = strings.TrimSuffix(t, opts.Marker)
			frozen = true
		}
		return t, frozen, ruleID.MatchString(t)
	}
	for i := start + 1; i < end; i++ {
		ln := i + 1
		l := lines[i]
		m := rowRe.FindStringSubmatch(l)
		if m == nil {
			if strings.TrimSpace(l) != "" {
				result.Violations = append(result.Violations, Violation{Line: ln, Class: 0, Text: fmt.Sprintf("unparseable census row: %q", clip(l))})
			}
			continue
		}
		bucket, sym, fileline, reason := m[1], m[3], m[4], m[5]
		result.Rows++
		result.Buckets[bucket]++
		id := identity{file: rsplitFile(fileline), symbol: sym}
		rowIDs[id] = true
		var tokens []string
		frozenRow := false
		for t := range strings.FieldsSeq(reason) {
			rid, frozen, ok := tokenOf(t)
			if !ok {
				continue
			}
			tokens = append(tokens, rid)
			if frozen {
				frozenRow = true
			}
			if citedBy[id] == nil {
				citedBy[id] = map[string]bool{}
			}
			citedBy[id][rid] = true
		}
		if bucket == opts.Covered {
			if frozenRow {
				result.Frozen++
			} else {
				result.Plain++
			}
			if len(tokens) == 0 { // (3)
				result.Violations = append(result.Violations, Violation{Line: ln, Class: 3, Text: fmt.Sprintf("%s row %s has no citation", opts.Covered, sym)})
			}
			for _, rid := range tokens { // (2), the half achta can answer
				if !knownRule[rid] {
					result.Violations = append(result.Violations, Violation{Line: ln, Class: 2, Text: fmt.Sprintf("%s row %s cites %s, which is not a rule the covers document knows", opts.Covered, sym, rid)})
				}
			}
		} else {
			for _, rid := range tokens { // (1)
				if knownRule[rid] {
					result.Violations = append(result.Violations, Violation{Line: ln, Class: 1, Text: fmt.Sprintf("%s row %s names rule %s (a covered symbol in an out-of-scope bucket?)", bucket, sym, rid)})
				}
			}
		}
		seen := map[string]bool{}
		for _, rid := range tokens { // (4)
			if seen[rid] {
				result.Violations = append(result.Violations, Violation{Line: ln, Class: 4, Text: fmt.Sprintf("row %s repeats citation %s", sym, rid)})
			}
			seen[rid] = true
		}
	}
	if result.Rows == 0 {
		return result, errors.New("the fenced table has no census rows")
	}

	// (5) bucket-led lines outside the fence; allowlist lines
	allowed := map[string]bool{}
	for i, l := range lines {
		if i > start && i < end {
			continue
		}
		if leadRe.MatchString(l) {
			result.Violations = append(result.Violations, Violation{Line: i + 1, Class: 5, Text: fmt.Sprintf("line outside the fenced table starts with a bucket token: %q", clip(l))})
		}
		if opts.AllowPrefix != "" && strings.HasPrefix(l, opts.AllowPrefix) {
			fields := strings.Fields(strings.TrimPrefix(l, opts.AllowPrefix))
			if len(fields) == 2 {
				allowed[fields[0]+" "+fields[1]] = true
			}
		}
	}

	// (7) and (8)
	var absences []string
	for _, r := range rules {
		for _, entry := range covers[r] {
			if !strings.Contains(entry, ":") {
				result.Violations = append(result.Violations, Violation{Line: 0, Class: 7, Text: fmt.Sprintf("unqualified covers entry: covers(%s) lists bare name %q", r, entry)})
				continue
			}
			i := strings.LastIndex(entry, ":")
			id := identity{file: entry[:i], symbol: entry[i+1:]}
			if rowIDs[id] {
				if !citedBy[id][r] {
					result.Violations = append(result.Violations, Violation{Line: 0, Class: 7, Text: fmt.Sprintf("covers(%s) proves a mutation at %s, which has census row(s) at that identity, but none cites %s", r, entry, r)})
				}
			} else {
				absences = append(absences, r+" "+entry)
			}
		}
	}
	sort.Strings(absences)
	result.Covers.Absences = len(absences)
	for _, a := range absences {
		if allowed[a] {
			result.Covers.Allowlisted++
			continue
		}
		result.Covers.New++
		result.Violations = append(result.Violations, Violation{Line: 0, Class: 8, Text: fmt.Sprintf("covers absence not allowlisted: %s has no census row at that identity", a)})
	}

	// (6) stated counts
	for _, b := range opts.Buckets {
		re := regexp.MustCompile(`^- ` + regexp.QuoteMeta(b) + `: (\d+)$`)
		for i, l := range lines {
			if m := re.FindStringSubmatch(l); m != nil {
				if n, _ := strconv.Atoi(m[1]); n != result.Buckets[b] {
					result.Violations = append(result.Violations, Violation{Line: i + 1, Class: 6, Text: fmt.Sprintf("stated %s: %d but the table counts %d", b, n, result.Buckets[b])})
				}
			}
		}
	}
	stated := func(name string, actual int) {
		re, ok := counts[name]
		if !ok {
			return
		}
		for i, l := range lines {
			if m := re.FindStringSubmatch(l); m != nil {
				if n, _ := strconv.Atoi(m[1]); n != actual {
					result.Violations = append(result.Violations, Violation{Line: i + 1, Class: 6, Text: fmt.Sprintf("stated %s %d but the table counts %d", name, n, actual)})
				}
			}
		}
	}
	stated("sum", result.Rows)
	stated("frozen", result.Frozen)
	stated("plain", result.Plain)

	sort.SliceStable(result.Violations, func(i, j int) bool { return result.Violations[i].Line < result.Violations[j].Line })
	if len(result.Violations) > 0 {
		result.Status = "fail"
	}
	return result, nil
}

func escapeAll(items []string) []string {
	out := make([]string, len(items))
	for i, s := range items {
		out[i] = regexp.QuoteMeta(s)
	}
	return out
}

func rsplitFile(fileline string) string {
	if i := strings.LastIndex(fileline, ":"); i > 0 {
		return fileline[:i]
	}
	return fileline
}

func clip(s string) string {
	if len(s) > 60 {
		return s[:60]
	}
	return s
}

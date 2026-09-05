package floorcensus

import (
	"strings"
	"testing"
)

var fixtureLines = []string{
	"# fixture census",
	"",
	"## The frozen `~` set (1 rows in one class)",
	"",
	"## Totals (sum = 4 = list length 4)",
	"",
	"- COVERED: 2",
	"- OOS-L: 0",
	"- OOS-O: 0",
	"- OOS-P: 1",
	"- OOS-T: 1",
	"",
	"named; the 1 plain rows are the first two arms combined.",
	"",
	"- ALLOW-COVERS-ABSENCE: GOOD-RULE-1 internal/x.go:Orphan",
	"",
	"## Full list (4 rows)",
	"",
	"```",
	"COVERED           2  GoodSym         internal/a.go:1  GOOD-RULE-1",
	"COVERED           1  TildeSym        internal/b.go:2  GOOD-RULE-1~",
	"OOS-P             1  PlumbSym        internal/c.go:3  P",
	"OOS-T             1  ToolSym         tools/d.go:4  T",
	"```",
	"",
}

func fixture(edits ...func(string) string) []byte {
	text := strings.Join(fixtureLines, "\n")
	for _, e := range edits {
		text = e(text)
	}
	return []byte(text)
}

func swap(from, to string) func(string) string {
	return func(s string) string { return strings.Replace(s, from, to, 1) }
}

func baseCovers() map[string][]string {
	return map[string][]string{
		"GOOD-RULE-1":   {"internal/a.go:GoodSym", "internal/b.go:TildeSym", "internal/x.go:Orphan"},
		"SLEEPY-RULE-1": {},
	}
}

func options() Options {
	return Options{
		Buckets:      []string{"COVERED", "OOS-L", "OOS-O", "OOS-P", "OOS-T"},
		Covered:      "COVERED",
		FenceHeading: "## Full list",
		Marker:       "~",
		CountSum:     `^## Totals \(sum = (\d+)`,
		CountFrozen:  "^## The frozen `~` set \\((\\d+) rows",
		CountPlain:   `the (\d+) plain rows`,
		AllowPrefix:  "- ALLOW-COVERS-ABSENCE:",
	}
}

func classes(r Result) string {
	var out []string
	for _, v := range r.Violations {
		out = append(out, strings.TrimSpace(strings.Repeat(" ", 0)+itoa(v.Class)))
	}
	return strings.Join(out, ",")
}

func itoa(n int) string { return string(rune('0' + n)) }

// RULE: FLOOR-CENSUS-1
func TestCheckRecountsCensusAgainstItselfAndCovers(t *testing.T) {
	clean, err := Check(fixture(), baseCovers(), options())
	if err != nil || clean.Status != "pass" || clean.Rows != 4 || clean.Buckets["COVERED"] != 2 || clean.Frozen != 1 || clean.Plain != 1 ||
		clean.Covers.Rules != 2 || clean.Covers.Mapped != 1 || clean.Covers.Absences != 1 || clean.Covers.Allowlisted != 1 || clean.Covers.New != 0 ||
		len(clean.Refused) != 1 {
		t.Fatalf("clean census: err=%v result=%+v", err, clean)
	}

	cases := []struct {
		name   string
		text   []byte
		covers map[string][]string
		class  string
	}{
		{"(1) out-of-scope row naming a rule", fixture(swap("internal/c.go:3  P", "internal/c.go:3  GOOD-RULE-1")), baseCovers(), "1"},
		{"(2) covered row citing an unknown rule", fixture(swap("internal/a.go:1  GOOD-RULE-1", "internal/a.go:1  NO-SUCH-RULE-1")), baseCovers(), "2"},
		{"(3) covered row with no citation", fixture(swap("internal/a.go:1  GOOD-RULE-1", "internal/a.go:1  ")), baseCovers(), "3"},
		{"(4) repeated citation in one row", fixture(swap("internal/a.go:1  GOOD-RULE-1", "internal/a.go:1  GOOD-RULE-1 GOOD-RULE-1")), baseCovers(), "4"},
		{"(5) bucket-led prose outside the fence", fixture(swap("named; the 1 plain rows", "OOS-P leaked into prose\nnamed; the 1 plain rows")), baseCovers(), "5"},
		{"(6) stated bucket count", fixture(swap("- COVERED: 2", "- COVERED: 7")), baseCovers(), "6"},
		{"(6) stated sum", fixture(swap("sum = 4 =", "sum = 9 =")), baseCovers(), "6"},
		{"(6) stated frozen", fixture(swap("set (1 rows", "set (5 rows")), baseCovers(), "6"},
		{"(6) stated plain", fixture(swap("the 1 plain rows", "the 6 plain rows")), baseCovers(), "6"},
		{"(7) covers pair with rows none of which cite", fixture(), map[string][]string{"GOOD-RULE-1": {"internal/a.go:GoodSym", "internal/b.go:TildeSym", "internal/x.go:Orphan", "internal/c.go:PlumbSym"}}, "7"},
		{"(7) unqualified covers entry", fixture(), map[string][]string{"GOOD-RULE-1": {"internal/a.go:GoodSym", "internal/b.go:TildeSym", "internal/x.go:Orphan", "Bare"}}, "7"},
		{"(8) covers absence not allowlisted", fixture(), map[string][]string{"GOOD-RULE-1": {"internal/a.go:GoodSym", "internal/b.go:TildeSym", "internal/x.go:Orphan", "internal/z.go:Ghost"}}, "8"},
	}
	for _, c := range cases {
		got, err := Check(c.text, c.covers, options())
		if err != nil || got.Status != "fail" || !strings.Contains(classes(got), c.class) {
			t.Fatalf("%s: err=%v status=%q classes=%q violations=%v", c.name, err, got.Status, classes(got), got.Violations)
		}
	}

	// The allowlist keeps an absence silent; a same-file same-name citation satisfies class 7.
	if got, err := Check(fixture(swap("- ALLOW-COVERS-ABSENCE: GOOD-RULE-1 internal/x.go:Orphan", "- ALLOW-COVERS-ABSENCE: GOOD-RULE-1 internal/x.go:Orphan\n- ALLOW-COVERS-ABSENCE: GOOD-RULE-1 internal/z.go:Ghost")),
		map[string][]string{"GOOD-RULE-1": {"internal/a.go:GoodSym", "internal/b.go:TildeSym", "internal/x.go:Orphan", "internal/z.go:Ghost"}}, options()); err != nil || got.Status != "pass" {
		t.Fatalf("allowlisted absence must be silent: err=%v status=%q violations=%v", err, got.Status, got.Violations)
	}

	// Without a marker there is no frozen/plain split and the marker-bearing token is not a citation.
	noMarker := options()
	noMarker.Marker, noMarker.CountFrozen, noMarker.CountPlain = "", "", ""
	if got, err := Check(fixture(), baseCovers(), noMarker); err != nil || got.Frozen != 0 || got.Plain != 2 || !strings.Contains(classes(got), "3") {
		t.Fatalf("no marker: err=%v frozen=%d plain=%d classes=%q", err, got.Frozen, got.Plain, classes(got))
	}
}

func TestCheckCannotEvaluate(t *testing.T) {
	bad := func(name string, mutate func(*Options), text []byte, covers map[string][]string) {
		o := options()
		mutate(&o)
		if _, err := Check(text, covers, o); err == nil {
			t.Fatalf("%s must be an error", name)
		}
	}
	bad("no buckets", func(o *Options) { o.Buckets = nil }, fixture(), baseCovers())
	bad("covered not a bucket", func(o *Options) { o.Covered = "NOPE" }, fixture(), baseCovers())
	bad("no fence heading", func(o *Options) { o.FenceHeading = "" }, fixture(), baseCovers())
	bad("fence heading absent from the page", func(o *Options) { o.FenceHeading = "## Nowhere" }, fixture(), baseCovers())
	bad("two-character marker", func(o *Options) { o.Marker = "~~" }, fixture(), baseCovers())
	bad("count pattern without a capture", func(o *Options) { o.CountSum = `sum = \d+` }, fixture(), baseCovers())
	bad("frozen count without a marker", func(o *Options) { o.Marker = "" }, fixture(), baseCovers())
	bad("nil covers", func(o *Options) {}, fixture(), nil)
	bad("no rows in the fence", func(o *Options) {}, fixture(swap("COVERED           2  GoodSym         internal/a.go:1  GOOD-RULE-1\nCOVERED           1  TildeSym        internal/b.go:2  GOOD-RULE-1~\nOOS-P             1  PlumbSym        internal/c.go:3  P\nOOS-T             1  ToolSym         tools/d.go:4  T\n", "")), baseCovers())
}

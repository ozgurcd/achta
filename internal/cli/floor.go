package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/ozgurcd/achta/internal/floorcensus"
	"github.com/ozgurcd/achta/internal/rulefloorclient"
	"github.com/ozgurcd/achta/internal/safefile"
)

func runFloor(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	rest, err := requireSubcommand(args, "census")
	if err != nil {
		return renderError(stdout, stderr, opts.json, floorcensus.Schema, err)
	}
	set := flagSet("floor census")
	fileValue := set.String("file", "", "census page (inside the workspace)")
	repoValue := set.String("repo", "", "repository whose rulefloor covers are read")
	buckets := set.String("bucket", "", "comma-separated bucket tokens that lead a census row")
	covered := set.String("covered", "", "the bucket whose rows must cite a rule")
	fenceHeading := set.String("fence-heading", "", "heading under which the fenced census table sits")
	marker := set.String("marker", "", "single trailing character on a citation marking the frozen arm")
	countSum := set.String("count-sum", "", "regexp with one capture: the stated total row count")
	countFrozen := set.String("count-frozen", "", "regexp with one capture: the stated frozen count")
	countPlain := set.String("count-plain", "", "regexp with one capture: the stated plain count")
	allowPrefix := set.String("allow-prefix", "", "line prefix of allowlisted covers absences (RULE file:Symbol follows)")
	coversFile := set.String("covers", "", "read a rulefloor.covers.v1 document from this file instead of executing rulefloor")
	rulefloorPath := set.String("rulefloor", "", "Rulefloor executable path")
	jsonMode := set.Bool("json", opts.json, "emit JSON")
	if err := parseFlags(set, rest); err != nil {
		opts.json = *jsonMode
		return renderError(stdout, stderr, opts.json, floorcensus.Schema, err)
	}
	opts.json = *jsonMode
	if *fileValue == "" || *buckets == "" || *covered == "" || *fenceHeading == "" {
		return renderError(stdout, stderr, opts.json, floorcensus.Schema, invalid("--file, --bucket, --covered and --fence-heading are required"))
	}
	if *repoValue == "" && *coversFile == "" {
		return renderError(stdout, stderr, opts.json, floorcensus.Schema, invalid("--repo is required unless --covers names a covers document"))
	}
	ws, err := resolveWorkspace(opts)
	if err != nil {
		return renderError(stdout, stderr, opts.json, floorcensus.Schema, err)
	}
	file, err := confinedPath(ws, *fileValue)
	if err != nil {
		return renderError(stdout, stderr, opts.json, floorcensus.Schema, err)
	}
	snapshot, err := safefile.Read(ws.Root, file, floorcensus.MaxCensus)
	if err != nil {
		return renderError(stdout, stderr, opts.json, floorcensus.Schema, invalid("read census: %v", err))
	}
	var covers rulefloorclient.Covers
	executable := ""
	if *coversFile != "" {
		path, err := confinedPath(ws, *coversFile)
		if err != nil {
			return renderError(stdout, stderr, opts.json, floorcensus.Schema, err)
		}
		raw, err := safefile.Read(ws.Root, path, floorcensus.MaxCensus)
		if err != nil {
			return renderError(stdout, stderr, opts.json, floorcensus.Schema, invalid("read covers: %v", err))
		}
		if err := json.Unmarshal(raw.Data, &covers); err != nil || covers.SchemaVersion != "rulefloor.covers.v1" || covers.Rules == nil {
			return renderError(stdout, stderr, opts.json, floorcensus.Schema, invalid("covers file is not a rulefloor.covers.v1 document"))
		}
	} else {
		repo, err := confinedPath(ws, *repoValue)
		if err != nil {
			return renderError(stdout, stderr, opts.json, floorcensus.Schema, err)
		}
		client, selected, err := (rulefloorclient.Client{Path: *rulefloorPath}).Resolve()
		if err != nil {
			return renderError(stdout, stderr, opts.json, floorcensus.Schema, invalid("Rulefloor executable: %v", err))
		}
		executable = selected
		covers, err = client.Covers(repo)
		if err != nil {
			return renderError(stdout, stderr, opts.json, floorcensus.Schema, invalid("Rulefloor covers using %s: %v", selected, err))
		}
	}
	var bucketList []string
	for b := range strings.SplitSeq(*buckets, ",") {
		if b = strings.TrimSpace(b); b != "" {
			bucketList = append(bucketList, b)
		}
	}
	result, err := floorcensus.Check(snapshot.Data, covers.Rules, floorcensus.Options{
		Buckets: bucketList, Covered: *covered, FenceHeading: *fenceHeading, Marker: *marker,
		CountSum: *countSum, CountFrozen: *countFrozen, CountPlain: *countPlain, AllowPrefix: *allowPrefix,
	})
	if err != nil {
		return renderError(stdout, stderr, opts.json, floorcensus.Schema, invalid("floor census: %v", err))
	}
	result.RulefloorExecutable = executable
	code := 0
	if result.Status != "pass" {
		code = 1
	}
	if opts.json {
		if writeJSON(stdout, stderr, result) != 0 {
			return 2
		}
		return code
	}
	for _, v := range result.Violations {
		where := "census"
		if v.Line > 0 {
			where = fmt.Sprintf("line %d", v.Line)
		}
		fmt.Fprintf(stdout, "floor census: %s class %d — %s\n", where, v.Class, v.Text)
	}
	names := make([]string, 0, len(result.Buckets))
	for b := range result.Buckets {
		names = append(names, b)
	}
	sort.Strings(names)
	parts := make([]string, 0, len(names))
	for _, b := range names {
		parts = append(parts, fmt.Sprintf("%s %d", b, result.Buckets[b]))
	}
	fmt.Fprintf(stdout, "floor census: %s; %d rows (%s); frozen %d, plain %d; covers %d rules (%d mapped), absences %d (allowlisted %d, new %d); %d violation(s); refused: armed state\n",
		result.Status, result.Rows, strings.Join(parts, ", "), result.Frozen, result.Plain,
		result.Covers.Rules, result.Covers.Mapped, result.Covers.Absences, result.Covers.Allowlisted, result.Covers.New, len(result.Violations))
	return code
}

package witness

import (
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/ozgurcd/achta/internal/gitstate"
)

const (
	Schema    = "gate-run.v1"
	MaxRecord = int64(8 << 20)
	MaxLine   = 64 << 10
)

var (
	namePattern    = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)
	targetPattern  = regexp.MustCompile(`^target: ([A-Za-z0-9][A-Za-z0-9._-]*) exit=(-?[0-9]+)$`)
	elapsedPattern = regexp.MustCompile(`^elapsed: ([A-Za-z0-9][A-Za-z0-9._-]*) ([0-9]+)s$`)
	shaPattern     = regexp.MustCompile(`^[0-9a-f]{40}$`)
	digestPattern  = regexp.MustCompile(`^[0-9a-f]{64}$`)
)

type Record struct {
	RepoHead  string
	Started   *time.Time
	Finished  *time.Time
	Plan      []string
	Targets   map[string]int
	ElapsedMS map[string]int64
	TreeKind  string
	TreeValue string
	Result    string
}

type SlowTarget struct {
	Name      string `json:"name"`
	ElapsedMS int64  `json:"elapsed_ms"`
}

type Summary struct {
	SchemaVersion           string       `json:"schema_version"`
	RecordSchemaVersion     string       `json:"record_schema_version"`
	Status                  string       `json:"status"`
	Completeness            string       `json:"completeness"`
	Freshness               string       `json:"freshness"`
	RepositoryHead          string       `json:"repository_head"`
	RecordedHead            string       `json:"recorded_head,omitempty"`
	PlannedTargets          int          `json:"planned_targets"`
	RecordedTargets         int          `json:"recorded_targets"`
	PassedTargets           int          `json:"passed_targets"`
	FailedTargets           int          `json:"failed_targets"`
	MissingTargets          int          `json:"missing_targets"`
	StartedAt               string       `json:"started_at,omitempty"`
	FinishedAt              string       `json:"finished_at,omitempty"`
	WallElapsedMS           *int64       `json:"wall_elapsed_ms,omitempty"`
	RecordedTargetElapsedMS *int64       `json:"recorded_target_elapsed_ms,omitempty"`
	SlowTargets             []SlowTarget `json:"slow_targets"`
}

func Parse(data []byte) (Record, error) {
	if len(data) == 0 {
		return Record{}, errors.New("empty witness record")
	}
	record := Record{Targets: make(map[string]int), ElapsedMS: make(map[string]int64)}
	seen := make(map[string]bool)
	planSet := make(map[string]bool)
	lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
	for lineNumber, line := range lines {
		if line == "" && lineNumber == len(lines)-1 {
			continue
		}
		if len(line) > MaxLine {
			return Record{}, fmt.Errorf("line %d exceeds limit", lineNumber+1)
		}
		if strings.ContainsAny(line, "\x00\r") {
			return Record{}, fmt.Errorf("line %d contains a control character", lineNumber+1)
		}
		switch {
		case strings.HasPrefix(line, "schema: "):
			if err := singleton(seen, "schema"); err != nil {
				return Record{}, err
			}
			if strings.TrimPrefix(line, "schema: ") != Schema {
				return Record{}, fmt.Errorf("unsupported witness schema %q", strings.TrimPrefix(line, "schema: "))
			}
		case strings.HasPrefix(line, "repo-head: "):
			if err := singleton(seen, "repo-head"); err != nil {
				return Record{}, err
			}
			record.RepoHead = strings.TrimSuffix(strings.TrimPrefix(line, "repo-head: "), " (dirty)")
		case strings.HasPrefix(line, "started: "):
			if err := singleton(seen, "started"); err != nil {
				return Record{}, err
			}
			value, err := time.Parse(time.RFC3339, strings.TrimPrefix(line, "started: "))
			if err != nil {
				return Record{}, fmt.Errorf("invalid started timestamp")
			}
			record.Started = &value
		case strings.HasPrefix(line, "finished: "):
			if err := singleton(seen, "finished"); err != nil {
				return Record{}, err
			}
			value, err := time.Parse(time.RFC3339, strings.TrimPrefix(line, "finished: "))
			if err != nil {
				return Record{}, fmt.Errorf("invalid finished timestamp")
			}
			record.Finished = &value
		case strings.HasPrefix(line, "plan:"):
			if err := singleton(seen, "plan"); err != nil {
				return Record{}, err
			}
			for _, name := range strings.Fields(strings.TrimPrefix(line, "plan:")) {
				if !namePattern.MatchString(name) || planSet[name] {
					return Record{}, fmt.Errorf("invalid or duplicate planned target %q", name)
				}
				planSet[name] = true
				record.Plan = append(record.Plan, name)
			}
			if len(record.Plan) == 0 {
				return Record{}, errors.New("empty target plan")
			}
		case targetPattern.MatchString(line):
			match := targetPattern.FindStringSubmatch(line)
			if _, exists := record.Targets[match[1]]; exists {
				return Record{}, fmt.Errorf("duplicate target %q", match[1])
			}
			exitCode, err := strconv.Atoi(match[2])
			if err != nil {
				return Record{}, fmt.Errorf("invalid target exit code")
			}
			record.Targets[match[1]] = exitCode
		case elapsedPattern.MatchString(line):
			match := elapsedPattern.FindStringSubmatch(line)
			if _, exists := record.ElapsedMS[match[1]]; exists {
				return Record{}, fmt.Errorf("duplicate elapsed target %q", match[1])
			}
			seconds, err := strconv.ParseInt(match[2], 10, 64)
			if err != nil || seconds > (1<<63-1)/1000 {
				return Record{}, errors.New("invalid elapsed duration")
			}
			record.ElapsedMS[match[1]] = seconds * 1000
		case strings.HasPrefix(line, "tree: sha256="):
			if err := singleton(seen, "tree"); err != nil {
				return Record{}, err
			}
			record.TreeKind, record.TreeValue = "sha256", strings.TrimPrefix(line, "tree: sha256=")
			if !digestPattern.MatchString(record.TreeValue) {
				return Record{}, errors.New("invalid tree digest")
			}
		case strings.HasPrefix(line, "tree: commit="):
			if err := singleton(seen, "tree"); err != nil {
				return Record{}, err
			}
			record.TreeKind, record.TreeValue = "commit", strings.TrimSuffix(strings.TrimPrefix(line, "tree: commit="), " (dirty-at-finalize)")
			if !shaPattern.MatchString(record.TreeValue) {
				return Record{}, errors.New("invalid tree commit")
			}
		case strings.HasPrefix(line, "result: "):
			if err := singleton(seen, "result"); err != nil {
				return Record{}, err
			}
			record.Result = strings.TrimPrefix(line, "result: ")
			if record.Result != "green" && record.Result != "red" {
				return Record{}, errors.New("invalid witness result")
			}
		case allowedInformational(line):
			continue
		default:
			return Record{}, fmt.Errorf("unsupported line %d", lineNumber+1)
		}
	}
	if !seen["schema"] || !seen["plan"] {
		return Record{}, errors.New("witness is missing schema or plan")
	}
	for name := range record.Targets {
		if !planSet[name] {
			return Record{}, fmt.Errorf("unplanned target %q", name)
		}
	}
	for name := range record.ElapsedMS {
		if !planSet[name] {
			return Record{}, fmt.Errorf("elapsed time for unplanned target %q", name)
		}
	}
	if record.Started != nil && record.Finished != nil && record.Finished.Before(*record.Started) {
		return Record{}, errors.New("finished timestamp precedes started timestamp")
	}
	return record, nil
}

func Summarize(record Record, repo, recordPath, requireHead string, slowest int) (Summary, error) {
	if slowest < 0 || slowest > 100 {
		return Summary{}, errors.New("slowest must be between 0 and 100")
	}
	head, err := gitstate.Head(repo)
	if err != nil {
		return Summary{}, fmt.Errorf("repository HEAD: %w", err)
	}
	if requireHead != "" && (!shaPattern.MatchString(requireHead) || requireHead != head) {
		return Summary{}, errors.New("required HEAD does not match repository HEAD")
	}
	summary := Summary{SchemaVersion: "achta.witness-summary.v1", RecordSchemaVersion: Schema, RepositoryHead: head, SlowTargets: []SlowTarget{}, PlannedTargets: len(record.Plan), RecordedTargets: len(record.Targets)}
	if record.RepoHead != "" {
		if resolved, resolveErr := gitstate.ResolveCommit(repo, record.RepoHead); resolveErr == nil {
			summary.RecordedHead = resolved
		}
	}
	for _, name := range record.Plan {
		exitCode, ok := record.Targets[name]
		if !ok {
			summary.MissingTargets++
			continue
		}
		if exitCode == 0 {
			summary.PassedTargets++
		} else {
			summary.FailedTargets++
		}
	}
	summary.Completeness = "complete"
	if summary.MissingTargets > 0 || record.Started == nil || record.Finished == nil || record.TreeKind == "" || record.Result == "" {
		summary.Completeness = "incomplete"
	}
	summary.Status = "green"
	if record.Result == "red" || summary.FailedTargets > 0 {
		summary.Status = "red"
	} else if summary.Completeness != "complete" {
		summary.Status = "incomplete"
	}
	if record.Started != nil {
		summary.StartedAt = record.Started.UTC().Format(time.RFC3339)
	}
	if record.Finished != nil {
		summary.FinishedAt = record.Finished.UTC().Format(time.RFC3339)
	}
	if record.Started != nil && record.Finished != nil {
		value := record.Finished.Sub(*record.Started).Milliseconds()
		summary.WallElapsedMS = &value
	}
	if len(record.ElapsedMS) > 0 {
		var total int64
		all := make([]SlowTarget, 0, len(record.ElapsedMS))
		for name, elapsed := range record.ElapsedMS {
			total += elapsed
			all = append(all, SlowTarget{Name: name, ElapsedMS: elapsed})
		}
		summary.RecordedTargetElapsedMS = &total
		sort.Slice(all, func(i, j int) bool {
			if all[i].ElapsedMS != all[j].ElapsedMS {
				return all[i].ElapsedMS > all[j].ElapsedMS
			}
			return all[i].Name < all[j].Name
		})
		if slowest > len(all) {
			slowest = len(all)
		}
		summary.SlowTargets = all[:slowest]
	}
	summary.Freshness = "cannot_evaluate"
	switch record.TreeKind {
	case "commit":
		clean, cleanErr := gitstate.IsClean(repo)
		if cleanErr != nil {
			return Summary{}, cleanErr
		}
		if record.TreeValue == head && clean {
			summary.Freshness = "current"
		} else {
			summary.Freshness = "stale"
		}
	case "sha256":
		repoAbs, absErr := filepath.Abs(repo)
		if absErr != nil {
			return Summary{}, absErr
		}
		recordAbs, absErr := filepath.Abs(recordPath)
		if absErr != nil {
			return Summary{}, absErr
		}
		rel, relErr := filepath.Rel(repoAbs, recordAbs)
		if relErr != nil || rel == ".." || filepath.IsAbs(rel) || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return Summary{}, errors.New("record must be inside repository for digest freshness")
		}
		digest, digestErr := gitstate.TreeDigest(repo, rel)
		if digestErr != nil {
			return Summary{}, digestErr
		}
		if digest == record.TreeValue {
			summary.Freshness = "current"
		} else {
			summary.Freshness = "stale"
		}
	}
	return summary, nil
}

func singleton(seen map[string]bool, name string) error {
	if seen[name] {
		return fmt.Errorf("duplicate %s line", name)
	}
	seen[name] = true
	return nil
}

func allowedInformational(line string) bool {
	for _, prefix := range []string{"gate: ", "note: ", "cites: ", "evidence: ", "tool: ", "tie-note: ", "xrepo: "} {
		if strings.HasPrefix(line, prefix) {
			return true
		}
	}
	return false
}

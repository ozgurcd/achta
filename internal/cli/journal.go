package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/ozgurcd/achta/internal/claim"
	"github.com/ozgurcd/achta/internal/gitstate"
	"github.com/ozgurcd/achta/internal/safefile"
	"github.com/ozgurcd/achta/internal/witness"
)

const journalSchema = "achta.journal-brief.v1"
const journalNoteSchema = "achta.journal-note.v1"
const journalMax = 1 << 20

var journalName = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]{0,127}$`)
var journalSHA = regexp.MustCompile(`^[0-9a-f]{7,40}$`)

type journalGate struct {
	File    string `json:"file"`
	Result  string `json:"result"`
	Head    string `json:"repo_head"`
	Witness string `json:"witness"`
}
type journalRepo struct {
	Repository string        `json:"repository"`
	Slice      string        `json:"slice"`
	Head       string        `json:"head"`
	Origin     string        `json:"origin"`
	Ahead      *int          `json:"ahead"`
	Behind     *int          `json:"behind"`
	Dirty      int           `json:"dirty_files"`
	Gates      []journalGate `json:"gates"`
	Notes      []string      `json:"notes"`
	gitDir     string
}
type journalBrief struct {
	SchemaVersion string        `json:"schema_version"`
	Repositories  []journalRepo `json:"repositories"`
	Decisions     []string      `json:"decisions_today"`
}

// journalRoot uses the existing wiki workspace convention only for automatic
// discovery. Explicit roots need not contain a wiki (including fixture roots).
func journalRoot(root string) (string, error) {
	if root == "" {
		ws, err := resolveWorkspace(globalOptions{})
		if err != nil {
			return "", errors.New("journal requires --workspace ABS")
		}
		root = ws.Root
		for dir := filepath.Dir(root); dir != filepath.Dir(dir); dir = filepath.Dir(dir) {
			if info, err := os.Stat(filepath.Join(dir, "wiki", "platform", "decisions.md")); err == nil && info.Mode().IsRegular() {
				root = dir
			}
		}
	}
	if !filepath.IsAbs(root) || claim.ValidateText(root, 4096) != nil {
		return "", errors.New("invalid journal workspace")
	}
	return filepath.EvalSymlinks(root)
}

func journalRepos(root string, now time.Time) ([]journalRepo, error) {
	entries, err := os.ReadDir(root)
	if err != nil || len(entries) > 4096 {
		return nil, errors.New("workspace unavailable or oversized")
	}
	paths := []string{root}
	for _, entry := range entries {
		if entry.IsDir() && !strings.HasPrefix(entry.Name(), ".") {
			paths = append(paths, filepath.Join(root, entry.Name()))
		}
	}
	result := []journalRepo{}
	for _, path := range paths {
		if _, err := os.Lstat(filepath.Join(path, ".git")); os.IsNotExist(err) {
			continue
		}
		dir, err := gitstate.CheckoutGitDir(path)
		if err != nil {
			return nil, errors.New("cannot inspect checkout metadata")
		}
		if _, err := os.Lstat(filepath.Join(dir, "achta-claim.json")); os.IsNotExist(err) {
			continue
		}
		state, err := claim.Run("status", claim.Options{Repo: path, Now: now})
		if err != nil {
			return nil, errors.New("cannot inspect claim")
		}
		if state.Claim == nil {
			continue
		}
		if !journalName.MatchString(state.Claim.Slice) || claim.ValidateText(path, 4096) != nil {
			return nil, errors.New("unsafe journal identity")
		}
		result = append(result, journalRepo{Repository: path, Slice: state.Claim.Slice, gitDir: dir, Gates: []journalGate{}, Notes: []string{}})
	}
	return result, nil
}

func collectJournal(root string, now time.Time) (journalBrief, error) {
	doc := journalBrief{SchemaVersion: journalSchema, Decisions: []string{}}
	repos, err := journalRepos(root, now)
	if err != nil {
		return doc, err
	}
	for i := range repos {
		r := &repos[i]
		r.Head, err = gitstate.Head(r.Repository)
		if err != nil {
			return doc, errors.New("cannot inspect repository HEAD")
		}
		r.Dirty, err = gitstate.DirtyFiles(r.Repository)
		if err != nil {
			return doc, err
		}
		branch, err := gitstate.Branch(r.Repository)
		if err == nil && branch != "detached" {
			ref := "refs/remotes/origin/" + branch
			if ahead, behind, err := gitstate.AheadBehind(r.Repository, ref); err == nil {
				r.Origin, r.Ahead, r.Behind = ref, &ahead, &behind
			}
		}
		paths, _ := filepath.Glob(filepath.Join(r.Repository, "GATE-RUN*.txt"))
		for _, path := range paths {
			if claim.ValidateText(filepath.Base(path), 256) != nil {
				return doc, errors.New("unsafe gate filename")
			}
			g := journalGate{File: filepath.Base(path), Result: "unavailable", Witness: "unavailable"}
			if snap, e := safefile.Read(r.Repository, path, witness.MaxRecord); e == nil {
				if record, e := witness.Parse(snap.Data); e == nil && journalSHA.MatchString(record.RepoHead) && record.Result != "" {
					g.Result, g.Head = record.Result, record.RepoHead
				}
			}
			if sha, e := gitstate.JournalWitness(r.Repository, path); e == nil {
				g.Witness = sha
			}
			r.Gates = append(r.Gates, g)
		}
		notes, err := readJournalNotes(r.gitDir, r.Slice)
		if err != nil {
			return doc, err
		}
		if len(notes) > 3 {
			notes = notes[len(notes)-3:]
		}
		r.Notes = notes
	}
	doc.Repositories = repos
	// Only decision identifiers and dates are extracted; prose never leaves the file.
	path := filepath.Join(root, "wiki", "platform", "decisions.md")
	if snap, e := safefile.Read(root, path, 16<<20); e == nil {
		heading := regexp.MustCompile(`^### ([A-Z][A-Z0-9]*-[0-9]+)\b.*\(` + now.UTC().Format("2006-01-02") + `\)$`)
		for _, line := range strings.Split(string(snap.Data), "\n") {
			if m := heading.FindStringSubmatch(line); m != nil {
				doc.Decisions = append(doc.Decisions, m[1])
			}
		}
	} else if !os.IsNotExist(e) {
		return doc, errors.New("cannot inspect decisions")
	}
	return doc, nil
}

// Fixed category lines preserve the whole selected set without silently
// dropping claimed repositories to fit a line budget.
func journalText(doc journalBrief) string {
	claims, states, gates, witnesses, notes := []string{}, []string{}, []string{}, []string{}, []string{}
	for _, r := range doc.Repositories {
		name := filepath.Base(r.Repository)
		claims = append(claims, r.Slice+"@"+r.Repository)
		comparison := "origin unavailable"
		if r.Ahead != nil {
			comparison = fmt.Sprintf("ahead=%d behind=%d", *r.Ahead, *r.Behind)
		}
		states = append(states, fmt.Sprintf("%s HEAD=%s %s dirty=%d", name, r.Head, comparison, r.Dirty))
		for _, g := range r.Gates {
			gates = append(gates, fmt.Sprintf("%s/%s %s repo-head=%s", name, g.File, g.Result, g.Head))
			witnesses = append(witnesses, name+"/"+g.File+"="+g.Witness)
		}
		for _, note := range r.Notes {
			notes = append(notes, r.Slice+"@"+name+": "+note)
		}
	}
	join := func(s []string) string {
		if len(s) == 0 {
			return "none"
		}
		return strings.Join(s, "; ")
	}
	return fmt.Sprintf("Journal (live, local origin refs; UTC decisions)\nclaims: %s\nrepositories: %s\ngates: %s\nwitnesses: %s\ndecisions today: %s\nnotes: %s\n", join(claims), join(states), join(gates), join(witnesses), join(doc.Decisions), join(notes))
}

func readJournalNotes(dir, slice string) ([]string, error) {
	snap, err := safefile.Read(dir, filepath.Join(dir, "achta", "journal", slice+".log"), journalMax)
	if os.IsNotExist(err) {
		return []string{}, nil
	}
	if err != nil {
		return nil, errors.New("cannot safely read journal notes")
	}
	notes := strings.Split(strings.TrimSuffix(string(snap.Data), "\n"), "\n")
	if len(notes) > 200 {
		return nil, errors.New("oversized journal history")
	}
	for _, note := range notes {
		if claim.ValidateText(note, 1024) != nil {
			return nil, errors.New("unsafe journal note")
		}
	}
	return notes, nil
}

func appendJournalNote(r journalRepo, text string, now time.Time) error {
	// Share claim writers' lock, so the holder cannot change during this append.
	lock, err := os.OpenFile(filepath.Join(r.gitDir, "achta-claim.lock"), os.O_CREATE|os.O_RDWR|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0600)
	if err != nil {
		return errors.New("cannot lock journal claim")
	}
	defer lock.Close()
	info, err := lock.Stat()
	if err != nil || !info.Mode().IsRegular() {
		return errors.New("unsafe claim lock")
	}
	if syscall.Flock(int(lock.Fd()), syscall.LOCK_EX|syscall.LOCK_NB) != nil {
		return errors.New("claim or journal busy")
	}
	defer syscall.Flock(int(lock.Fd()), syscall.LOCK_UN)
	state, err := claim.Run("status", claim.Options{Repo: r.Repository, Now: now})
	if err != nil || state.Claim == nil || state.Claim.Slice != r.Slice {
		return errors.New("claim changed before note")
	}
	for _, path := range []string{filepath.Join(r.gitDir, "achta"), filepath.Join(r.gitDir, "achta", "journal")} {
		if err := os.Mkdir(path, 0700); err != nil && !os.IsExist(err) {
			return errors.New("cannot create journal directory")
		}
		info, err := os.Lstat(path)
		if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return errors.New("unsafe journal directory")
		}
		if os.Chmod(path, 0700) != nil {
			return errors.New("cannot secure journal directory")
		}
	}
	notes, err := readJournalNotes(r.gitDir, r.Slice)
	if err != nil {
		return err
	}
	notes = append(notes, text)
	if len(notes) > 200 {
		notes = notes[len(notes)-200:]
	}
	data := []byte(strings.Join(notes, "\n") + "\n")
	path := filepath.Join(r.gitDir, "achta", "journal", r.Slice+".log")
	snap, err := safefile.Read(r.gitDir, path, journalMax)
	if os.IsNotExist(err) {
		err = safefile.Create(r.gitDir, path, data, journalMax, 0600)
	} else if err == nil {
		if snap.Mode != 0600 {
			return errors.New("journal note file must be private")
		}
		err = snap.Replace(data, journalMax)
	}
	if err != nil {
		return errors.New("cannot safely append journal note")
	}
	return nil
}

func runJournal(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	opts.json = opts.json || jsonRequested(args)
	fail := func() int {
		return renderError(stdout, stderr, opts.json, journalSchema, invalid("journal unavailable: check workspace, claim and bounded non-secret input"))
	}
	if len(args) == 0 || (args[0] != "brief" && args[0] != "note") {
		return fail()
	}
	set := flagSet("journal")
	root := set.String("workspace", opts.workspace, "absolute workspace")
	slice := set.String("slice", "", "claim holder")
	jsonMode := set.Bool("json", opts.json, "one JSON document")
	if set.Parse(args[1:]) != nil {
		return fail()
	}
	opts.json = *jsonMode
	workspace, err := journalRoot(*root)
	if err != nil {
		return fail()
	}
	now := time.Now()
	if args[0] == "brief" {
		if set.NArg() != 0 || *slice != "" {
			return fail()
		}
		doc, err := collectJournal(workspace, now)
		if err != nil {
			return fail()
		}
		if opts.json {
			return writeJSON(stdout, stderr, doc)
		}
		fmt.Fprint(stdout, journalText(doc))
		return 0
	}
	if set.NArg() != 1 || !journalName.MatchString(*slice) || strings.TrimSpace(set.Arg(0)) == "" || claim.ValidateText(set.Arg(0), 1024) != nil {
		return fail()
	}
	repos, err := journalRepos(workspace, now)
	if err != nil {
		return fail()
	}
	count := 0
	for _, r := range repos {
		if r.Slice == *slice {
			if appendJournalNote(r, set.Arg(0), now) != nil {
				return fail()
			}
			count++
		}
	}
	if count == 0 {
		return fail()
	}
	if opts.json {
		return writeJSON(stdout, stderr, struct {
			SchemaVersion string `json:"schema_version"`
			Repositories  int    `json:"repositories"`
		}{journalNoteSchema, count})
	}
	fmt.Fprintf(stdout, "journal note recorded in %d claimed checkout(s)\n", count)
	return 0
}

// All work, including input reading, has a small wall bound. Only a completed
// result is published, so errors and timeouts cannot emit partial context.
func runJournalHook(stdin io.Reader, stdout io.Writer) int {
	done := make(chan string, 1)
	go func() {
		defer func() {
			if recover() != nil {
				done <- ""
			}
		}()
		data, err := io.ReadAll(io.LimitReader(stdin, 65537))
		if err != nil || len(data) > 65536 {
			done <- ""
			return
		}
		var payload struct {
			Source string `json:"source"`
			CWD    string `json:"cwd"`
		}
		if json.Unmarshal(data, &payload) != nil || (payload.Source != "compact" && payload.Source != "resume") || !filepath.IsAbs(payload.CWD) {
			done <- ""
			return
		}
		root := payload.CWD
		// Discover the enclosing workspace marker without reading instruction files.
		for dir := payload.CWD; ; dir = filepath.Dir(dir) {
			if info, e := os.Stat(filepath.Join(dir, "wiki", "platform", "decisions.md")); e == nil && info.Mode().IsRegular() {
				root = dir
			}
			if filepath.Dir(dir) == dir {
				break
			}
		}
		doc, err := collectJournal(root, time.Now())
		if err != nil {
			done <- ""
			return
		}
		var out bytes.Buffer
		if json.NewEncoder(&out).Encode(map[string]any{"hookSpecificOutput": map[string]string{"hookEventName": "SessionStart", "additionalContext": journalText(doc)}}) != nil {
			done <- ""
			return
		}
		done <- out.String()
	}()
	select {
	case result := <-done:
		fmt.Fprint(stdout, result)
	case <-time.After(2 * time.Second):
	}
	return 0
}

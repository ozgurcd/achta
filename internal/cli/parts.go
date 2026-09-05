package cli

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ozgurcd/achta/internal/parts"
	"github.com/ozgurcd/achta/internal/safefile"
	"github.com/ozgurcd/achta/internal/workspace"
)

type partsOperationDocument struct {
	SchemaVersion string `json:"schema_version"`
	Operation     string `json:"operation"`
	Status        string `json:"status"`
	Directory     string `json:"dir"`
	LockFile      string `json:"lock"`
	Version       string `json:"version"`
	Parts         int    `json:"parts"`
	Changed       bool   `json:"changed"`
	Bumped        bool   `json:"bumped"`
}

func runParts(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	if len(args) == 0 {
		return renderError(stdout, stderr, opts.json, parts.VerifySchema, invalid("parts requires lock or verify"))
	}
	switch args[0] {
	case "lock":
		return runPartsLock(args[1:], stdout, stderr, opts)
	case "verify":
		return runPartsVerify(args[1:], stdout, stderr, opts)
	default:
		return renderError(stdout, stderr, opts.json, parts.VerifySchema, invalid("unknown parts subcommand %q", args[0]))
	}
}

func runPartsLock(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	set := flagSet("parts lock")
	dirValue := set.String("dir", "", "directory of direct .txt part files (inside the workspace)")
	lockValue := set.String("lock", "", "lock artifact (inside the workspace)")
	bump := set.Bool("bump", false, "increment the existing lock version")
	jsonMode := set.Bool("json", opts.json, "emit JSON")
	if err := parseFlags(set, args); err != nil {
		opts.json = *jsonMode
		return renderError(stdout, stderr, opts.json, parts.OperationSchema, err)
	}
	opts.json = *jsonMode
	if *dirValue == "" || *lockValue == "" {
		return renderError(stdout, stderr, opts.json, parts.OperationSchema, invalid("--dir and --lock are required"))
	}
	ws, err := resolveWorkspace(opts)
	if err != nil {
		return renderError(stdout, stderr, opts.json, parts.OperationSchema, err)
	}
	input, err := readPartDirectory(ws, *dirValue)
	if err != nil {
		return renderError(stdout, stderr, opts.json, parts.OperationSchema, invalid("read parts: %v", err))
	}
	lockPath, err := confinedPath(ws, *lockValue)
	if err != nil {
		return renderError(stdout, stderr, opts.json, parts.OperationSchema, err)
	}
	var current *parts.Lock
	var snapshot safefile.Snapshot
	present := true
	snapshot, err = safefile.Read(ws.Root, lockPath, parts.MaxLock)
	if errors.Is(err, os.ErrNotExist) {
		present = false
	} else if err != nil {
		return renderError(stdout, stderr, opts.json, parts.OperationSchema, invalid("read lock: %v", err))
	} else {
		parsed, parseErr := parts.Parse(snapshot.Data)
		if parseErr != nil {
			return renderError(stdout, stderr, opts.json, parts.OperationSchema, invalid("parse lock: %v", parseErr))
		}
		current = &parsed
	}
	lock, encoded, err := parts.Update(current, *bump, input)
	if err != nil {
		return renderError(stdout, stderr, opts.json, parts.OperationSchema, invalid("parts lock: %v", err))
	}
	confirmed, err := readPartDirectory(ws, *dirValue)
	if err != nil {
		return renderError(stdout, stderr, opts.json, parts.OperationSchema, invalid("recheck parts: %v", err))
	}
	if !sameParts(input, confirmed) {
		return renderError(stdout, stderr, opts.json, parts.OperationSchema, invalid("part directory changed concurrently; refusing lock write"))
	}
	changed := !present || !bytes.Equal(snapshot.Data, encoded)
	if changed {
		if present {
			err = snapshot.Replace(encoded, parts.MaxLock)
		} else {
			err = safefile.Create(ws.Root, lockPath, encoded, parts.MaxLock, 0o644)
		}
		if err != nil {
			return renderError(stdout, stderr, opts.json, parts.OperationSchema, invalid("write lock: %v", err))
		}
	}
	status := "unchanged"
	if changed {
		status = "updated"
	}
	document := partsOperationDocument{SchemaVersion: parts.OperationSchema, Operation: "lock", Status: status, Directory: *dirValue, LockFile: *lockValue, Version: lock.Version, Parts: len(lock.Entries), Changed: changed, Bumped: *bump}
	if opts.json {
		return writeJSON(stdout, stderr, document)
	}
	fmt.Fprintf(stdout, "parts lock: %s; version=%s parts=%d lock=%s\n", status, lock.Version, len(lock.Entries), *lockValue)
	return 0
}

// RULE: PARTS-LOCK-1
func runPartsVerify(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	set := flagSet("parts verify")
	dirValue := set.String("dir", "", "directory of direct .txt part files (inside the workspace)")
	lockValue := set.String("lock", "", "lock artifact (inside the workspace)")
	jsonMode := set.Bool("json", opts.json, "emit JSON")
	if err := parseFlags(set, args); err != nil {
		opts.json = *jsonMode
		return renderError(stdout, stderr, opts.json, parts.VerifySchema, err)
	}
	opts.json = *jsonMode
	if *dirValue == "" || *lockValue == "" {
		return renderError(stdout, stderr, opts.json, parts.VerifySchema, invalid("--dir and --lock are required"))
	}
	ws, err := resolveWorkspace(opts)
	if err != nil {
		return renderError(stdout, stderr, opts.json, parts.VerifySchema, err)
	}
	input, err := readPartDirectory(ws, *dirValue)
	if err != nil {
		return renderError(stdout, stderr, opts.json, parts.VerifySchema, invalid("read parts: %v", err))
	}
	lockPath, err := confinedPath(ws, *lockValue)
	if err != nil {
		return renderError(stdout, stderr, opts.json, parts.VerifySchema, err)
	}
	snapshot, err := safefile.Read(ws.Root, lockPath, parts.MaxLock)
	if err != nil {
		return renderError(stdout, stderr, opts.json, parts.VerifySchema, invalid("read lock: %v", err))
	}
	lock, err := parts.Parse(snapshot.Data)
	if err != nil {
		return renderError(stdout, stderr, opts.json, parts.VerifySchema, invalid("parse lock: %v", err))
	}
	confirmedLock, err := safefile.Read(ws.Root, lockPath, parts.MaxLock)
	if err != nil {
		return renderError(stdout, stderr, opts.json, parts.VerifySchema, invalid("recheck lock: %v", err))
	}
	if confirmedLock.Size != snapshot.Size || confirmedLock.ModTime != snapshot.ModTime || confirmedLock.Mode != snapshot.Mode || confirmedLock.Digest != snapshot.Digest {
		return renderError(stdout, stderr, opts.json, parts.VerifySchema, invalid("lock changed concurrently; refusing verification"))
	}
	confirmedInput, err := readPartDirectory(ws, *dirValue)
	if err != nil {
		return renderError(stdout, stderr, opts.json, parts.VerifySchema, invalid("recheck parts: %v", err))
	}
	if !sameParts(input, confirmedInput) {
		return renderError(stdout, stderr, opts.json, parts.VerifySchema, invalid("part directory changed concurrently; refusing verification"))
	}
	result, err := parts.Verify(*dirValue, *lockValue, lock, input)
	if err != nil {
		return renderError(stdout, stderr, opts.json, parts.VerifySchema, invalid("parts verify: %v", err))
	}
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
	for _, check := range result.Checks {
		if check.Status == "match" {
			continue
		}
		fmt.Fprintf(stdout, "parts verify: %s %s", check.Name, check.Status)
		if check.LockedSHA256 != "" {
			fmt.Fprintf(stdout, "; locked_sha256=%s", check.LockedSHA256)
		}
		if check.ActualSHA256 != "" {
			fmt.Fprintf(stdout, "; actual_sha256=%s", check.ActualSHA256)
		}
		fmt.Fprintln(stdout)
	}
	fmt.Fprintf(stdout, "parts verify: %s; version=%s locked=%d on_disk=%d\n", result.Status, result.Version, result.Locked, result.OnDisk)
	return code
}

func readPartDirectory(ws workspace.Workspace, value string) ([]parts.Part, error) {
	directory, err := confinedPath(ws, value)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(directory)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%q is not a directory", value)
	}
	entries, err := f.ReadDir(parts.MaxDirEntries + 1)
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, err
	}
	if len(entries) > parts.MaxDirEntries {
		return nil, fmt.Errorf("directory exceeds %d entries", parts.MaxDirEntries)
	}
	input := make([]parts.Part, 0)
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".txt") {
			continue
		}
		if !entry.Type().IsRegular() || !parts.ValidName(entry.Name()) {
			return nil, fmt.Errorf("part %q is not a valid regular file", entry.Name())
		}
		path := filepath.Join(directory, entry.Name())
		snapshot, err := safefile.Read(ws.Root, path, parts.MaxPart)
		if err != nil {
			return nil, fmt.Errorf("read part %q: %w", entry.Name(), err)
		}
		input = append(input, parts.Part{Name: entry.Name(), Data: snapshot.Data})
	}
	sort.Slice(input, func(i, j int) bool { return input[i].Name < input[j].Name })
	if len(input) == 0 {
		return nil, errors.New("directory contains no direct .txt part files")
	}
	if len(input) > parts.MaxParts {
		return nil, fmt.Errorf("directory exceeds %d part files", parts.MaxParts)
	}
	return input, nil
}

func sameParts(left, right []parts.Part) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i].Name != right[i].Name || !bytes.Equal(left[i].Data, right[i].Data) {
			return false
		}
	}
	return true
}

package cli

import (
	"bufio"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/ozgurcd/achta/internal/claim"
	"github.com/ozgurcd/achta/internal/gitstate"
)

const gateSchema = "achta.gate-run.v1"

type gateResult struct {
	SchemaVersion string   `json:"schema_version"`
	Command       string   `json:"command"`
	ExitCode      int      `json:"exit_code"`
	DurationMS    int64    `json:"duration_ms"`
	LogPath       string   `json:"log_path"`
	Failures      []string `json:"failures"`
}

// gateSummary deliberately withholds lines that can carry values. The private
// log retains their exact bytes. This is presentation, never a gate judgement.
func gateSummary(s string) string {
	if claim.ValidateText(s, 512) != nil || strings.ContainsAny(s, "=\"'\\") {
		return "[details withheld; see private log]"
	}
	return s
}

func gateFlags(args []string) []string {
	for i, value := range args {
		if value == "--" {
			return args[:i]
		}
	}
	return args
}

func runGate(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	opts.json = opts.json || jsonRequested(gateFlags(args))
	refuse := func(message string) int {
		return renderError(stdout, stderr, opts.json, gateSchema, invalid("%s", message))
	}
	if len(args) < 1 || args[0] != "run" {
		return refuse("gate requires run --repo ABS -- COMMAND [ARG...]")
	}
	flags := gateFlags(args[1:])
	if len(flags)+2 >= len(args) {
		return refuse("gate run requires -- COMMAND [ARG...]")
	}
	set := flagSet("gate run")
	repo := set.String("repo", "", "absolute checkout root")
	jsonMode := set.Bool("json", opts.json, "emit JSON")
	if set.Parse(flags) != nil || set.NArg() != 0 {
		return refuse("invalid gate run flags")
	}
	opts.json = *jsonMode
	if rejectSecretLikeInput(*repo) != nil || claim.ValidateText(*repo, 4096) != nil {
		return refuse("invalid gate repository selector")
	}
	gitDir, err := gitstate.CheckoutGitDir(*repo)
	if err != nil {
		return refuse("gate run requires an available absolute Git checkout root")
	}
	dir := filepath.Join(gitDir, "achta", "logs")
	for _, path := range []string{filepath.Join(gitDir, "achta"), dir} {
		if err := os.Mkdir(path, 0700); err != nil && !os.IsExist(err) {
			return refuse("cannot create private gate log directory")
		}
		info, err := os.Lstat(path)
		if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return refuse("gate log directory must be a real directory")
		}
		if os.Chmod(path, 0700) != nil {
			return refuse("cannot secure gate log directory")
		}
	}
	// Serialize runs in this checkout, so retention cannot unlink an active log.
	lock, err := os.OpenFile(filepath.Join(dir, ".lock"), os.O_CREATE|os.O_RDWR|syscall.O_NOFOLLOW, 0600)
	if err != nil {
		return refuse("cannot open gate log lock")
	}
	defer lock.Close()
	if syscall.Flock(int(lock.Fd()), syscall.LOCK_EX|syscall.LOCK_NB) != nil {
		return refuse("another gate run holds this checkout log directory")
	}
	defer syscall.Flock(int(lock.Fd()), syscall.LOCK_UN)
	argv := args[len(flags)+2:]
	name := regexp.MustCompile(`[^A-Za-z0-9_-]`).ReplaceAllString(filepath.Base(argv[0]), "_")
	if len(name) > 40 {
		name = name[:40]
	}
	if claim.ValidateText(name, 40) != nil {
		name = "command"
	}
	log, err := os.CreateTemp(dir, time.Now().UTC().Format("20060102T150405.000000000Z")+"-"+name+"-*.log")
	if err != nil {
		return refuse("cannot create private gate log")
	}
	before := gateRecords(*repo)
	started := time.Now()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	command := exec.CommandContext(ctx, argv[0], argv[1:]...)
	command.Dir, command.Stdout, command.Stderr = *repo, log, log
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	command.Cancel = func() error { return syscall.Kill(-command.Process.Pid, syscall.SIGKILL) }
	result := gateResult{SchemaVersion: gateSchema, Command: name + " (arguments withheld)", LogPath: log.Name(), Failures: []string{}}
	if err := command.Run(); err != nil {
		result.ExitCode = 127
		if failure, ok := err.(*exec.ExitError); ok {
			result.ExitCode = failure.ExitCode()
			if status, ok := failure.Sys().(syscall.WaitStatus); ok && status.Signaled() {
				result.ExitCode = 128 + int(status.Signal())
			}
		} else {
			_, _ = fmt.Fprintln(log, "achta: command could not start")
		}
	}
	result.DurationMS = time.Since(started).Milliseconds()
	if log.Close() != nil {
		return refuse("cannot finish gate log")
	}
	if result.ExitCode != 0 {
		result.Failures = gateFailures(*repo, result.LogPath, before)
	}
	entries, err := filepath.Glob(filepath.Join(dir, "*.log"))
	if err != nil {
		return refuse("cannot enumerate gate logs")
	}
	sort.Strings(entries)
	for len(entries) > 20 {
		if os.Remove(entries[0]) != nil {
			return refuse("cannot retain newest gate logs")
		}
		entries = entries[1:]
	}
	if opts.json {
		if writeJSON(stdout, stderr, result) != 0 {
			return 2
		}
	} else {
		fmt.Fprintf(stdout, "command: %s\nexit code: %d\nduration: %dms\nlog: %s\n", result.Command, result.ExitCode, result.DurationMS, result.LogPath)
		for _, line := range result.Failures {
			fmt.Fprintln(stdout, line)
		}
	}
	return result.ExitCode
}

func gateRecord(path string) []byte {
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Size() > 4<<20 {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	return data
}

func gateRecords(repo string) map[string][32]byte {
	result := map[string][32]byte{}
	paths, _ := filepath.Glob(filepath.Join(repo, "GATE-RUN*.txt"))
	for _, path := range paths {
		result[path] = sha256.Sum256(gateRecord(path))
	}
	return result
}

func gateFailures(repo, logPath string, before map[string][32]byte) []string {
	var lines []string
	paths, _ := filepath.Glob(filepath.Join(repo, "GATE-RUN*.txt"))
	target := regexp.MustCompile(`^target: ([A-Za-z0-9_.-]+) exit=([1-9][0-9]*)$`)
	for _, path := range paths {
		data := gateRecord(path)
		if hash, ok := before[path]; ok && hash == sha256.Sum256(data) {
			continue
		}
		failed := map[string]bool{}
		for _, line := range strings.Split(string(data), "\n") {
			if match := target.FindStringSubmatch(line); match != nil {
				failed[match[1]] = true
			}
		}
		for _, line := range strings.Split(string(data), "\n") {
			for name := range failed {
				if strings.HasPrefix(line, "evidence: ["+name+"] ") {
					lines = append(lines, gateSummary(line))
				}
			}
		}
	}
	if len(lines) == 0 {
		file, err := os.Open(logPath)
		if err != nil {
			return []string{"failure details unavailable; see private log"}
		}
		defer file.Close()
		if info, err := file.Stat(); err == nil && info.Size() > 64<<10 {
			_, _ = file.Seek(-(64 << 10), io.SeekEnd)
		}
		scanner := bufio.NewScanner(file)
		scanner.Buffer(make([]byte, 4096), 65537)
		for scanner.Scan() {
			line := scanner.Text()
			lower := strings.ToLower(line)
			if strings.Contains(lower, "error") || strings.Contains(lower, "fail") || strings.Contains(lower, "panic") || strings.Contains(lower, "refus") {
				lines = append(lines, gateSummary(line))
			}
		}
	}
	if len(lines) > 20 {
		lines = lines[len(lines)-20:]
	}
	if len(lines) == 0 {
		return []string{"command failed; see private log"}
	}
	return lines
}

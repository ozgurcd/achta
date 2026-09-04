package gitstate

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewestWitnessHistoryMatrix(t *testing.T) {
	repo := t.TempDir()
	runGitTest(t, repo, "init", "-b", "main")
	runGitTest(t, repo, "config", "user.name", "Achta Test")
	runGitTest(t, repo, "config", "user.email", "achta@example.invalid")
	record := filepath.Join(repo, "GATE-RUN.txt")
	writeGitTest(t, record, "one\n")
	runGitTest(t, repo, "add", "GATE-RUN.txt")
	runGitTest(t, repo, "commit", "-m", "base")
	if _, err := NewestWitness(repo, record); err == nil {
		t.Fatal("history without an accepted witness passed")
	}

	writeGitTest(t, record, "malformed\n")
	runGitTest(t, repo, "add", "GATE-RUN.txt")
	runGitTest(t, repo, "commit", "-m", "prefix Witness: make verify green at wrong-position")
	writeGitTest(t, record, "accepted-one\n")
	runGitTest(t, repo, "add", "GATE-RUN.txt")
	runGitTest(t, repo, "commit", "-m", "Witness: make verify green at fixture-one")
	first := runGitTest(t, repo, "rev-parse", "HEAD")
	writeGitTest(t, filepath.Join(repo, "work.txt"), "first commit after witness\n")
	runGitTest(t, repo, "add", "work.txt")
	runGitTest(t, repo, "commit", "-m", "work")
	got, err := NewestWitness(repo, record)
	if err != nil || got != first {
		t.Fatalf("first commit after witness = %q, err=%v, want %q", got, err, first)
	}

	writeGitTest(t, record, "accepted-two\n")
	runGitTest(t, repo, "add", "GATE-RUN.txt")
	runGitTest(t, repo, "commit", "-m", "Witness: make verify green at fixture-two")
	second := runGitTest(t, repo, "rev-parse", "HEAD")
	got, err = NewestWitness(repo, record)
	if err != nil || got != second {
		t.Fatalf("newest witness = %q, err=%v, want %q", got, err, second)
	}

	runGitTest(t, repo, "switch", "-c", "witness-topic")
	writeGitTest(t, record, "accepted-on-topic\n")
	runGitTest(t, repo, "add", "GATE-RUN.txt")
	runGitTest(t, repo, "commit", "-m", "Witness: make verify green at merged-fixture")
	mergedWitness := runGitTest(t, repo, "rev-parse", "HEAD")
	runGitTest(t, repo, "switch", "main")
	writeGitTest(t, filepath.Join(repo, "main.txt"), "main work\n")
	runGitTest(t, repo, "add", "main.txt")
	runGitTest(t, repo, "commit", "-m", "main work")
	runGitTest(t, repo, "merge", "--no-ff", "witness-topic", "-m", "merge witness history")
	got, err = NewestWitness(repo, record)
	if err != nil || got != mergedWitness {
		t.Fatalf("merged witness = %q, err=%v, want %q", got, err, mergedWitness)
	}
}

func TestRunRefusesMissingGitExecutable(t *testing.T) {
	t.Setenv("PATH", "")
	_, err := run(".", nil, "--version")
	if err == nil || !strings.Contains(err.Error(), "discover Git executable") {
		t.Fatalf("missing Git discovery err = %v", err)
	}
}

// RULE: GIT-READ-ONLY-1
func TestRunDisablesOptionalGitLocks(t *testing.T) {
	dir := t.TempDir()
	gitPath := filepath.Join(dir, "git")
	if err := os.WriteFile(gitPath, []byte("#!/bin/sh\nprintf '%s\\n' \"$@\"\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	output, err := run("/fixture/repo", nil, "status", "--porcelain")
	if err != nil {
		t.Fatal(err)
	}
	want := "--no-optional-locks\n-C\n/fixture/repo\nstatus\n--porcelain\n"
	if output != want {
		t.Fatalf("git arguments\nwant: %q\ngot:  %q", want, output)
	}
}

func runGitTest(t *testing.T, repo string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v: %s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

func writeGitTest(t *testing.T, path, value string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(value), 0o644); err != nil {
		t.Fatal(err)
	}
}

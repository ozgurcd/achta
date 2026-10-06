package hookcd

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"testing"
)

// RULE: HOOK-CD-READS-1
func TestReadOnlyForms(t *testing.T) {
	for _, command := range []string{
		"git config --get user.name",
		"git config --get-all user.name",
		"git config --get-regexp '^user\\.'",
		"git config --list", "git config -l",
		"git config --show-origin --get user.name",
		"git config --show-origin --list",
		"git config --show-origin user.name",
		"git config user.name",
		"git config --global user.name",
		"git config --file config user.name",
		"git -c example.value=config config --get user.name",
		"git --git-dir config config user.name",
		"git config --type bool --get feature.enabled",
		"git config --get-regexp user.name pattern",
		"git config -- user.name",
		"achta --help", "achta -h",
		"achta wiki derive", "achta --json wiki derive --check",
		"achta wiki derive --print sample",
	} {
		t.Run(command, func(t *testing.T) { assertHookCommand(t, command, Allow) })
	}
	for name := range achtaWriteCommands {
		for _, flag := range []string{"--help", "-h"} {
			command := "achta " + name + " " + flag
			t.Run(command, func(t *testing.T) { assertHookCommand(t, command, Allow) })
		}
	}
}

func TestPreservedWriteForms(t *testing.T) {
	for _, args := range []string{
		"user.name Example", "--add user.name Example", "--replace-all user.name Example",
		"--unset user.name", "--unset-all user.name", "--edit", "-e",
		"--rename-section old new", "--remove-section old",
		"--global user.name Example", "--file config user.name Example",
		"--get user.name --unset user.name", "--list --add user.name Example",
		"--show-origin user.name Example", "-- user.name Example",
		"--unknown user.name", "", "--show-origin",
	} {
		command := "git config " + args
		t.Run(command, func(t *testing.T) { assertHookCommand(t, command, Deny) })
	}
	for name := range gitWriteVerbs {
		command := "git " + name
		t.Run(command, func(t *testing.T) { assertHookCommand(t, command, Deny) })
	}
	for name := range achtaWriteCommands {
		command := "achta " + name
		if name == "wiki derive" {
			command += " --write"
		}
		t.Run(command, func(t *testing.T) { assertHookCommand(t, command, Deny) })
	}
	for _, command := range []string{
		"git config --get user.name > out",
		"git config user.name | tee out",
		"achta decision add --help > out",
		"achta decision add --help; touch out",
		"achta wiki derive | tee out",
		"achta wiki derive --write=true",
		"achta decision add --title --help --body-file body.md",
		"achta claim take --repo /repo --slice first --note -h",
		"achta claim release --repo /repo --slice second --force --reason --help",
		"achta wiki pin -- --help",
	} {
		t.Run(command, func(t *testing.T) { assertHookCommand(t, command, Deny) })
	}
}

// The optional executable replays the exact cases against an installed release.
// Payload commands are data: neither this harness nor the hook executes them.
func assertHookCommand(t *testing.T, command string, want Verdict) {
	t.Helper()
	payload, err := json.Marshal(map[string]any{"tool_input": map[string]string{"command": command}})
	if err != nil {
		t.Fatal(err)
	}
	got := Evaluate(payload)
	if got.Verdict != want {
		t.Fatalf("verdict=%s want=%s; statement %d (%s): %s", got.Verdict, want, got.Statement, got.Verb, got.Reason)
	}
	if binary := os.Getenv("ACHTA_HOOK_BINARY"); binary != "" {
		cmd := exec.Command(binary, "hook", "cd")
		cmd.Stdin = bytes.NewReader(payload)
		var stdout, stderr bytes.Buffer
		cmd.Stdout, cmd.Stderr = &stdout, &stderr
		err := cmd.Run()
		wantCode := 0
		if want == Deny {
			wantCode = 2
		}
		if cmd.ProcessState == nil || cmd.ProcessState.ExitCode() != wantCode || stdout.Len() != 0 || (want == Allow && stderr.Len() != 0) {
			t.Fatalf("installed hook want exit=%d stdout empty: error=%v stdout=%q stderr=%q", wantCode, err, stdout.String(), stderr.String())
		}
	}
}

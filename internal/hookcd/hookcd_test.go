package hookcd

import (
	"encoding/json"
	"testing"
)

// RULE: HOOK-CD-WORKDIR-1
func TestVerdictTable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		command string
		want    Verdict
		verb    string
	}{
		{name: "unknown verb is not a write", command: "deploy --dry-run", want: Allow},
		{name: "git status is read only", command: "git status --short", want: Allow},
		{name: "go test is read only", command: "go test ./...", want: Allow},
		{name: "gofmt diff is read only", command: "gofmt -d main.go", want: Allow},
		{name: "sed without in place is read only", command: "sed -n '1,5p' README.md", want: Allow},
		{name: "rulefloor check is read only", command: "rulefloor check --repo .", want: Allow},
		{name: "achta mirror is read only", command: "achta mirror check --master a --mirror b", want: Allow},
		{name: "quoted separators are data", command: "printf 'make; git commit'", want: Allow},
		{name: "heredoc body is data", command: "cat <<'EOF'\nmake verify\ngit commit -m hidden\nEOF\necho done", want: Allow},
		{name: "multiple heredoc bodies are data", command: "cat <<ONE <<'TWO'\nmake verify\nONE\ngit commit -m hidden\nTWO\necho done", want: Allow},
		{name: "absolute first cd covers make", command: "cd /repo && make verify", want: Allow},
		{name: "tilde first cd covers git", command: "cd ~/repo; git commit -m ok", want: Allow},
		{name: "home first cd covers go mod", command: "cd $HOME/repo\ngo mod tidy", want: Allow},
		{name: "braced home first cd covers touch", command: "cd ${HOME}/repo && touch marker", want: Allow},
		{name: "cd option is not the exact first-statement form", command: "cd -- /repo && make verify", want: Deny, verb: "make"},
		{name: "git absolute C covers write", command: "git -C /repo commit -m ok", want: Allow},
		{name: "make absolute C covers write", command: "make -C /repo verify", want: Allow},
		{name: "make without cd denies", command: "make verify", want: Deny, verb: "make"},
		{name: "git commit denies", command: "git commit -m nope", want: Deny, verb: "git commit"},
		{name: "go mod denies", command: "go mod tidy", want: Deny, verb: "go mod"},
		{name: "go get denies", command: "go get example.com/mod", want: Deny, verb: "go get"},
		{name: "go install denies", command: "go install example.com/tool", want: Deny, verb: "go install"},
		{name: "go generate denies", command: "go generate ./...", want: Deny, verb: "go generate"},
		{name: "gofmt write denies", command: "gofmt -w main.go", want: Deny, verb: "gofmt -w"},
		{name: "sed in place denies", command: "sed -i.bak 's/a/b/' file", want: Deny, verb: "sed -i"},
		{name: "mv denies", command: "mv a b", want: Deny, verb: "mv"},
		{name: "cp denies", command: "cp a b", want: Deny, verb: "cp"},
		{name: "rm denies", command: "rm file", want: Deny, verb: "rm"},
		{name: "mkdir denies", command: "mkdir out", want: Deny, verb: "mkdir"},
		{name: "touch denies", command: "touch marker", want: Deny, verb: "touch"},
		{name: "tee in pipeline denies", command: "printf x | tee out", want: Deny, verb: "tee"},
		{name: "truncate redirect denies", command: "printf x > out", want: Deny, verb: ">"},
		{name: "append redirect denies", command: "printf x >> out", want: Deny, verb: ">>"},
		{name: "stderr to stdout duplication allows", command: "go test ./... 2>&1 | tail", want: Allow},
		{name: "dev null and fd duplication allow", command: "cmd >/dev/null 2>&1", want: Allow},
		{name: "stdout to stderr duplication allows", command: "echo x >&2", want: Allow},
		{name: "combined dev null redirect allows", command: "cmd &>/dev/null", want: Allow},
		{name: "numbered stderr file denies", command: "cmd 2>err.log", want: Deny, verb: ">"},
		{name: "combined output file denies", command: "cmd &>out", want: Deny, verb: ">"},
		{name: "numbered append file denies", command: "printf x 1>>log", want: Deny, verb: ">>"},
		{name: "first cd covers mixed redirect and tee", command: "cd /r && make verify 2>&1 | tee log", want: Allow},
		{name: "or separates statements", command: "true || touch marker", want: Deny, verb: "touch"},
		{name: "rulefloor rehash denies", command: "rulefloor rehash RULE --repo .", want: Deny, verb: "rulefloor rehash"},
		{name: "achta writer denies", command: "achta wiki pin repo --sha deadbeef", want: Deny, verb: "achta wiki pin"},
		{name: "relative cd does not count", command: "cd repo && make verify", want: Deny, verb: "make"},
		{name: "cd must be first", command: "pwd; cd /repo; make verify", want: Deny, verb: "make"},
		{name: "assignment before cd defeats first statement", command: "MODE=ci; cd /repo; make verify", want: Deny, verb: "make"},
		{name: "assignment before cd in one statement defeats it", command: "MODE=ci cd /repo && make verify", want: Deny, verb: "make"},
		{name: "relative C does not count", command: "git -C repo commit -m nope", want: Deny, verb: "git commit"},
		{name: "every writing statement needs C", command: "make -C /one verify; git commit -m nope", want: Deny, verb: "git commit"},
		{name: "every writing statement has C", command: "make -C /one verify; git -C /two commit -m ok", want: Allow},
		{name: "heredoc opener redirect still writes", command: "cat > out <<EOF\nmake verify\nEOF", want: Deny, verb: ">"},
		{name: "unterminated heredoc warns", command: "cat <<EOF\nmake verify", want: Warn},
		{name: "unclosed quote warns", command: "make 'verify", want: Warn},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			payload, err := json.Marshal(map[string]any{
				"tool_name":  "Bash",
				"tool_input": map[string]any{"command": test.command},
			})
			if err != nil {
				t.Fatal(err)
			}
			got := Evaluate(payload)
			if got.Verdict != test.want {
				t.Fatalf("verdict = %q, want %q; result = %#v", got.Verdict, test.want, got)
			}
			if test.verb != "" && got.Verb != test.verb {
				t.Fatalf("verb = %q, want %q", got.Verb, test.verb)
			}
		})
	}
}

func TestCommittedGitWriteVerbTable(t *testing.T) {
	t.Parallel()

	want := []string{
		"add", "am", "apply", "bisect", "branch", "checkout", "cherry-pick", "clean", "clone",
		"commit", "config", "fetch", "gc", "init", "merge", "mv", "notes", "pull", "push",
		"rebase", "remote", "reset", "restore", "revert", "rm", "stash", "submodule", "switch",
		"tag", "worktree",
	}
	if len(gitWriteVerbs) != len(want) {
		t.Fatalf("git write table has %d entries, want %d", len(gitWriteVerbs), len(want))
	}
	for _, verb := range want {
		if _, ok := gitWriteVerbs[verb]; !ok {
			t.Fatalf("git write table is missing %q", verb)
		}
	}
}

func TestMalformedPreToolUseInputWarnsAndAllows(t *testing.T) {
	t.Parallel()

	for _, payload := range [][]byte{
		[]byte("not json"),
		[]byte(`{"tool_input":{"command":42}}`),
		[]byte(`{"tool_input":{}}`),
		[]byte(`{"tool_input":{"command":"make"}} trailing`),
	} {
		if got := Evaluate(payload); got.Verdict != Warn {
			t.Fatalf("Evaluate(%q) = %#v, want warning allowance", payload, got)
		}
	}
}

package hookcd

import (
	"encoding/json"
	"strings"
	"testing"
)

// RULE: HOOK-CD-FALSE-POSITIVE-1
// Every Allow row is a command the hook once denied although it could not
// write into a repository (TOOL-FAULTS-1103 ACH-02). Every Deny row is a
// neighbour that can write and must stay denied.
func TestFalsePositiveTable(t *testing.T) {
	tests := []struct {
		name, command string
		want          Verdict
	}{
		// Read-only git listing forms.
		{"tag contains", "git tag --contains abc123", Allow},
		{"tag contains equals", "git tag --contains=abc123", Allow},
		{"tag list pattern", "git tag --list 'v0.5.*'", Allow},
		{"tag short list", "git tag -l", Allow},
		{"tag points-at", "git tag --points-at HEAD", Allow},
		{"branch contains", "git branch --contains abc123", Allow},
		{"branch remote contains", "git branch -r --contains abc123", Allow},
		{"branch all merged", "git branch -a --merged main", Allow},
		{"branch show-current", "git branch --show-current", Allow},
		{"branch list pattern", "git branch --list 'fix/*'", Allow},
		{"branch verbose listing", "git branch -vv", Allow},
		{"branch all listing", "git branch -a", Allow},
		{"absolute C branch contains", "git -C /abs/repo branch --contains abc123", Allow},
		{"absolute C tag contains", "git -C /abs/repo tag --contains abc123", Allow},
		{"other repository read-only listing", "cd /abs/repo && git -C /abs/other branch --show-current", Allow},
		{"cd branch show-current", "cd /abs/repo && git branch --show-current", Allow},
		{"cd tag contains piped", "cd /abs/repo && git tag --contains abc123 | head -3", Allow},
		{"git option before verb", "git -c core.pager=cat branch --show-current", Allow},

		// Absolute scratch targets outside any repository.
		{"redirect to tmp", "cd /abs/repo && go test ./... > /tmp/out.txt 2>&1", Allow},
		{"append to private tmp", "cd /abs/repo && go test ./... >> /private/tmp/out.txt", Allow},
		{"var tmp", "cd /abs/repo && printf x > /var/tmp/f", Allow},
		{"mkdir in tmp", "cd /abs/repo && mkdir -p /tmp/work/x", Allow},
		{"cp into tmp", "cd /abs/repo && cp go.mod /tmp/go.mod.bak", Allow},
		{"cp target directory in tmp", "cd /abs/repo && cp -t /tmp/dir go.mod", Allow},
		{"tee in tmp", "cd /abs/repo && printf x | tee /tmp/f", Allow},
		{"rm in tmp", "cd /abs/repo && rm -rf /tmp/work", Allow},
		{"gofmt write inside anchor", "cd /abs/repo && gofmt -w internal/x.go", Allow},
		{"gofmt write then redirected test", "cd /abs/repo && gofmt -w internal/x.go && go test ./... > /tmp/o 2>&1", Allow},
		{"gofmt list to tmp", "cd /abs/repo && gofmt -l . > /tmp/fmt.txt", Allow},
		{"go install into tmp", "cd /abs/repo && GOBIN=/tmp/bin go install example.com/tool@v0.5.19", Allow},
		{"go install into tmp without cd", "GOBIN=/tmp/bin go install example.com/tool@v0.5.19", Allow},
		{"go install into quoted private tmp", "GOBIN=\"/private/tmp/bin\" go install example.com/tool@v1", Allow},
		{"go install into tmp through env", "cd /abs/repo && env GOBIN=/tmp/bin go install example.com/tool@v1", Allow},

		// Writers that stay denied.
		{"branch create", "git branch newbranch", Deny},
		{"branch create at revision", "git branch newbranch abc123", Deny},
		{"branch force delete", "git branch -D old", Deny},
		{"branch delete long", "git branch --delete old", Deny},
		{"branch move", "git branch -m old new", Deny},
		{"branch force", "git branch -f x HEAD", Deny},
		{"branch upstream", "git branch --set-upstream-to=origin/main", Deny},
		{"branch listing hides delete", "git branch --contains abc123 --delete x", Deny},
		{"branch show-current with operand", "git branch --show-current extra", Deny},
		{"branch listing then create", "git branch --contains abc123; git branch newbranch", Deny},
		{"branch listing redirect", "git branch --show-current > out", Deny},
		{"branch listing tee", "git tag --contains abc123 | tee out", Deny},
		{"tag create", "git tag v1.0.0", Deny},
		{"tag create at revision", "git tag v1.0.0 abc123", Deny},
		{"tag annotated", "git tag -a v1.0.0 -m msg", Deny},
		{"tag delete", "git tag -d v1.0.0", Deny},
		{"tag force", "git tag -f v1.0.0", Deny},
		{"tag listing hides delete", "git tag --list -d v1.0.0", Deny},
		{"tag contains with operand and force", "git tag --contains abc123 -f v1", Deny},
		{"worktree add stays denied", "git worktree add /abs/wt -b x", Deny},
		{"scratch sibling prefix", "cd /abs/repo && printf x > /tmpfoo/f", Deny},
		{"scratch traversal", "cd /abs/repo && printf x > /tmp/../abs/other/f", Deny},
		{"scratch variable component", "cd /abs/repo && printf x > /tmp/$NAME/f", Deny},
		{"scratch root itself", "cd /abs/repo && printf x > /tmp", Deny},
		{"scratch then other repository", "cd /abs/repo && printf x > /tmp/f > /abs/other/f", Deny},
		{"scratch beside other repository target", "cd /abs/repo && touch /abs/other/f /tmp/ok", Deny},
		{"scratch rm root", "cd /abs/repo && rm -rf /tmp", Deny},
		{"scratch copy target traversal", "cd /abs/repo && cp -t /tmp/d/../../abs/other go.mod", Deny},
		{"scratch is not a git selector", "cd /abs/repo && git -C /tmp/wt commit -m x", Deny},
		{"scratch is not a make selector", "cd /abs/repo && make -C /tmp/wt verify", Deny},
		{"go install without GOBIN", "go install example.com/tool@v1", Deny},
		{"go install GOBIN other repository", "GOBIN=/abs/other go install example.com/tool", Deny},
		{"go install GOBIN traversal", "GOBIN=/tmp/../abs/other go install example.com/tool", Deny},
		{"go install GOBIN variable", "GOBIN=$BIN go install example.com/tool", Deny},
		{"go install GOBIN is separate statement", "GOBIN=/tmp/bin; go install example.com/tool", Deny},
		{"go install GOBIN is data", "go install example.com/tool GOBIN=/tmp/bin", Deny},
		{"go install may rewrite go.mod", "GOBIN=/tmp/bin go install -mod=mod ./cmd/x", Deny},
		{"go install modfile", "GOBIN=/tmp/bin go install -modfile=x.mod ./cmd/x", Deny},
		{"branch listing value looks like option", "git branch --contains --delete x", Deny},
		{"go get with scratch GOBIN", "GOBIN=/tmp/bin go get example.com/x", Deny},
		{"go mod with scratch GOBIN", "GOBIN=/tmp/bin go mod tidy", Deny},
		{"variable redirect", "cd /abs/repo && printf x > $OUT", Deny},
		{"variable mkdir", "mkdir -p $DIR/x", Deny},
		{"variable worktree", "git worktree add $WT -b x", Deny},
		{"variable gofmt target", "cd /abs/repo && gofmt -w $FILE", Deny},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) { assertHookCommand(t, test.command, test.want) })
	}
}

// RULE: HOOK-CD-DENY-MESSAGE-1
// A denial caused by a shell variable names that variable and says to write
// the literal absolute path. It never echoes the surrounding arguments
// (HOOK-CD-RETRY-1), so a secret in the statement cannot reach the log.
func TestDenyNamesVariableAndSuggestsLiteralForm(t *testing.T) {
	tests := []struct {
		name, command, variable string
	}{
		{"variable redirect", "cd /abs/repo && printf SECRET1 > $OUT", "$OUT"},
		{"variable mkdir", "pwd; mkdir -p $DIR/SECRET1", "$DIR"},
		{"variable git worktree", "git worktree add $WT -b SECRET1", "$WT"},
		{"braced variable", "printf SECRET1 > ${OUT}/f", "${OUT}"},
		{"home prefix is literal, next variable is named", "printf SECRET1 > $HOME/$SUB/f", "$SUB"},
		{"relative redirect has no variable hint", "printf SECRET1 > out", ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			payload, err := json.Marshal(map[string]any{"tool_input": map[string]string{"command": test.command}})
			if err != nil {
				t.Fatal(err)
			}
			got := Evaluate(payload)
			if got.Verdict != Deny {
				t.Fatalf("verdict = %s, want deny", got.Verdict)
			}
			if (test.variable != "") != strings.Contains(got.Hint, "literal absolute path") || !strings.Contains(got.Hint, test.variable) {
				t.Fatalf("hint = %q, want the variable %q and the literal-path advice", got.Hint, test.variable)
			}
			if strings.Contains(got.Hint+got.Reason, "SECRET1") {
				t.Fatalf("denial echoes a command argument: hint=%q reason=%q", got.Hint, got.Reason)
			}
		})
	}
}

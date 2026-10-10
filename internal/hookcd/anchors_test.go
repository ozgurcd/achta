package hookcd

import "testing"

// RULE: HOOK-CD-ANCHORS-1
func TestAnchoredWriteChains(t *testing.T) {
	tests := []struct {
		name, command string
		want          Verdict
	}{
		{"one line absolute redirect", "cd /abs/repo && cat > /abs/repo/file", Allow},
		{"stripped cd absolute redirect", "cat > /abs/repo/file", Allow},
		{"one line claim", "cd /abs/repo && achta claim take --repo /abs/repo --slice X", Allow},
		{"stripped cd claim", "achta claim take --repo /abs/repo --slice X", Allow},
		{"repo equals form", "achta claim take --repo=/abs/repo --slice X", Allow},
		{"relative repo under cd", "cd /abs/repo && achta claim take --repo . --slice X", Allow},
		{"relative rulefloor repo under cd", "cd /abs/repo && rulefloor rehash RULE --repo .", Allow},
		{"make trailing selector", "make verify -C /abs/repo", Allow},
		{"export chain", "cd /abs/repo\nexport GIT_OPTIONAL_LOCKS=0 && git fetch origin", Allow},
		{"export git selector", "export GIT_OPTIONAL_LOCKS=0 && git -C /abs/repo fetch origin", Allow},
		{"heredoc after cd", "cd /abs/repo\ncat > tasks/todo.md <<'EOF'\nmake verify\nEOF", Allow},
		{"stripped cd absolute heredoc", "cat > /abs/repo/tasks/todo.md <<'EOF'\nmake verify\nEOF", Allow},
		{"absolute direct targets", "touch /abs/repo/a /abs/repo/b; printf x | tee /abs/repo/out", Allow},
		{"absolute cp destination", "cp relative-input /abs/repo/out", Allow},
		{"claim note is data", "achta claim take --repo /abs/repo --slice X --note -h", Allow},
		{"claim reason is data", "achta claim release --repo /abs/repo --slice X --force --reason --help", Allow},
		{"other repository formatter", "cd /abs/repo && gofmt -w /abs/other/file", Deny},
		{"other repository sed", "cd /abs/repo && sed -i 's/a/b/' /abs/other/file", Deny},
		{"copy target option is relative", "cp -t relative-dir /abs/repo/input", Deny},
		{"copy target option escapes", "cd /abs/repo && cp --target-directory=/abs/other /abs/repo/input", Deny},
		{"quoted git selector value", "git -c alias.name=-C commit -m /abs/repo", Deny},
		{"relative heredoc stays denied", "cat > tasks/todo.md <<'EOF'\nmake verify\nEOF", Deny},
		{"relative cd", "cd repo && touch file", Deny},
		{"relative cd loses anchor", "cd /abs/repo; cd child; touch file", Deny},
		{"other repository redirect", "cd /abs/repo && cat > /abs/other/file", Deny},
		{"other repository claim", "cd /abs/repo && achta claim take --repo /abs/other --slice X", Deny},
		{"other repository git", "cd /abs/repo && git -C /abs/other fetch origin", Deny},
		{"other repository direct target", "cd /abs/repo && touch /abs/other/file", Deny},
		{"anchor prefix sibling", "cd /abs/repo && touch /abs/repo-other/file", Deny},
		{"relative traversal", "cd /abs/repo && touch ../other/file", Deny},
		{"absolute traversal", "cd /abs/repo && touch /abs/repo/../other/file", Deny},
		{"cd to other repository", "cd /abs/repo; cd /abs/other; touch file", Deny},
		{"pipeline writer lacks selector", "git -C /abs/repo fetch origin | tee out", Deny},
		{"relative redirect ignores git selector", "git -C /abs/repo fetch origin > out", Deny},
		{"each redirect needs anchor", "cat > /abs/repo/out 2>err", Deny},
		{"option value cannot anchor", "achta decision add --title '--repo /abs/repo'", Deny},
		{"echoed selector cannot anchor", "echo -C /abs/repo | tee out", Deny},
		{"relative repo selector", "achta claim take --repo repo --slice X", Deny},
		{"later relative git selector", "git -C /abs/repo -C child fetch origin", Deny},
		{"relative move source writes", "mv relative-source /abs/repo/out", Deny},
		{"all direct targets anchored", "touch /abs/repo/a relative-b", Deny},
		{"variable target is not literal", "cd /abs/repo && touch $OTHER/file", Deny},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) { assertHookCommand(t, test.command, test.want) })
	}
}

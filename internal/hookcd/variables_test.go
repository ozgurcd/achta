package hookcd

import (
	"encoding/json"
	"strings"
	"testing"
)

func commandPayload(t *testing.T, command string) []byte {
	t.Helper()
	payload, err := json.Marshal(map[string]any{"tool_input": map[string]string{"command": command}})
	if err != nil {
		t.Fatal(err)
	}
	return payload
}

// RULE: HOOK-CD-VARIABLE-1
// A variable is expanded only when its assignment is an unconditional,
// top-level, simple statement that runs before the use, nothing later touches
// it, and the use is double-quoted (TOOL-FAULTS-1103 ACH-05). Every Allow row
// is a command the hook once denied although the shell uses the same literal
// path. Every Deny row is a neighbour whose value the shell may not share with
// the hook; the independent review of the first fix (ACHTA-ACH05-REVIEW-1104)
// found bypasses F1-F3, which are the "review" rows.
func TestVariableExpansionTable(t *testing.T) {
	tests := []struct {
		name, command string
		want          Verdict
	}{
		// Literal assignment, then a double-quoted use.
		{"git -C variable", `D=/abs/repo; git -C "$D" add .`, Allow},
		{"git -C braced variable", `D=/abs/repo && git -C "${D}" commit -m x`, Allow},
		{"make -C variable", "R=/abs/repo\nmake -C \"$R\" verify", Allow},
		{"achta --repo variable", `R=/abs/repo; achta claim take --repo "$R"`, Allow},
		{"git -C variable subdirectory", `D=/abs/repo; git -C "$D/sub" add .`, Allow},
		{"git -C variable then suffix", `D=/abs/repo; git -C "$D"/sub add .`, Allow},
		{"redirect to variable under cd", "cd /abs/repo\nD=/abs/repo/out; echo hi >> \"$D/file\"", Allow},
		{"redirect to scratch variable", `D=/tmp/work; echo hi > "$D/f"`, Allow},
		{"mkdir and tee in a scratch variable", `D=/tmp/w; mkdir -p "$D/x"; echo hi | tee "$D/x/f"`, Allow},
		{"quoted assignment", `D="/abs/repo"; git -C "$D" add .`, Allow},
		{"single quoted assignment", `D='/abs/repo'; git -C "$D" add .`, Allow},
		{"export assignment is not tracked", `export D=/abs/repo; git -C "$D" add .`, Deny},
		{"chained assignment", `A=/abs; B="$A/repo"; git -C "$B" add .`, Allow},
		{"variable inside the cd anchor via selector", "cd /abs/repo\nD=/abs/repo/wt; git -C \"$D\" add .", Allow},
		{"assignment after cd and-list is conditional", `cd /abs/repo && D=/abs/repo/wt; git -C "$D" add .`, Deny},
		{"reassignment uses the latest value", `D=/abs/one; D=/abs/two; git -C "$D" add .`, Allow},
		{"variable followed by a non-name character", `D=/abs/repo; git -C "$D/." add .`, Allow},
		{"assignment on its own line", "D=/abs/repo\ngit -C \"$D\" add .", Allow},
		{"unrelated tail after use", `D=/abs/repo; git -C "$D" add .; echo done`, Allow},

		// nohup background launches.
		{"nohup make with a variable selector", `R=/abs/repo; nohup make -C "$R" verify > /tmp/log 2>&1 &`, Allow},
		{"nohup absolute selector and scratch redirect", "nohup make -C /abs/repo verify > /tmp/log 2>&1 &", Allow},
		{"nohup under cd", "cd /abs/repo && nohup make verify > /tmp/log 2>&1 &", Allow},
		{"nohup with a scratch variable redirect", `L=/tmp/log; nohup make -C /abs/repo verify > "$L" 2>&1 &`, Allow},
		{"nohup without selector or cd stays denied", "nohup make verify > /tmp/log 2>&1 &", Deny},
		{"nohup relative redirect stays denied", "nohup make -C /abs/repo verify > log 2>&1 &", Deny},
		{"nohup with a relative selector stays denied", "nohup make -C repo verify > /tmp/log 2>&1 &", Deny},

		// Unquoted use is word-split, globbed and dropped when empty by the shell.
		{"unquoted use", `D=/abs/repo; git -C $D add .`, Deny},
		{"unquoted braced use", `D=/abs/repo; git -C ${D} add .`, Deny},
		{"unquoted redirect use", `D=/tmp/work; echo hi > $D/f`, Deny},
		{"unquoted scratch touch", `D=/tmp/f; touch $D`, Deny},
		{"single quoted variable is literal text", `D=/abs/repo; touch '$D/f'`, Deny},
		{"escaped variable is literal text", `D=/abs/repo; touch \$D/f`, Deny},
		{"escaped variable inside double quotes is literal text", `D=/abs/repo; touch "\$D/f"`, Deny},

		// Protection stays.
		{"unassigned variable", `git -C "$D" add .`, Deny},
		{"variable assigned in another call is unknown", `cd /abs/repo && echo hi >> "$OUT/file"`, Deny},
		{"assignment from another variable that is unknown", `D="$X"; git -C "$D" add .`, Deny},
		{"unquoted assignment from a variable", `A=/abs; B=$A/repo; git -C "$B" add .`, Deny},
		{"assignment from a relative path", `D=repo; git -C "$D" add .`, Deny},
		{"home assignment is not expanded", "cd /abs/repo\nD=$HOME/work; touch \"$D/f\"", Deny},
		{"tilde assignment is not expanded", "cd /abs/repo\nD=~/work; touch \"$D/f\"", Deny},
		{"single quoted tilde assignment is literal", `D='~/work'; touch "$D/f"`, Deny},
		{"assignment to a glob", `D=/abs/*; git -C "$D" add .`, Deny},
		{"assignment with whitespace", `D='/abs/a b'; git -C "$D" add .`, Deny},
		{"variable outside the anchor", `cd /abs/repo && D=/abs/other; git -C "$D" add .`, Deny},
		{"variable traversal out of the anchor", `cd /abs/repo && D=/abs/repo/..; git -C "$D/other" add .`, Deny},
		{"variable redirect outside the anchor", "cd /abs/repo\nD=/abs/other; echo hi > \"$D/f\"", Deny},
		{"variable that is a scratch root as file", "cd /abs/repo\nD=/tmp; echo hi > \"$D\"", Deny},
		{"scratch variable is not a git selector", `cd /abs/repo && D=/tmp/wt; git -C "$D" commit -m x`, Deny},
		{"variable reassigned to an unknown value", `D=/abs/repo; D="$X"; git -C "$D" add .`, Deny},
		{"variable reassigned to a relative path", `D=/abs/repo; D=rel; git -C "$D" add .`, Deny},
		{"append assignment invalidates", `D=/abs/repo; D+=x; git -C "$D" add .`, Deny},
		{"array element assignment invalidates", `D=/abs/repo; D[0]=x; git -C "$D" add .`, Deny},
		{"unset invalidates", `D=/abs/repo; unset D; git -C "$D" add .`, Deny},
		{"read invalidates", `D=/abs/repo; read D; git -C "$D" add .`, Deny},
		{"eval invalidates", `D=/abs/repo; eval 'D=rel'; git -C "$D" add .`, Deny},
		{"source invalidates", `D=/abs/repo; source ./env.sh; git -C "$D" add .`, Deny},
		{"dot source invalidates", `D=/abs/repo; . ./env.sh; git -C "$D" add .`, Deny},
		{"command dot source invalidates", `D=/abs/repo; command . ./env.sh; git -C "$D" add .`, Deny},
		{"export invalidates", `D=/abs/repo; export D=rel; git -C "$D" add .`, Deny},
		{"declare invalidates", `D=/abs/repo; declare D=rel; git -C "$D" add .`, Deny},
		{"for loop invalidates", `D=/abs/repo; for D in a b; git -C "$D" add .`, Deny},
		{"printf -v invalidates", `D=/abs/repo; printf -v D rel; git -C "$D" add .`, Deny},
		{"assignment expansion invalidates", `D=/abs/repo; : "${D:=x}"; git -C "$D" add .`, Deny},
		{"prefix assignment is not a shell variable", `D=/abs/repo git -C "$D" add .`, Deny},
		{"assignment in a pipeline is a subshell", `echo x | D=/abs/repo; git -C "$D" add .`, Deny},
		{"assignment in a subshell", `(D=/abs/repo); git -C "$D" add .`, Deny},
		{"assignment in a group", `{ D=/abs/repo; }; git -C "$D" add .`, Deny},
		{"background assignment is a subshell", "D=/abs/repo &\ngit -C \"$D\" add .", Deny},
		{"parameter expansion operator is not expanded", `D=/abs/repo; git -C "${D:-rel}" add .`, Deny},
		{"prefix of a longer name is another variable", `D=/abs/repo; git -C "$DX" add .`, Deny},
		{"HOME is not reassignable", "cd /abs/repo\nHOME=/tmp/x; touch \"$HOME/f\"", Deny},
		{"cd to a variable does not anchor a later relative write", `D=/abs/repo; cd "$D"; touch f`, Deny},
		{"later cd outside the anchor", "cd /abs/repo; touch a; cd /abs/other; touch b", Deny},

		// Review ACHTA-ACH05-REVIEW-1104 F2: an assignment that may not run.
		{"review F2 or-list assignment", `true || D=/tmp/f; git -C "$D" commit -m x`, Deny},
		{"review F2 and-list assignment", `false && D=/Users/x/r; git -C "$D" commit -m x`, Deny},
		{"review F2 assignment after and-list tail", `true && D=/abs/repo && git -C "$D" commit -m x`, Deny},
		{"review F2 assignment before or", `D=/abs/repo || true; git -C "$D" commit -m x`, Deny},
		{"review F2 newline continues an and-list", "true &&\nD=/abs/repo\ngit -C \"$D\" commit -m x", Deny},
		{"review F2 if body", "if false; then\nD=/Users/x/r\nfi\ngit -C \"$D\" commit -m x", Deny},
		{"review F2 if body on one line", `if false; then D=/Users/x/r; fi; git -C "$D" commit -m x`, Deny},
		{"review F2 else body", "if true; then :; else\nD=/abs/repo\nfi\ngit -C \"$D\" commit -m x", Deny},
		{"review F2 while body", "while false; do\nD=/abs/repo\ndone\ngit -C \"$D\" commit -m x", Deny},
		{"review F2 case body", "case x in y) D=/abs/repo;; esac\ngit -C \"$D\" commit -m x", Deny},
		{"review F2 function body", "f() {\nD=/Users/x/r\n}\ngit -C \"$D\" commit -m x", Deny},
		{"review F2 function keyword body", "function f\nD=/abs/repo\ngit -C \"$D\" commit -m x", Deny},
		{"review F2 empty cd under the leading cd", "cd /Users/x/r\ntrue || D=/Users/x/r/sub\ncd \"$D\"\ngit commit -m x", Deny},
		{"review F2 empty cd unquoted", "cd /Users/x/r\ntrue || D=/Users/x/r/sub\ncd $D\ngit commit -m x", Deny},

		// Review F1: IFS splits an unquoted expansion.
		{"review F1 IFS before assignment", `IFS=/; D=/tmp/f; touch $D`, Deny},
		{"review F1 IFS after assignment", `D=/tmp/f; IFS=/; touch $D`, Deny},
		{"review F1 IFS with a leading cd", "cd /Users/x/r\nIFS=/\nD=/tmp/f\ntouch $D", Deny},
		{"review F1 IFS and a quoted use", `IFS=/; D=/abs/repo; git -C "$D" commit -m x`, Deny},
		{"review F1 IFS in env prefix", `IFS=/ true; D=/tmp/f; touch $D`, Deny},
		{"review F1 IFS unset", `D=/tmp/f; unset IFS; touch $D`, Deny},

		// Review F3: arithmetic reassignment.
		{"review F3 arithmetic command", `D=/abs/repo; (( D = 5 )); git -C "$D" commit -m x`, Deny},
		{"review F3 arithmetic command unspaced", `D=/abs/repo; ((D=5)); git -C $D commit -m x`, Deny},
		{"review F3 arithmetic quoted use", `D=/abs/repo; ((D=5)); git -C "$D" commit -m x`, Deny},
		{"review F3 let", `D=/abs/repo; let D=5; git -C "$D" commit -m x`, Deny},
		{"review F3 let unquoted", `D=/abs/repo; let D=5; git -C $D commit -m x`, Deny},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) { assertHookCommand(t, test.command, test.want) })
	}
}

// The table is fail-closed on a construct that can run or change a variable
// out of order: once one appears, nothing expands for the rest of the command,
// even a variable assigned later.
func TestVariableTableStaysOffAfterAConstruct(t *testing.T) {
	for name, command := range map[string]string{
		"if":         "if false; then :; fi\nD=/abs/repo\ngit -C \"$D\" commit -m x",
		"arithmetic": "((x=1))\nD=/abs/repo\ngit -C \"$D\" commit -m x",
		"IFS":        "IFS=/\nD=/abs/repo\ngit -C \"$D\" commit -m x",
		"eval":       "eval :\nD=/abs/repo\ngit -C \"$D\" commit -m x",
		"function":   "f() { :; }\nD=/abs/repo\ngit -C \"$D\" commit -m x",
	} {
		t.Run(name, func(t *testing.T) { assertHookCommand(t, command, Deny) })
	}
}

// Exponential self-expansion must be refused rather than expanded. The limits
// are a Limits value, so tests inject small ones.
func TestVariableExpansionIsBounded(t *testing.T) {
	var command strings.Builder
	command.WriteString("A=/abs/repo")
	for index := 0; index < 40; index++ {
		command.WriteString(`; A="$A$A"`)
	}
	command.WriteString(`; git -C "$A" add .`)
	if got := Evaluate(commandPayload(t, command.String())); got.Verdict != Deny {
		t.Fatalf("doubling expansion verdict = %s, want deny", got.Verdict)
	}
	small := Limits{MaxVariables: 2, MaxValueBytes: 16}
	for name, test := range map[string]struct {
		command string
		want    Verdict
	}{
		"within limits":      {`A=/abs/r; git -C "$A" add .`, Allow},
		"value too long":     {`A=/abs/repo/long/dir; git -C "$A" add .`, Deny},
		"expanded too long":  {`A=/abs/r; B="$A/aaaaaaaaaa"; git -C "$B" add .`, Deny},
		"too many variables": {`A=/abs/a; B=/abs/b; C=/abs/c; git -C "$A" add .`, Deny},
		"reassign is free":   {`A=/abs/a; A=/abs/b; A=/abs/c; git -C "$A" add .`, Allow},
	} {
		t.Run(name, func(t *testing.T) {
			if got := EvaluateWithLimits(commandPayload(t, test.command), small); got.Verdict != test.want {
				t.Fatalf("verdict = %s, want %s (%s)", got.Verdict, test.want, got.Problem)
			}
		})
	}
}

// RULE: HOOK-CD-OWN-SUGGESTION-1
// A denial must not tell the caller to use a form the hook then refuses. A
// -C or --repo selector does not move a shell redirect, and a selector outside
// the leading cd is refused, so those fixes say what is accepted.
func TestFixDoesNotSuggestARefusedForm(t *testing.T) {
	tests := []struct {
		name, command string
		wantFix       []string
		notFix        []string
	}{
		{"relative redirect after an absolute selector", "make -C /abs/wiki postcheck > out.txt", []string{"redirect", "absolute"}, []string{"make -C <abs>"}},
		{"relative tee target", "cd /abs/repo && printf x | tee ../out", []string{"absolute"}, []string{"make -C <abs>"}},
		{"selector outside the leading cd", "cd /abs/repo && git -C /abs/other commit -m x", []string{"inside the leading cd", "own command"}, []string{"git -C <abs>"}},
		{"later cd leaves the leading cd", "cd /abs/repo; touch a; cd /abs/other; touch b", []string{"later cd"}, nil},
		{"no anchor at all", "make verify", []string{"git -C <abs>"}, nil},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := Evaluate(commandPayload(t, test.command))
			if got.Verdict != Deny {
				t.Fatalf("verdict = %s, want deny", got.Verdict)
			}
			for _, want := range test.wantFix {
				if !strings.Contains(got.Fix+got.Problem, want) {
					t.Errorf("fix %q problem %q lack %q", got.Fix, got.Problem, want)
				}
			}
			for _, not := range test.notFix {
				if strings.Contains(got.Fix, not) {
					t.Errorf("fix %q suggests the refused form %q", got.Fix, not)
				}
			}
		})
	}
}

// The corrected forms the fixes name are replayed through the hook: each must
// be allowed.
func TestSuggestedCorrectionsAreAllowed(t *testing.T) {
	for _, command := range []string{
		"make -C /abs/wiki postcheck > /tmp/postcheck.txt",
		"cd /abs/wiki\nmake postcheck > out.txt",
		"cd /abs/repo && git -C /abs/repo commit -m x",
		"cd /abs/other && git commit -m x",
	} {
		assertHookCommand(t, command, Allow)
	}
}

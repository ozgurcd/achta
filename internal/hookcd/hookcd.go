package hookcd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"unicode"
)

const (
	Schema     = "achta.hook-cd.v1"
	MaxInput   = 1 << 20
	MaxCommand = 256 << 10

	redirectOutputToken = "\x00>"
	redirectAppendToken = "\x00>>"
)

type Verdict string

const (
	Allow Verdict = "allow"
	Deny  Verdict = "deny"
	Warn  Verdict = "warn"
)

type Result struct {
	Verdict   Verdict
	Statement int
	Verb      string
	Reason    string
	Hint      string
	// Problem says why the denied write is unanchored. It never contains
	// command arguments (HOOK-CD-RETRY-1).
	Problem string
}

var gitWriteVerbs = map[string]struct{}{
	"add": {}, "am": {}, "apply": {}, "bisect": {}, "branch": {},
	"checkout": {}, "cherry-pick": {}, "clean": {}, "clone": {},
	"commit": {}, "config": {}, "fetch": {}, "gc": {}, "init": {},
	"merge": {}, "mv": {}, "notes": {}, "pull": {}, "push": {},
	"rebase": {}, "remote": {}, "reset": {}, "restore": {}, "revert": {},
	"rm": {}, "stash": {}, "submodule": {}, "switch": {}, "tag": {},
	"worktree": {},
}

var achtaWriteCommands = map[string]struct{}{
	"amendments declare": {},
	"amendments rebase":  {},
	"claim take":         {},
	"claim release":      {},
	"decision add":       {},
	"gate run":           {},
	"journal note":       {},
	"parts lock":         {},
	"wiki derive":        {},
	"wiki pin":           {},
	"witness finalize":   {},
	"witness init":       {},
	"witness step":       {},
}

var directWriteVerbs = map[string]struct{}{
	"cp": {}, "make": {}, "mkdir": {}, "mv": {}, "rm": {}, "tee": {}, "touch": {},
}

func Evaluate(payload []byte) Result {
	if len(payload) == 0 || len(payload) > MaxInput {
		return warning("PreToolUse input is empty or exceeds the input bound")
	}
	var envelope struct {
		ToolInput struct {
			Command *string `json:"command"`
		} `json:"tool_input"`
	}
	decoder := json.NewDecoder(bytes.NewReader(payload))
	if err := decoder.Decode(&envelope); err != nil {
		return warning("PreToolUse input is not one JSON document")
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return warning("PreToolUse input contains trailing JSON data")
	}
	if envelope.ToolInput.Command == nil || len(*envelope.ToolInput.Command) > MaxCommand {
		return warning("tool_input.command is missing, not a string, or exceeds the command bound")
	}

	command, err := stripHeredocBodies(*envelope.ToolInput.Command)
	if err != nil {
		return warning(err.Error())
	}
	statements, err := splitStatements(command)
	if err != nil {
		return warning(err.Error())
	}
	if len(statements) == 0 {
		return Result{Verdict: Allow}
	}
	anchor, directory := "", ""
	for index, statement := range statements {
		tokens, err := lex(statement)
		if err != nil {
			return warning(err.Error())
		}
		if index == 0 && firstStatementIsAbsoluteCD(statement) {
			anchor = literalPath(tokens[1])
			directory = anchor
			continue
		}
		if len(tokens) > 0 && tokens[0] == "cd" {
			directory = ""
			if len(tokens) == 2 && anchoredPath(tokens[1], "", anchor) && anchor != "" {
				directory = literalPath(tokens[1])
			}
		}
		if verb, problem := unanchoredWriter(tokens, directory, anchor); verb != "" {
			return Result{
				Verdict:   Deny,
				Statement: index + 1,
				Verb:      verb,
				Reason:    "a write lacks an absolute anchor or escapes the leading cd",
				Hint:      variableHint(tokens),
				Problem:   problem,
			}
		}
	}
	return Result{Verdict: Allow}
}

// variableHint names the first shell variable the hook cannot expand. Only the
// variable name is reported, never the argument around it (HOOK-CD-RETRY-1).
// The home forms are literal to the hook and never reported.
func variableHint(tokens []string) string {
	for _, token := range tokens {
		if name := shellVariable(token); name != "" {
			return fmt.Sprintf("%s is a shell variable the hook cannot expand; write the literal absolute path instead (cd /abs/repo on its own line, git -C /abs/repo, or an absolute target)", name)
		}
	}
	return ""
}

// shellVariable returns the first $NAME or ${NAME} in token other than HOME.
func shellVariable(token string) string {
	for rest := token; ; {
		_, after, found := strings.Cut(rest, "$")
		if !found {
			return ""
		}
		rest = after
		braced := strings.HasPrefix(after, "{")
		name := strings.TrimPrefix(after, "{")
		end := 0
		for end < len(name) && (name[end] == '_' || unicode.IsLetter(rune(name[end])) || unicode.IsDigit(rune(name[end]))) {
			end++
		}
		if end == 0 || name[:end] == "HOME" {
			continue
		}
		if braced {
			return "${" + name[:end] + "}"
		}
		return "$" + name[:end]
	}
}

func warning(reason string) Result {
	return Result{Verdict: Warn, Reason: reason}
}

func stripHeredocBodies(command string) (string, error) {
	lines := strings.Split(command, "\n")
	out := make([]string, 0, len(lines))
	for index := 0; index < len(lines); index++ {
		line := lines[index]
		heredocs, err := heredocDelimiters(line)
		if err != nil {
			return "", err
		}
		out = append(out, line)
		for _, heredoc := range heredocs {
			terminated := false
			for index++; index < len(lines); index++ {
				candidate := lines[index]
				if heredoc.stripTabs {
					candidate = strings.TrimLeft(candidate, "\t")
				}
				if candidate == heredoc.delimiter {
					terminated = true
					break
				}
			}
			if !terminated {
				return "", fmt.Errorf("heredoc body is unterminated")
			}
		}
	}
	return strings.Join(out, "\n"), nil
}

type heredocSpec struct {
	delimiter string
	stripTabs bool
}

func heredocDelimiters(line string) ([]heredocSpec, error) {
	var found []heredocSpec
	quote := byte(0)
	escaped := false
	for index := 0; index < len(line); index++ {
		char := line[index]
		if escaped {
			escaped = false
			continue
		}
		if quote == 0 && char == '\\' {
			escaped = true
			continue
		}
		if quote != 0 {
			if char == quote {
				quote = 0
			}
			continue
		}
		if char == '\'' || char == '"' {
			quote = char
			continue
		}
		if char != '<' || index+1 >= len(line) || line[index+1] != '<' {
			continue
		}
		if index+2 < len(line) && line[index+2] == '<' {
			index += 2
			continue
		}
		cursor := index + 2
		stripTabs := false
		if cursor < len(line) && line[cursor] == '-' {
			stripTabs = true
			cursor++
		}
		for cursor < len(line) && unicode.IsSpace(rune(line[cursor])) {
			cursor++
		}
		if cursor >= len(line) {
			return nil, fmt.Errorf("heredoc delimiter is missing")
		}
		delimiterQuote := byte(0)
		if line[cursor] == '\'' || line[cursor] == '"' {
			delimiterQuote = line[cursor]
			cursor++
		}
		start := cursor
		for cursor < len(line) {
			char = line[cursor]
			if delimiterQuote != 0 {
				if char == delimiterQuote {
					break
				}
			} else if unicode.IsSpace(rune(char)) || strings.ContainsRune(";|&<>", rune(char)) {
				break
			}
			cursor++
		}
		if start == cursor {
			return nil, fmt.Errorf("heredoc delimiter is empty")
		}
		if delimiterQuote != 0 && (cursor >= len(line) || line[cursor] != delimiterQuote) {
			return nil, fmt.Errorf("heredoc delimiter quote is unterminated")
		}
		found = append(found, heredocSpec{delimiter: line[start:cursor], stripTabs: stripTabs})
		if delimiterQuote != 0 {
			cursor++
		}
		index = cursor - 1
	}
	if quote != 0 || escaped {
		return nil, fmt.Errorf("shell statement has an unterminated quote or escape")
	}
	return found, nil
}

func splitStatements(command string) ([]string, error) {
	var statements []string
	var current strings.Builder
	quote := rune(0)
	escaped := false
	comment := false
	runes := []rune(command)
	flush := func() {
		if statement := strings.TrimSpace(current.String()); statement != "" {
			statements = append(statements, statement)
		}
		current.Reset()
	}
	for index := 0; index < len(runes); index++ {
		char := runes[index]
		if comment {
			if char == '\n' {
				comment = false
				flush()
			}
			continue
		}
		if escaped {
			current.WriteRune(char)
			escaped = false
			continue
		}
		if quote == 0 && char == '\\' {
			current.WriteRune(char)
			escaped = true
			continue
		}
		if quote != 0 {
			current.WriteRune(char)
			if char == quote {
				quote = 0
			}
			continue
		}
		if char == '\'' || char == '"' {
			quote = char
			current.WriteRune(char)
			continue
		}
		if char == '`' || (char == '$' && index+1 < len(runes) && runes[index+1] == '(') {
			return nil, fmt.Errorf("shell command substitution is outside the hook parser")
		}
		if char == '#' && (current.Len() == 0 || unicode.IsSpace(previousRune(current.String()))) {
			comment = true
			continue
		}
		if char == '\n' || char == ';' {
			flush()
			continue
		}
		if index+1 < len(runes) && ((char == '&' && runes[index+1] == '&') || (char == '|' && runes[index+1] == '|')) {
			flush()
			index++
			continue
		}
		current.WriteRune(char)
	}
	if quote != 0 || escaped {
		return nil, fmt.Errorf("shell statement has an unterminated quote or escape")
	}
	flush()
	return statements, nil
}

func previousRune(value string) rune {
	runes := []rune(value)
	if len(runes) == 0 {
		return 0
	}
	return runes[len(runes)-1]
}

func lex(statement string) ([]string, error) {
	var tokens []string
	var token strings.Builder
	quote := rune(0)
	escaped := false
	runes := []rune(statement)
	flush := func() {
		if token.Len() > 0 {
			tokens = append(tokens, token.String())
			token.Reset()
		}
	}
	for index := 0; index < len(runes); index++ {
		char := runes[index]
		if escaped {
			token.WriteRune(char)
			escaped = false
			continue
		}
		if quote == 0 && char == '\\' {
			escaped = true
			continue
		}
		if quote != 0 {
			if char == quote {
				quote = 0
			} else {
				token.WriteRune(char)
			}
			continue
		}
		if char == '\'' || char == '"' {
			quote = char
			continue
		}
		if unicode.IsSpace(char) {
			flush()
			continue
		}
		if char == '|' {
			flush()
			tokens = append(tokens, "|")
			continue
		}
		if char == '<' || char == '>' {
			flush()
			operator := string(char)
			if index+1 < len(runes) && runes[index+1] == char {
				operator += string(char)
				index++
			}
			if operator == ">" {
				operator = redirectOutputToken
			} else if operator == ">>" {
				operator = redirectAppendToken
			}
			tokens = append(tokens, operator)
			continue
		}
		token.WriteRune(char)
	}
	if quote != 0 || escaped {
		return nil, fmt.Errorf("shell statement has an unterminated quote or escape")
	}
	flush()
	return tokens, nil
}

func firstStatementIsAbsoluteCD(statement string) bool {
	tokens, err := lex(statement)
	if err != nil || len(tokens) != 2 || tokens[0] != "cd" || contains(tokens, "|") || contains(tokens, ">") || contains(tokens, ">>") {
		return false
	}
	return isAbsoluteSelector(tokens[1])
}

func segmentWritingVerb(tokens []string) string {
	command, args, environment := invocationEnvironment(tokens)
	switch command {
	case "git":
		name, rest := subcommandArguments(args, map[string]bool{"-C": true, "-c": true, "--git-dir": true, "--work-tree": true, "--namespace": true, "--exec-path": true})
		if name == "config" && gitConfigReadOnly(rest) {
			return ""
		}
		if gitListingReadOnly(name, rest) {
			return ""
		}
		if name != "" {
			if _, ok := gitWriteVerbs[name]; ok {
				return "git " + name
			}
		}
	case "go":
		subcommand := subcommand(args, map[string]bool{"-C": true})
		if subcommand == "install" && gobinScratch(environment) && !goModuleFlagPresent(args) {
			return ""
		}
		if subcommand == "mod" || subcommand == "get" || subcommand == "install" || subcommand == "generate" {
			return "go " + subcommand
		}
	case "gofmt":
		if flagPresent(args, "-w") {
			return "gofmt -w"
		}
	case "sed":
		for _, arg := range args {
			if arg == "--in-place" || strings.HasPrefix(arg, "-i") {
				return "sed -i"
			}
		}
	case "rulefloor":
		if subcommand(args, nil) == "rehash" {
			return "rulefloor rehash"
		}
	case "achta":
		if achtaHelpRequested(args) {
			return ""
		}
		if name := achtaCommand(args); name != "" {
			if name == "wiki derive" && !flagPresent(args, "--write") {
				return ""
			}
			if _, ok := achtaWriteCommands[name]; ok {
				return "achta " + name
			}
		}
	default:
		if _, ok := directWriteVerbs[command]; ok {
			return command
		}
	}
	return ""
}

func invocation(tokens []string) (string, []string) {
	command, args, _ := invocationEnvironment(tokens)
	return command, args
}

// invocationEnvironment also returns the VAR=value assignments that precede
// the command, in order, whether bare or after env.
func invocationEnvironment(tokens []string) (string, []string, []string) {
	var environment []string
	index := 0
	for index < len(tokens) {
		base := filepath.Base(tokens[index])
		if isAssignment(tokens[index]) {
			environment = append(environment, tokens[index])
			index++
			continue
		}
		switch base {
		case "command", "nohup", "time", "sudo":
			index++
			continue
		case "env":
			index++
			for index < len(tokens) && (strings.HasPrefix(tokens[index], "-") || isAssignment(tokens[index])) {
				if isAssignment(tokens[index]) {
					environment = append(environment, tokens[index])
				}
				index++
			}
			continue
		}
		return base, tokens[index+1:], environment
	}
	return "", nil, environment
}

func isAssignment(token string) bool {
	name, _, ok := strings.Cut(token, "=")
	if !ok || name == "" {
		return false
	}
	for index, char := range name {
		if !(char == '_' || unicode.IsLetter(char) || (index > 0 && unicode.IsDigit(char))) {
			return false
		}
	}
	return true
}

func subcommand(args []string, valueFlags map[string]bool) string {
	name, _ := subcommandArguments(args, valueFlags)
	return name
}

func subcommandArguments(args []string, valueFlags map[string]bool) (string, []string) {
	for index := 0; index < len(args); index++ {
		arg := args[index]
		if valueFlags[arg] {
			index++
			continue
		}
		if strings.HasPrefix(arg, "-") {
			continue
		}
		return filepath.Base(arg), args[index+1:]
	}
	return "", nil
}

// Only known read actions/modifiers exempt config from the writer table.
// Unknown options and any write action retain the existing write verdict.
func gitConfigReadOnly(args []string) bool {
	readAction, operands := false, 0
	for index := 0; index < len(args); index++ {
		arg, _, hasValue := strings.Cut(args[index], "=")
		switch arg {
		case "--get", "--get-all", "--get-regexp", "--list", "-l":
			readAction = true
		case "--show-origin", "--show-scope", "--global", "--local", "--system", "--worktree",
			"--includes", "--no-includes", "--null", "-z", "--name-only",
			"--bool", "--int", "--bool-or-int", "--path", "--expiry-date":
		case "--file", "-f", "--blob", "--type", "--default":
			if !hasValue {
				index++
				if index >= len(args) {
					return false
				}
			}
		case "--":
			operands += len(args) - index - 1
			return readAction || operands == 1
		default:
			if strings.HasPrefix(arg, "-") {
				return false
			}
			operands++
		}
	}
	return readAction || operands == 1
}

// listingFlags describes the read-only options of a git listing verb. Any
// option outside the table, or an operand without a listing option, keeps the
// write verdict, so `git branch name` and `git tag -d name` stay denied.
type listingFlags struct {
	boolean    map[string]bool // options without a value
	valued     map[string]bool // options whose value is the next argument or =value
	listing    map[string]bool // options that switch the verb to list mode
	shorts     string          // letters allowed in a cluster such as -vv
	noOperands map[string]bool // options that forbid operands
}

var gitListingVerbs = map[string]listingFlags{
	"branch": {
		boolean: map[string]bool{"-a": true, "--all": true, "-r": true, "--remotes": true, "-v": true, "--verbose": true,
			"--list": true, "-l": true, "--show-current": true, "--no-color": true, "--ignore-case": true, "-i": true, "--no-column": true},
		valued:     map[string]bool{"--contains": true, "--no-contains": true, "--merged": true, "--no-merged": true, "--points-at": true, "--sort": true, "--format": true},
		listing:    map[string]bool{"--list": true, "-l": true, "--contains": true, "--no-contains": true, "--merged": true, "--no-merged": true, "--points-at": true, "--show-current": true},
		shorts:     "arvli",
		noOperands: map[string]bool{"--show-current": true},
	},
	"tag": {
		boolean: map[string]bool{"--list": true, "-l": true, "--no-color": true, "--ignore-case": true, "-i": true, "--no-column": true},
		valued:  map[string]bool{"--contains": true, "--no-contains": true, "--merged": true, "--no-merged": true, "--points-at": true, "--sort": true, "--format": true},
		listing: map[string]bool{"--list": true, "-l": true, "--contains": true, "--no-contains": true, "--merged": true, "--no-merged": true, "--points-at": true},
		shorts:  "li",
	},
}

// gitListingReadOnly accepts only explicit listing invocations. A bare
// `git branch` or `git tag` keeps the write verdict.
func gitListingReadOnly(verb string, args []string) bool {
	flags, ok := gitListingVerbs[verb]
	if !ok {
		return false
	}
	seen, listing, forbidden, operands := false, false, false, 0
	for index := 0; index < len(args); index++ {
		arg := args[index]
		if arg == "--" {
			operands += len(args) - index - 1
			break
		}
		if !strings.HasPrefix(arg, "-") {
			operands++
			continue
		}
		name, _, hasValue := strings.Cut(arg, "=")
		switch {
		case flags.valued[name]:
			if !hasValue {
				index++
				// A --sort key may be descending (-committerdate); other
				// values that look like options keep the write verdict.
				if index >= len(args) || (strings.HasPrefix(args[index], "-") && name != "--sort") {
					return false
				}
			}
		case flags.boolean[arg]:
		case !strings.HasPrefix(arg, "--") && len(arg) > 2 && strings.Trim(arg[1:], flags.shorts) == "":
			for _, letter := range arg[1:] {
				listing = listing || flags.listing["-"+string(letter)]
			}
			seen = true
			continue
		default:
			return false
		}
		seen = true
		listing = listing || flags.listing[name]
		forbidden = forbidden || flags.noOperands[name]
	}
	if !seen || (operands > 0 && (!listing || forbidden)) {
		return false
	}
	return true
}

// A go install into a literal scratch GOBIN writes only a binary there.
// Module-changing flags keep the write verdict.
func goModuleFlagPresent(args []string) bool {
	for _, arg := range args {
		name, _, _ := strings.Cut(strings.TrimLeft(arg, "-"), "=")
		if name == "mod" || name == "modfile" {
			return true
		}
	}
	return false
}

// A flag value named --help is data, not a help request. Achta's boolean
// options consume no following value; other options use a value or =value.
func achtaHelpRequested(args []string) bool {
	for index := 0; index < len(args); index++ {
		arg := args[index]
		switch arg {
		case "--":
			return false
		case "--help", "-h":
			return true
		case "--json", "--quiet", "--timing", "--check", "--write", "--force":
		default:
			if strings.HasPrefix(arg, "-") && !strings.Contains(arg, "=") {
				index++
			}
		}
	}
	return false
}

func achtaCommand(args []string) string {
	positionals := make([]string, 0, 2)
	for index := 0; index < len(args); index++ {
		switch args[index] {
		case "--workspace", "--wiki-dir":
			index++
		case "--json", "--quiet", "--timing":
		default:
			if strings.HasPrefix(args[index], "-") {
				continue
			}
			positionals = append(positionals, args[index])
			if len(positionals) == 2 {
				return strings.Join(positionals, " ")
			}
		}
	}
	return strings.Join(positionals, " ")
}

func hasAbsoluteC(tokens []string) bool {
	for index := 0; index+1 < len(tokens); index++ {
		if tokens[index] == "-C" && isAbsoluteSelector(tokens[index+1]) {
			return true
		}
	}
	return false
}

func isAbsoluteSelector(path string) bool {
	return filepath.IsAbs(path) || path == "~" || strings.HasPrefix(path, "~/") || path == "$HOME" || strings.HasPrefix(path, "$HOME/") || path == "${HOME}" || strings.HasPrefix(path, "${HOME}/")
}

func pipelineSegments(tokens []string) [][]string {
	segments := make([][]string, 0, 1)
	start := 0
	for index, token := range tokens {
		if token == "|" {
			segments = append(segments, tokens[start:index])
			start = index + 1
		}
	}
	return append(segments, tokens[start:])
}

func flagPresent(args []string, flag string) bool {
	for _, arg := range args {
		if arg == flag || strings.HasPrefix(arg, flag+"=") {
			return true
		}
	}
	return false
}

func contains(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

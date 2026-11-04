package hookcd

import (
	"path/filepath"
	"strings"
	"unicode"
)

const (
	// DefaultMaxVariables bounds how many distinct variables one command may
	// hold in the table. Past it the hook stops expanding and denies as before.
	DefaultMaxVariables = 64
	// DefaultMaxValueBytes bounds one stored value and, times MaxVariables, the
	// bytes one statement may gain by expansion. It stops $A$A doubling.
	DefaultMaxValueBytes = 4096
)

func (limits Limits) normalized() Limits {
	if limits.MaxVariables <= 0 {
		limits.MaxVariables = DefaultMaxVariables
	}
	if limits.MaxValueBytes <= 0 {
		limits.MaxValueBytes = DefaultMaxValueBytes
	}
	return limits
}

// variables holds the shell variables the hook may expand: those assigned, as
// an unconditional top-level statement of their own, to a literal absolute path
// earlier in the same command. It is lexical and never reads the environment.
// Anything that could make the shell's value differ from the table's, or that
// the hook cannot follow, drops the name or disables the table for the rest of
// the command, so an unknown value is never guessed (HOOK-CD-VARIABLE-1).
type variables struct {
	limits   Limits
	values   map[string]string
	overflow bool
	// disabled is set for good by a construct that can run or change a
	// variable out of order: control flow, functions, subshells, arithmetic,
	// IFS, eval and the like. A disabled table expands nothing.
	disabled bool
}

func newVariables(limits Limits) *variables {
	return &variables{limits: limits.normalized(), values: map[string]string{}}
}

// budget is the most bytes one statement may gain by expansion.
func (v *variables) budget() int { return v.limits.MaxVariables * v.limits.MaxValueBytes }

// expand resolves the $NAME or ${NAME} that starts at runes[index]. It returns
// the value and the index of the reference's last rune. Only an exact
// reference to a table entry resolves; operators such as ${NAME:-x} and names
// that run into a longer name do not.
func (v *variables) expand(runes []rune, index int) (string, int, bool) {
	if v.disabled {
		return "", 0, false
	}
	cursor := index + 1
	braced := cursor < len(runes) && runes[cursor] == '{'
	if braced {
		cursor++
	}
	start := cursor
	for cursor < len(runes) && nameRune(runes[cursor], cursor == start) {
		cursor++
	}
	if cursor == start {
		return "", 0, false
	}
	name := string(runes[start:cursor])
	if braced {
		if cursor >= len(runes) || runes[cursor] != '}' {
			return "", 0, false
		}
		cursor++
	} else if cursor < len(runes) && runes[cursor] > unicode.MaxASCII {
		return "", 0, false
	}
	value, ok := v.values[name]
	return value, cursor - 1, ok
}

func nameRune(char rune, first bool) bool {
	return char == '_' || (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (!first && char >= '0' && char <= '9')
}

// disable empties the table and stops all expansion for the rest of the
// command.
func (v *variables) disable() {
	v.disabled = true
	clear(v.values)
}

// screen runs before a statement is expanded. It disables the table when the
// statement holds a construct whose effect on variables the hook does not
// follow: control flow, grouping, functions, arithmetic, IFS, mutator words and
// ${NAME:=x}. words are the statement's words with no expansion applied.
func (v *variables) screen(statement string, words []string) {
	if v.disabled {
		return
	}
	if (len(words) > 0 && structuralWords[words[0]]) || unquotedStructure(statement) || mutatesVariables(words) {
		v.disable()
		return
	}
	for _, word := range words {
		if strings.Contains(word, "IFS") || assignsByExpansion(word) {
			v.disable()
			return
		}
	}
}

// structuralWords start a compound command or declare a function. The hook
// cannot tell whether the statements that follow run.
var structuralWords = map[string]bool{
	"if": true, "then": true, "else": true, "elif": true, "fi": true,
	"do": true, "done": true, "case": true, "esac": true, "function": true,
	"time": true, "!": true, "[[": true, "]]": true, "coproc": true,
}

// unquotedStructure reports an unquoted, unescaped ( ) { } in the statement:
// subshells, groups, function bodies, arithmetic, process substitution, case
// patterns and brace or ${} expansion.
func unquotedStructure(statement string) bool {
	quote := rune(0)
	escaped := false
	for _, char := range statement {
		switch {
		case escaped:
			escaped = false
		case quote == 0 && char == '\\':
			escaped = true
		case quote != 0:
			if char == quote {
				quote = 0
			}
		case char == '\'' || char == '"':
			quote = char
		case char == '(' || char == ')' || char == '{' || char == '}':
			return true
		}
	}
	return false
}

// assignsByExpansion reports a ${NAME=x} or ${NAME:=x} reference, which assigns
// when NAME is unset. Quoted or not, it is refused.
func assignsByExpansion(word string) bool {
	for rest := word; ; {
		_, after, found := strings.Cut(rest, "${")
		if !found {
			return false
		}
		rest = after
		end := 0
		for end < len(after) && nameRune(rune(after[end]), end == 0) {
			end++
		}
		tail := after[end:]
		if end > 0 && (strings.HasPrefix(tail, "=") || strings.HasPrefix(tail, ":=")) {
			return true
		}
	}
}

// observe updates the table from one statement, after the statement itself
// was checked against the table as it stood before. simple says the statement
// is an unconditional top-level one: neither joined to the previous statement
// by && or ||, nor followed by anything but ; a newline or &&. quoteAt gives
// each word's first quote or escape offset (-1 for none); one before the "="
// makes the word a command name and no assignment.
func (v *variables) observe(tokens []string, quoteAt []int, simple bool) {
	if len(tokens) == 0 || v.disabled {
		return
	}
	allAssignments := simple
	for index, token := range tokens {
		if !isAssignment(token) || (index < len(quoteAt) && quoteAt[index] >= 0 && quoteAt[index] <= strings.Index(token, "=")) {
			allAssignments = false
			break
		}
	}
	if allAssignments {
		for _, token := range tokens {
			name, value, _ := strings.Cut(token, "=")
			v.set(name, value)
		}
		return
	}
	// Any other statement that names a variable as an assignment target may
	// have changed it to a value the hook does not know: forget the name.
	for _, token := range tokens {
		if name := compoundTarget(token); name != "" {
			delete(v.values, name)
		}
	}
}

// set stores name=value when value is a plain absolute path, and otherwise
// forgets name. Reassigning a name never counts twice against the limit.
func (v *variables) set(name, value string) {
	delete(v.values, name)
	if v.overflow || name == "HOME" || !v.storable(value) {
		return
	}
	if len(v.values) >= v.limits.MaxVariables {
		v.overflow = true
		clear(v.values)
		return
	}
	v.values[name] = value
}

// storable accepts only a value the shell would use verbatim as one absolute
// word: plain path characters only, so no whitespace, $, ~, glob, quote,
// separator or redirection.
func (v *variables) storable(value string) bool {
	if value == "" || len(value) > v.limits.MaxValueBytes || !strings.HasPrefix(value, "/") {
		return false
	}
	for _, char := range value {
		if !plainPathRune(char) {
			return false
		}
	}
	literal := literalPath(value)
	return literal != "" && filepath.IsAbs(literal)
}

func plainPathRune(char rune) bool {
	return char == '/' || char == '.' || char == '_' || char == '-' || char == '+' || char == ',' || char == ':' || char == '@' || char == '%' || char == '=' ||
		(char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9')
}

// variableMutators are words that can change a variable outside a plain
// assignment. Any word equal to one disables the table for the rest of the
// command, even as data, so the hook fails closed.
var variableMutators = map[string]bool{
	"eval": true, "source": true, ".": true, "read": true, "unset": true,
	"declare": true, "typeset": true, "local": true, "readonly": true,
	"mapfile": true, "readarray": true, "getopts": true,
	"for": true, "select": true, "while": true, "until": true,
	"export": true, "let": true, "trap": true, "alias": true,
}

func mutatesVariables(tokens []string) bool {
	printf := false
	dashV := false
	for index, token := range tokens {
		if variableMutators[token] && (token != "." || dotIsSource(tokens[:index])) {
			return true
		}
		printf = printf || token == "printf"
		dashV = dashV || token == "-v"
	}
	return printf && dashV
}

// dotIsSource reports whether a "." after the given words is the source
// builtin: it is in command position, behind only assignments, command,
// builtin or a pipe or background mark. Elsewhere it is a path argument, as in
// git add .
func dotIsSource(before []string) bool {
	for _, word := range before {
		if !isAssignment(word) && word != "command" && word != "builtin" && word != "|" && word != "&" {
			return false
		}
	}
	return true
}

// compoundTarget names the variable a NAME=x, NAME+=x or NAME[i]=x token sets.
func compoundTarget(token string) string {
	left, _, found := strings.Cut(token, "=")
	if !found {
		return ""
	}
	left = strings.TrimSuffix(left, "+")
	if open := strings.Index(left, "["); open > 0 {
		left = left[:open]
	}
	if left == "" || !isAssignment(left+"=") {
		return ""
	}
	return left
}

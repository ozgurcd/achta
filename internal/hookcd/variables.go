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
// a statement of their own, to a literal absolute path earlier in the same
// command. It is lexical and never reads the environment. Any statement that
// could change a variable in a way the hook cannot follow clears the table, so
// an unknown value is never guessed (HOOK-CD-VARIABLE-1).
type variables struct {
	limits   Limits
	values   map[string]string
	overflow bool
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

// observe updates the table from one statement, after the statement itself
// was checked against the table as it stood before.
func (v *variables) observe(tokens []string) {
	if len(tokens) == 0 {
		return
	}
	if mutatesVariables(tokens) {
		clear(v.values)
		return
	}
	assignments := tokens
	if assignments[0] == "export" {
		assignments = assignments[1:]
	}
	allAssignments := len(assignments) > 0
	for _, token := range assignments {
		if !isAssignment(token) {
			allAssignments = false
			break
		}
	}
	if allAssignments {
		for _, token := range assignments {
			name, value, _ := strings.Cut(token, "=")
			v.set(name, value)
		}
		return
	}
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
// word: no whitespace (word splitting), no $ or leading ~ (a quoted one is
// literal text, an unquoted one expands, and the hook cannot tell which), no
// glob.
func (v *variables) storable(value string) bool {
	if value == "" || len(value) > v.limits.MaxValueBytes || !strings.HasPrefix(value, "/") {
		return false
	}
	for _, char := range value {
		if unicode.IsSpace(char) {
			return false
		}
	}
	literal := literalPath(value)
	return literal != "" && filepath.IsAbs(literal) && !strings.Contains(value, "$")
}

// variableMutators are words that can change a variable outside a plain
// assignment. Any token equal to one clears the table, even as data, so the
// hook fails closed.
var variableMutators = map[string]bool{
	"eval": true, "source": true, ".": true, "read": true, "unset": true,
	"declare": true, "typeset": true, "local": true, "readonly": true,
	"mapfile": true, "readarray": true, "getopts": true,
	"for": true, "select": true, "while": true, "until": true,
}

func mutatesVariables(tokens []string) bool {
	printf := false
	dashV := false
	for _, token := range tokens {
		if variableMutators[token] {
			return true
		}
		printf = printf || token == "printf"
		dashV = dashV || token == "-v"
	}
	return printf && dashV
}

// compoundTarget names the variable a NAME+=x or NAME[i]=x token changes.
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

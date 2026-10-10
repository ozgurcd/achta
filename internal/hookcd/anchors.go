package hookcd

import (
	"path/filepath"
	"strings"
)

// literalPath normalizes only the documented home forms, without reading the
// environment or expanding arbitrary shell expressions. This is a lexical
// location guard, not a filesystem or symlink authorization boundary.
func literalPath(path string) string {
	for _, home := range []string{"${HOME}", "$HOME", "~"} {
		if path == home || strings.HasPrefix(path, home+"/") {
			path = "/~home" + strings.TrimPrefix(path, home)
			break
		}
	}
	if strings.ContainsAny(path, "$`*?[") {
		return ""
	}
	return filepath.Clean(path)
}

func anchoredPath(path, directory, anchor string) bool {
	path = literalPath(path)
	if path == "" {
		return false
	}
	if !filepath.IsAbs(path) {
		if directory == "" {
			return false
		}
		path = filepath.Join(directory, path)
	}
	return anchor == "" || anchor == "/" || path == anchor || strings.HasPrefix(path, anchor+"/")
}

// Check redirects and pipeline invocations independently: -C changes the
// program's directory, not the shell's redirect or another pipeline writer.
func unanchoredWriter(tokens []string, directory, anchor string) string {
	for index, token := range tokens {
		if (token == redirectOutputToken || token == redirectAppendToken) && index+1 < len(tokens) {
			target := tokens[index+1]
			if target != "/dev/null" && !strings.HasPrefix(target, "&") && !anchoredPath(target, directory, anchor) {
				return strings.TrimPrefix(token, "\x00")
			}
		}
	}
	for _, segment := range pipelineSegments(tokens) {
		if verb := segmentWritingVerb(segment); verb != "" && !anchoredInvocation(segment, directory, anchor) {
			return verb
		}
	}
	return ""
}

func anchoredInvocation(tokens []string, directory, anchor string) bool {
	command, args := invocation(tokens)
	selector, selected, valid := invocationSelector(command, args, directory, anchor)
	if selected {
		if !valid {
			return false
		}
		directory = selector
	}
	// Direct file writers must anchor every target. A copy's source is a read;
	// both sides of a move are writes. Redirect operands belong to the shell.
	if command == "cp" || command == "mv" || command == "rm" || command == "mkdir" || command == "touch" || command == "tee" || command == "gofmt" || command == "sed" {
		var operands []string
		target, program := "", command != "sed"
		for index := 0; index < len(args); index++ {
			arg := args[index]
			if arg == redirectOutputToken || arg == redirectAppendToken || arg == "<" || arg == "<<" {
				index++
				continue
			}
			if (command == "cp" || command == "mv") && (arg == "-t" || arg == "--target-directory" || strings.HasPrefix(arg, "--target-directory=")) {
				if strings.Contains(arg, "=") {
					target = strings.TrimPrefix(arg, "--target-directory=")
				} else {
					index++
					if index >= len(args) {
						return false
					}
					target = args[index]
				}
				if !anchoredPath(target, directory, anchor) {
					return false
				}
				continue
			}
			if (command == "sed" && (arg == "-e" || arg == "-f" || arg == "--expression" || arg == "--file")) || (command == "gofmt" && arg == "-r") {
				index++
				program = true
				continue
			}
			if strings.HasPrefix(arg, "-") || arg == "&" {
				if command == "sed" && (strings.HasPrefix(arg, "-e") || strings.HasPrefix(arg, "-f") || strings.HasPrefix(arg, "--expression=") || strings.HasPrefix(arg, "--file=")) {
					program = true
				}
				continue
			}
			if !program {
				program = true
				continue
			}
			operands = append(operands, arg)
		}
		if command == "cp" && len(operands) > 0 {
			if target != "" {
				operands = nil
			} else {
				operands = operands[len(operands)-1:]
			}
		}
		for _, path := range operands {
			if !anchoredPath(path, directory, anchor) {
				return false
			}
		}
		return len(operands) > 0 || target != "" || directory != ""
	}
	return directory != ""
}

func invocationSelector(command string, args []string, directory, anchor string) (string, bool, bool) {
	flag := ""
	switch command {
	case "git", "go", "make":
		flag = "-C"
	case "achta", "rulefloor":
		flag = "--repo"
	default:
		return "", false, true
	}
	selector, selected := "", false
	for index := 0; index < len(args); index++ {
		arg := args[index]
		if arg == "--" {
			break
		}
		path := ""
		if arg == flag {
			if index+1 >= len(args) {
				return "", true, false
			}
			index++
			path = args[index]
		} else if strings.HasPrefix(arg, flag+"=") {
			path = strings.TrimPrefix(arg, flag+"=")
		} else {
			if flag == "-C" && command != "make" && !strings.HasPrefix(arg, "-") {
				break // Git/Go selectors precede the verb; later values are data.
			}
			if arg == "-c" || arg == "--git-dir" || arg == "--work-tree" || (flag == "--repo" && strings.HasPrefix(arg, "-") && !strings.Contains(arg, "=") && arg != "--json" && arg != "--quiet" && arg != "--timing" && arg != "--write" && arg != "--force" && arg != "--check") {
				index++
			}
			continue
		}
		selected = true
		if !isAbsoluteSelector(path) {
			if directory == "" || literalPath(path) == "" {
				return "", true, false
			}
			path = filepath.Join(directory, literalPath(path))
		}
		if (flag == "-C" && !hasAbsoluteC([]string{flag, path})) || !anchoredPath(path, "", anchor) {
			return "", true, false
		}
		selector = literalPath(path)
	}
	return selector, selected, true
}

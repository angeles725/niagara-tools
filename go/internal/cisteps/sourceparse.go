package cisteps

import (
	"path"
	"regexp"
	"strings"
)

// The source-line parser of the shellcheck closure (sourced.go): which `.`/`source` commands a script runs
// and the literal path tail each one names. (retro 2026-11-15 coordination-stretch T14, T22)

var (
	sourceDirective = regexp.MustCompile(`^[ \t]*#[ \t]*shellcheck[ \t].*\bsource=(\S+)`)
	// expansion is one parameter or command expansion: $NAME, $1, ${...}, $(...).
	expansion = regexp.MustCompile(`\$(?:[A-Za-z_][A-Za-z0-9_]*|[0-9@*#?$!-]|\{[^}]*\}|\([^)]*\))`)
	// sourceWord is a `.` or `source` word followed by its argument; sourceCommands keeps the ones in command
	// position (commandPrefix), outside quotes and comments.
	sourceWord = regexp.MustCompile(`(?:^|[^A-Za-z0-9_./-])(\.|source)[ \t]+\S`)
	// commandPrefix is text a command can follow on the same line: nothing, a separator or a keyword.
	commandPrefix = regexp.MustCompile(`(?:^|;|&&|\|\||\||\{|\(|!|(?:^|[\s;&|({])(?:then|do|else|if|while|until|time))[ \t]*$`)
)

// sourceRef is one `.`/`source` line: the literal path tail it names ("" when the path cannot be read), or
// with exact, a repository path from a `# shellcheck source=SCRIPTDIR/...` directive.
type sourceRef struct {
	line  int
	tail  string
	exact bool
}

// sourceRefs lists the source commands of a script: at the start of a line or after a separator or keyword
// (`[ -f x ] && . x`, `then . x`, `{ . x; }`), here-document text, quotes and comments skipped. A
// `# shellcheck source=` line names the path of the next command line's first source (/dev/null: none to
// follow); comment and blank lines in between keep it.
func sourceRefs(script string, data []byte) []sourceRef {
	lines := strings.Split(string(data), "\n")
	body := heredocBody(lines)
	var refs []sourceRef
	directive := ""
	for i, line := range lines {
		if body[i] {
			continue
		}
		if m := sourceDirective.FindStringSubmatch(line); m != nil {
			directive = m[1]
			continue
		}
		if t := strings.TrimSpace(line); t == "" || strings.HasPrefix(t, "#") {
			continue
		}
		for _, arg := range sourceCommands(line[:commentAt(line)]) {
			switch {
			case directive == "/dev/null":
			case strings.HasPrefix(directive, "SCRIPTDIR/"):
				refs = append(refs, sourceRef{i + 1, path.Join(path.Dir(script), strings.TrimPrefix(directive, "SCRIPTDIR/")), true})
			case directive != "":
				refs = append(refs, sourceRef{i + 1, literalTail(directive), false})
			default:
				refs = append(refs, sourceRef{i + 1, literalTail(firstCommand(arg)), false})
			}
			directive = ""
		}
		directive = ""
	}
	return refs
}

// sourceCommands returns the argument text of each source command on a line, in order.
func sourceCommands(code string) []string {
	var args []string
	for _, m := range sourceWord.FindAllStringSubmatchIndex(code, -1) {
		at := m[2]
		if quotedAt(code, at) || !commandPrefix.MatchString(code[:at]) {
			continue
		}
		args = append(args, strings.TrimLeft(code[m[3]:], " \t"))
	}
	return args
}

// firstCommand cuts the argument text at the first unquoted `;`, `&&` or `||`.
func firstCommand(arg string) string {
	for i := 0; i < len(arg); i++ {
		if (arg[i] == ';' || strings.HasPrefix(arg[i:], "&&") || strings.HasPrefix(arg[i:], "||")) && !quotedAt(arg, i) {
			return strings.TrimSpace(arg[:i])
		}
	}
	return strings.TrimSpace(arg)
}

// literalTail is the path after the argument's last expansion, quotes and leading ./ and ../ dropped; ""
// when no literal file name follows the expansions or one could not be parsed.
func literalTail(arg string) string {
	if locs := expansion.FindAllStringIndex(arg, -1); len(locs) > 0 {
		arg = arg[locs[len(locs)-1][1]:]
	}
	arg = strings.NewReplacer(`"`, "", `'`, "").Replace(arg)
	if strings.ContainsAny(arg, "$`(){} \t") {
		return ""
	}
	parts := strings.Split(arg, "/")
	for i := len(parts) - 1; i >= 0; i-- {
		if parts[i] == ".." || parts[i] == "." || parts[i] == "" {
			parts = parts[i+1:]
			break
		}
	}
	if len(parts) == 0 || parts[len(parts)-1] == "" {
		return ""
	}
	return strings.Join(parts, "/")
}

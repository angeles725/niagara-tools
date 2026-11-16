package checkbatsskips

import (
	"strings"
	"unicode/utf8"
)

// The allowlist dialect, as the header of tests/bats-skip-allowlist.txt states it: POSIX ERE as Go
// regexp.CompilePOSIX reads it, less every construct Go compiles with a meaning POSIX ERE or GNU grep -E
// does not give it.
const (
	// escapable are the characters a backslash may precede: the ERE operators, made literal.
	escapable = `.[](){}*+?|^$\`
	// posixClasses are the class names a bracket expression may hold as [:name:].
	posixClasses = " alnum alpha blank cntrl digit graph lower print punct space upper xdigit "
)

// foreign is the first construct of a compiled pattern line that is outside the dialect, "" when the
// whole line is inside it:
//   - a backslash before anything but an ERE operator (Go reads \n \t \x41 \123 as control, hex and octal
//     escapes and \< \> as literal characters; GNU grep reads them as letters and word boundaries);
//   - a backslash inside a bracket expression (an escape for Go, a literal backslash for POSIX);
//   - [:name:] with a name that is not a POSIX class ([:word:] is Go's own);
//   - an equivalence class [=a=] or a collating symbol [.a.] (Go reads the characters one by one).
func foreign(line string) string {
	for i := 0; i < len(line); i++ {
		switch line[i] {
		case '\\':
			if !escapes(line, i) {
				return escape(line, i)
			}
			i++
		case '[':
			end, found := bracket(line, i)
			if found != "" {
				return found
			}
			i = end
		}
	}
	return ""
}

// bracket scans the bracket expression that opens at line[open]; end is the index of its closing `]`.
// A `]` right after the opening `[` or `[^` is a member, not the end.
func bracket(line string, open int) (end int, found string) {
	i := open + 1
	if i < len(line) && line[i] == '^' {
		i++
	}
	if i < len(line) && line[i] == ']' {
		i++
	}
	for ; i < len(line) && line[i] != ']'; i++ {
		switch {
		case line[i] == '\\':
			return i, escape(line, i)
		case line[i] == '[' && i+1 < len(line) && strings.IndexByte(":=.", line[i+1]) >= 0:
			kind := line[i+1 : i+2]
			n := strings.Index(line[i+2:], kind+"]")
			if n < 0 {
				return i, line[i : i+2]
			}
			unit := line[i : i+2+n+2]
			if kind != ":" || !strings.Contains(posixClasses, " "+line[i+2:i+2+n]+" ") {
				return i, unit
			}
			i += len(unit) - 1
		}
	}
	return i, ""
}

// escapes reports whether the backslash at line[i] precedes an ERE operator.
func escapes(line string, i int) bool {
	return i+1 < len(line) && strings.IndexByte(escapable, line[i+1]) >= 0
}

// escape is the backslash at line[i] with the character after it, as written.
func escape(line string, i int) string {
	_, size := utf8.DecodeRuneInString(line[i+1:])
	return line[i : i+1+size]
}

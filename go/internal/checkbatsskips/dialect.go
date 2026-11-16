package checkbatsskips

import (
	"regexp"
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

// interval is a brace expression Go and GNU grep -E both read as a repetition count: {n}, {n,} or {n,m},
// each bound written without a leading zero (Go reads {02} as a literal, GNU grep as the count 2).
var interval = regexp.MustCompile(`^\{(0|[1-9][0-9]*)(,(0|[1-9][0-9]*)?)?\}`)

// repeatedAnchor is a star right after the start anchor: Go repeats the anchor, GNU grep -E warns
// "* at start of expression" and reads a literal star.
const repeatedAnchor = "^*"

// foreign is the first construct of a compiled pattern line that is outside the dialect, "" when the
// whole line is inside it:
//   - a backslash before anything but an ERE operator (Go reads \n \t \x41 \123 as control, hex and octal
//     escapes and \< \> as literal characters; GNU grep reads them as letters and word boundaries);
//   - a backslash inside a bracket expression (an escape for Go, a literal backslash for POSIX);
//   - [:name:] with a name that is not a POSIX class ([:word:] is Go's own);
//   - an equivalence class [=a=] or a collating symbol [.a.] (Go reads the characters one by one);
//   - a class name in single brackets, [:alpha:] (Go reads a set of those characters; GNU grep refuses it);
//   - an unescaped brace that opens no {n}, {n,} or {n,m} interval (Go reads {,2} and {} as literals; GNU
//     grep reads {,2} as the interval 0 to 2 and refuses {}), a bound with a leading zero included ({02});
//   - a star right after the start anchor, ^* (see repeatedAnchor).
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
		case '{':
			unit := interval.FindString(line[i:])
			if unit == "" {
				return brace(line, i)
			}
			i += len(unit) - 1
		case '^':
			if strings.HasPrefix(line[i:], repeatedAnchor) {
				return repeatedAnchor
			}
		}
	}
	return ""
}

// brace is the brace expression that opens at line[i] as written, up to its closing brace; the opening
// brace alone when none closes it.
func brace(line string, i int) string {
	if n := strings.IndexByte(line[i:], '}'); n >= 0 {
		return line[i : i+n+1]
	}
	return line[i : i+1]
}

// bracket scans the bracket expression that opens at line[open]; end is the index of its closing `]`.
// A `]` right after the opening `[` or `[^` is a member, not the end.
func bracket(line string, open int) (end int, found string) {
	i := open + 1
	if i < len(line) && line[i] == '^' {
		i++
	}
	first, nested := i, false
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
			i, nested = i+len(unit)-1, true
		}
	}
	if i < len(line) && !nested && bareClass(line[first:i]) {
		return i, line[open : i+1]
	}
	return i, ""
}

// bareClass reports the members of a bracket expression that GNU grep takes for a class name written
// without its outer brackets: they start and end with a colon and hold something else in between.
func bareClass(members string) bool {
	return len(members) > 2 && members[0] == ':' && members[len(members)-1] == ':' && strings.Trim(members, ":") != ""
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

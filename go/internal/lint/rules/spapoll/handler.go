package spapoll

import "strings"

// handler.go reads a catch handler at its own level (issue #966 item 3): a throw or a return inside a function
// the handler passes to another call (`list.forEach(i => { return; })`) leaves the catch, not the callback.

// exitWord is a throw or a return statement; rejectCall is the Promise.reject( a concise handler hands back.
var (
	exitWord   = rx(`(^|[^A-Za-z0-9_$])(throw|return)([^A-Za-z0-9_$]|$)`)
	rejectCall = rx(`Promise[[:space:]]*\.[[:space:]]*reject[[:space:]]*\(`)
	fnHead     = rx(`(^|[^A-Za-z0-9_$])function([[:space:]]*\*)?([[:space:]]+[A-Za-z0-9_$]+)?[[:space:]]*\(`)
	fnAt       = rx(`^function([[:space:]]*\*)?([[:space:]]+[A-Za-z0-9_$]+)?[[:space:]]*\(`)
)

// exitsCatch: the handler leaves the catch with a throw, a return or (a concise arrow) Promise.reject(..) at its
// own level. handler is a block catch's "{ ..." body or a promise catch's "(handler" argument, each without its
// closing byte. A conditional exit (`if (fatal) { return; }`) counts, as it did; a named handler (`.catch(onError)`)
// shows no exit; an exit in a nested function, a comment or a string does not count (code arrives blanked).
func exitsCatch(handler string) bool {
	if !strings.HasPrefix(handler, "(") {
		return exitWord.MatchString(ownLevel(handler[min(1, len(handler)):]))
	}
	body, concise := handlerBody(handler[1:])
	own := ownLevel(body)
	return exitWord.MatchString(own) || concise && rejectCall.MatchString(own)
}

// handlerBody is the text after the opening of the first function written in arg (an arrow or a `function`),
// and whether it is a concise arrow body (an expression): "" when arg writes no function.
func handlerBody(arg string) (body string, concise bool) {
	arrow := strings.Index(arg, "=>")
	if head := fnHead.FindStringIndex(arg); head != nil && (arrow < 0 || head[0] < arrow) {
		shut := matching(arg, head[1]-1, '(', ')')
		if shut < 0 {
			return "", false
		}
		open := strings.IndexByte(arg[shut:], '{')
		if open < 0 {
			return "", false
		}
		return arg[shut+open+1:], false
	}
	if arrow < 0 {
		return "", false
	}
	rest := strings.TrimLeft(arg[arrow+2:], " \t\r\n")
	if strings.HasPrefix(rest, "{") {
		return rest[1:], false
	}
	return rest, true
}

// ownLevel is s with the body of every function it writes (an arrow's or a `function`'s) replaced by blanks.
func ownLevel(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); {
		from, to := nested(s, i)
		if to <= from {
			b.WriteByte(s[i])
			i++
			continue
		}
		b.WriteString(strings.Repeat(" ", to-from))
		i = to
	}
	return b.String()
}

// nested is the byte range [from, to) of the function body that starts at s[at] (an arrow's body after "=>", a
// `function`'s braces), or an empty range.
func nested(s string, at int) (from, to int) {
	if strings.HasPrefix(s[at:], "=>") {
		from = at + 2
		for from < len(s) && strings.IndexByte(" \t\r\n", s[from]) >= 0 {
			from++
		}
		if from < len(s) && s[from] == '{' {
			return from, blockEnd(s, from)
		}
		return from, conciseEnd(s, from)
	}
	if at > 0 && isIdent(s[at-1]) {
		return 0, 0
	}
	if loc := fnAt.FindStringIndex(s[at:]); loc != nil {
		shut := matching(s, at+loc[1]-1, '(', ')')
		if shut < 0 {
			return 0, 0
		}
		open := strings.IndexByte(s[shut:], '{')
		if open < 0 {
			return 0, 0
		}
		return shut + open, blockEnd(s, shut+open)
	}
	return 0, 0
}

// blockEnd is one past the "}" closing the "{" at s[open] (len(s) when it never closes).
func blockEnd(s string, open int) int {
	if shut := matching(s, open, '{', '}'); shut >= 0 {
		return shut + 1
	}
	return len(s)
}

// conciseEnd is the byte where a concise arrow body starting at from ends: a "," or ";" outside any bracket, or
// the closer of a bracket the body did not open.
func conciseEnd(s string, from int) int {
	depth := 0
	for i := from; i < len(s); i++ {
		switch s[i] {
		case '(', '[', '{':
			depth++
		case ')', ']', '}':
			if depth == 0 {
				return i
			}
			depth--
		case ',', ';':
			if depth == 0 {
				return i
			}
		}
	}
	return len(s)
}

func isIdent(c byte) bool {
	return c == '_' || c == '$' || '0' <= c && c <= '9' || 'a' <= c|0x20 && c|0x20 <= 'z'
}

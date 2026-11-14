package envfile

import (
	"os/user"
	"strings"
)

// The reasons an expansion is refused. None carries text of the line or a value; reasonUnset names the
// variable.
const (
	reasonSubstitution = "a command substitution or arithmetic ($(...), $((...)), $[...]) is not supported: write the value out"
	reasonBraces       = "inside ${...} only a variable name is supported (no default, trim, replace, length or $1 form): write the value out"
	reasonSpecial      = "a shell parameter ($1, $@, $?, $$, $_ and the like) is not supported"
	reasonDollarQuote  = "a $'...' or $\"...\" string is not supported: use plain quotes"
	reasonTilde        = "only ~ and ~/ expand to the home directory: write the path of another user out"
	reasonNoHome       = "~ cannot be expanded: HOME is not set and the user's home directory is unknown"
	reasonUnset        = "%s is not set (the file is read as under bash's set -u: an unset variable stops the read)"
)

const (
	// specials are the bytes that, after "$", name a shell parameter other than a variable.
	specials = "0123456789@*#?-$!"
	// loginBytes are the bytes that, after a leading "~", would start another user's name (or ~+ and ~-).
	loginBytes = "+-._"
	homeName   = "HOME"
	// lastArgument is "$_", a shell parameter although it reads as a variable name.
	lastArgument = "_"
)

// scope resolves a variable as bash did while it sourced the file: an earlier assignment of the file
// first, then the reader's own variables (its environment).
type scope struct {
	file  map[string]string
	outer Lookup
}

func (s *scope) get(name string) (value string, set bool) {
	if v, ok := s.file[name]; ok {
		return v, true
	}
	return s.outer(name)
}

// expand reads what the "$" at line[i] stands for and returns it with the index after it. $NAME and
// ${NAME} are the variable's value; an unset one is refused, naming it. A "$" that starts no expansion is
// itself. Every other form is refused. quoted says the "$" stands inside double quotes.
func (s *scope) expand(line string, i int, quoted bool) (value string, next int, reason string) {
	rest := line[i+1:]
	var name string
	switch {
	case rest == "":
		return "$", i + 1, ""
	case rest[0] == '{':
		end := strings.IndexByte(rest, '}')
		if end < 0 || !isName(rest[1:end]) {
			return "", i, reasonBraces
		}
		name, next = rest[1:end], i+1+end+1
	case rest[0] == '(' || rest[0] == '[':
		return "", i, reasonSubstitution
	case strings.IndexByte(specials, rest[0]) >= 0:
		return "", i, reasonSpecial
	case (rest[0] == '\'' || rest[0] == '"') && !quoted:
		return "", i, reasonDollarQuote
	default:
		n := 0
		for n < len(rest) && isName(rest[:n+1]) {
			n++
		}
		if n == 0 {
			return "$", i + 1, ""
		}
		name, next = rest[:n], i+1+n
	}
	if name == lastArgument {
		return "", i, reasonSpecial
	}
	value, set := s.get(name)
	if !set {
		return "", i, strings.Replace(reasonUnset, "%s", name, 1)
	}
	return value, next, ""
}

// tilde reads the bare "~" at line[i], where bash looks for a tilde prefix (the start of a value, or after
// a ":" in it). Alone, or before "/" or ":", it is the home directory; before a quote, an escape or an
// expansion bash leaves it as text; before a name it would be another user's home, which is refused.
func (s *scope) tilde(line string, i int) (value, reason string) {
	if i+1 < len(line) {
		switch c := line[i+1]; {
		case c == '/' || c == ':' || c == ' ' || c == '\t':
		case strings.IndexByte(loginBytes, c) >= 0 || isName(string(c)) || c >= '0' && c <= '9':
			return "", reasonTilde
		default:
			return "~", ""
		}
	}
	if home, set := s.get(homeName); set {
		return home, ""
	}
	if u, err := user.Current(); err == nil { // bash asks the user database when HOME is unset
		return u.HomeDir, ""
	}
	return "", reasonNoHome
}

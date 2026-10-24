// Package hotwrite holds what lint-changed-hot-write and lint-persist-hot-write share: the changed()
// callback they start from, the method bodies they read, the zero-arg methods changed() calls and the
// non-transient slot-o-matic properties whose setters they look for. Both rules (rules/changedhotwrite,
// rules/persisthotwrite) read a file through View and these helpers, never a private copy.
package hotwrite

import (
	"regexp"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
)

// rx compiles a gawk regular expression: leftmost-longest.
func rx(s string) *regexp.Regexp { re := regexp.MustCompile(s); re.Longest(); return re }

var (
	changedSig = rx(`changed[[:space:]]*\([[:space:]]*Property[[:space:]]+[A-Za-z_][A-Za-z0-9_]*`)
	propHead   = rx(`^.*Property[[:space:]]+`)
	propDecl   = rx(`Property[[:space:]]+[A-Za-z_][A-Za-z0-9_]*[[:space:]]*=[[:space:]]*newProperty\(`)
	propTail   = rx(`[[:space:]]*=.*`)
	flagsHead  = rx(`^.*newProperty\(`)
	zeroCall   = rx(`[A-Za-z_][A-Za-z0-9_]*[[:space:]]*\([[:space:]]*\)[[:space:]]*;`)
	callTail   = rx(`[[:space:]]*\([[:space:]]*\)[[:space:]]*;.*`)
)

// keywords are the names a zero-arg call to which is not a callee.
var keywords = map[string]bool{"if": true, "for": true, "while": true, "switch": true, "catch": true, "try": true,
	"finally": true, "return": true, "new": true, "else": true, "do": true, "super": true, "this": true,
	"Clock": true, "Sys": true}

// View is a file's lines as both rules read them: each raw line cut at its first "//" (the awk's
// sub(/\/\/.*$/, "")).
func View(f *javasrc.File) []string {
	out := make([]string, len(f.Raw))
	for i, ln := range f.Raw {
		out[i], _, _ = strings.Cut(ln, "//")
	}
	return out
}

// Changed is the 0-based line of the first `changed(Property <name>` and that name, ok false when the file
// has none.
func Changed(lines []string) (line int, prop string, ok bool) {
	for i, ln := range lines {
		if m := changedSig.FindString(ln); m != "" {
			return i, m[len(propHead.FindString(m)):], true
		}
	}
	return 0, "", false
}

// BodyEnd is the 0-based line where the braces counted from line start (the whole line) first return to
// depth 0 after one opened, ok false when they never do.
func BodyEnd(lines []string, start int) (int, bool) {
	depth, started := 0, false
	for i := start; i < len(lines); i++ {
		for _, c := range []byte(lines[i]) {
			switch c {
			case '{':
				depth, started = depth+1, true
			case '}':
				depth--
			}
		}
		if started && depth == 0 {
			return i, true
		}
	}
	return 0, false
}

// Decl is the 0-based line of the first `name()` (a non-identifier byte before it, blanks allowed in the
// parentheses) not followed by ";", the zero-arg method's declaration; ok false when there is none.
func Decl(lines []string, name string) (int, bool) {
	re := rx(`[^A-Za-z0-9_]` + regexp.QuoteMeta(name) + `[[:space:]]*\([[:space:]]*\)`)
	for i, ln := range lines {
		probe := " " + ln
		if loc := re.FindStringIndex(probe); loc != nil && !strings.HasPrefix(strings.TrimLeft(probe[loc[1]:], " \t\n\v\f\r"), ";") {
			return i, true
		}
	}
	return 0, false
}

// Callees are the names of the zero-arg methods lines[from..to] call (`name();`), keywords out, each once
// in first-call order.
func Callees(lines []string, from, to int) []string {
	var out []string
	seen := map[string]bool{}
	for i := from; i <= to; i++ {
		rest := lines[i]
		for {
			loc := zeroCall.FindStringIndex(rest)
			if loc == nil {
				break
			}
			name := callTail.ReplaceAllString(rest[loc[0]:loc[1]], "")
			rest = rest[loc[1]:]
			if !keywords[name] && !seen[name] {
				seen[name] = true
				out = append(out, name)
			}
		}
	}
	return out
}

// NonTransient are the names of the raw slot-o-matic properties (`Property <name> = newProperty(<flags>,`)
// whose flags do not mention TRANSIENT, in declaration order: one per line, its flags read after the line's
// last "newProperty(" up to the first ",".
func NonTransient(lines []string) []string {
	var out []string
	for _, ln := range lines {
		m := propDecl.FindString(ln)
		if m == "" {
			continue
		}
		nm := propTail.ReplaceAllString(m, "")
		nm = nm[len(propHead.FindString(nm)):]
		flags, _, _ := strings.Cut(ln[len(flagsHead.FindString(ln)):], ",")
		if !strings.Contains(flags, "TRANSIENT") && nm != "" {
			out = append(out, nm)
		}
	}
	return out
}

// Setter is the setter name of property name: "set" and the name with its first byte upper-cased (C locale).
func Setter(name string) string {
	if c := name[0]; c >= 'a' && c <= 'z' {
		return "set" + string(c-'a'+'A') + name[1:]
	}
	return "set" + name
}

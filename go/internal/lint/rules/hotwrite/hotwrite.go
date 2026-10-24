// Package hotwrite holds what lint-changed-hot-write and lint-persist-hot-write share: the changed()
// callback they start from, the method bodies they read (javasrc.Methods), the zero-arg methods changed() calls and the
// non-transient slot-o-matic properties whose setters they look for. Both rules (rules/changedhotwrite,
// rules/persisthotwrite) read a file through View and these helpers, never a private copy.
package hotwrite

import (
	"regexp"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
	"github.com/angeles725/niagara-tools/go/internal/lint/rules/linescan"
)

// rx compiles a gawk regular expression: leftmost-longest.
func rx(s string) *regexp.Regexp { re := regexp.MustCompile(s); re.Longest(); return re }

var (
	changedSig = rx(`changed[[:space:]]*\([[:space:]]*Property[[:space:]]+[A-Za-z_][A-Za-z0-9_]*`)
	propHead   = rx(`^.*Property[[:space:]]+`)
	propDecl   = rx(`Property[[:space:]]+[A-Za-z_][A-Za-z0-9_]*[[:space:]]*=[[:space:]]*newProperty\(`)
	propTail   = rx(`[[:space:]]*=.*`)
	zeroCall   = rx(`[A-Za-z_][A-Za-z0-9_]*[[:space:]]*\([[:space:]]*\)[[:space:]]*;`)
	callTail   = rx(`[[:space:]]*\([[:space:]]*\)[[:space:]]*;.*`)
)

// keywords are the names a zero-arg call to which is not a callee.
var keywords = map[string]bool{"if": true, "for": true, "while": true, "switch": true, "catch": true, "try": true,
	"finally": true, "return": true, "new": true, "else": true, "do": true, "super": true, "this": true,
	"Clock": true, "Sys": true}

// View is a file's lines as both rules read them: code only, comments blanked and literal contents removed
// (linescan.Code), so a brace, a call or a setter inside a comment or a string is never code.
func View(f *javasrc.File) []string { return linescan.Code(f) }

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

// Body is a method body as both rules read it: its first and last 0-based lines.
type Body struct{ Start, End int }

// ChangedBody is the body of the first `changed(Property <name>` method and that name: from its signature
// line to the end of the javasrc.Methods body that opens there or after it; ok false when there is none.
func ChangedBody(f *javasrc.File, lines []string) (b Body, prop string, ok bool) {
	sig, prop, ok := Changed(lines)
	if !ok {
		return Body{}, "", false
	}
	for _, m := range f.Methods() {
		if m.Name == "changed" && m.Start-1 >= sig {
			return Body{sig, m.End - 1}, prop, true
		}
	}
	return Body{}, "", false
}

// sigLines is how many lines above a body's opening line its signature may start (javasrc's back-scan).
const sigLines = 20

// ZeroArg is the body of the first zero-arg method name (javasrc.Methods): a body named name whose signature,
// the code from the previous body's end to its opening line, holds `name()` not followed by ";".
func ZeroArg(f *javasrc.File, lines []string, name string) (Body, bool) {
	decl := rx(`(^|[^A-Za-z0-9_])` + regexp.QuoteMeta(name) + `[[:space:]]*\([[:space:]]*\)`)
	prev := 0
	for _, m := range f.Methods() {
		sig := strings.Join(lines[max(prev, m.Start-1-sigLines):m.Start], "\n")
		prev = m.End
		if m.Name != name {
			continue
		}
		for _, loc := range decl.FindAllStringIndex(sig, -1) {
			if !strings.HasPrefix(strings.TrimLeft(sig[loc[1]:], " \t\n\v\f\r"), ";") {
				return Body{m.Start - 1, m.End - 1}, true
			}
		}
	}
	return Body{}, false
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

// Prop is one raw slot-o-matic property declaration (`Property <name> = newProperty(<flags>, <default>, ...`).
type Prop struct {
	Name      string
	Transient bool // its flags mention TRANSIENT
	Simple    bool // its default is a simple value (see simpleValue): Baja skips a write of an equal value
}

// simpleValue is a default whose slot holds a simple value: a number, boolean or string literal (contents
// blanked in the code view) or a BDouble/BFloat/BInteger/BLong/BBoolean/BString constant or factory call.
var simpleValue = rx(`^[[:space:]]*(-?[0-9][0-9.]*[dDfFlL]?|true|false|"[^"]*"|B(Double|Float|Integer|Long|Boolean|String)\.[A-Za-z_][A-Za-z0-9_]*(\([^()]*\))?)[[:space:]]*$`)

// Properties are the file's slot-o-matic property declarations in order: every declaration of a line, its
// flags and default read as the first two top-level arguments of its own newProperty( call (over up to 5
// more lines when the call wraps).
func Properties(lines []string) []Prop {
	var out []Prop
	for i, ln := range lines {
		for _, loc := range propDecl.FindAllStringIndex(ln, -1) {
			nm := propTail.ReplaceAllString(ln[loc[0]:loc[1]], "")
			nm = nm[len(propHead.FindString(nm)):]
			if nm == "" {
				continue
			}
			args := Args(joined(lines, i, loc[1], 5))
			p := Prop{Name: nm}
			if len(args) > 0 {
				p.Transient = strings.Contains(args[0], "TRANSIENT")
			}
			if len(args) > 1 {
				p.Simple = simpleValue.MatchString(args[1])
			}
			out = append(out, p)
		}
	}
	return out
}

// NonTransient are the names of the properties whose flags do not mention TRANSIENT, in declaration order.
func NonTransient(lines []string) []string {
	var out []string
	for _, p := range Properties(lines) {
		if !p.Transient {
			out = append(out, p.Name)
		}
	}
	return out
}

// joined is line i from byte at, then up to n more lines, joined by "\n".
func joined(lines []string, i, at, n int) string {
	return strings.Join(append([]string{lines[i][at:]}, lines[i+1:min(i+1+n, len(lines))]...), "\n")
}

// Args are the top-level arguments of a call whose "(" is just before s, up to its ")" (or the end of s):
// split at commas outside (), [] and {}.
func Args(s string) []string {
	var out []string
	depth, from := 0, 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '(', '[', '{':
			depth++
		case ')', ']', '}':
			if depth == 0 {
				return append(out, s[from:i])
			}
			depth--
		case ',':
			if depth == 0 {
				out, from = append(out, s[from:i]), i+1
			}
		}
	}
	return append(out, s[from:])
}

// Setter is the setter name of property name: "set" and the name with its first byte upper-cased (C locale).
func Setter(name string) string {
	if c := name[0]; c >= 'a' && c <= 'z' {
		return "set" + string(c-'a'+'A') + name[1:]
	}
	return "set" + name
}

// SetterCall matches a call of property name's setter: Setter(name) at an identifier boundary, blanks allowed
// before its "(" (`resetXHours(` is not the setter `setXHours(`, #151).
func SetterCall(name string) *regexp.Regexp {
	return rx(`(^|[^A-Za-z0-9_])` + regexp.QuoteMeta(Setter(name)) + `[[:space:]]*\(`)
}

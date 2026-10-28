// Package charsetdefault is lint-charset-default: text bytes encoded or decoded with the platform default charset.
// METHODOLOGY.md § rt (components) makes text bytes name their charset (corpus B1419 §1419.4 row c): the default is
// the host's (cp1252 on a Windows station, UTF-8 on a JACE), so the same module writes different bytes per host.
// toolbelt/lint-charset-default.sh is the wrapper and its header is the user-facing contract.
//
// A per-file rule over *.java, read as javasrc tokens (comments blanked, a literal one token), so a comment or a
// string never matches. Rows: CD1 a no-argument getBytes(); CD2 a new String with one or three arguments whose first
// argument is a byte array (a name declared byte[] / byte...[] in the file, or a getBytes / toByteArray /
// readAllBytes call); CD3 a one-argument InputStreamReader / OutputStreamWriter; CD4 any FileReader / FileWriter
// (Java 8 has no charset overload for them). A new String over a char[] or a String is not judged.
package charsetdefault

import (
	"fmt"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
	"github.com/angeles725/niagara-tools/go/internal/lint"
)

// ID is the lint name printed in every row.
const ID = "lint-charset-default"

// Spec registers the rule: [--strict] [--] <src-root>, WARN rows (exit 1 under --strict).
func Spec() lint.Spec {
	return lint.Spec{
		Rule:       rule{},
		Mode:       lint.One,
		Usage:      "usage: lint-charset-default.sh [--strict] <src-root>",
		Select:     lint.Selection{Names: []string{"*.java"}},
		EmptyError: lint.EmptyRow{Msg: "no *.java to scan (K20: wrong path or empty scaffold)"},
	}
}

type rule struct{}

func (rule) ID() string              { return ID }
func (rule) Severity() lint.Severity { return lint.Warn }

const (
	where = " (METHODOLOGY.md § rt (components))"
	cd1   = "CD1 getBytes() with no charset encodes with the platform default: pass StandardCharsets.UTF_8" + where
	cd2   = "CD2 new String(byte[]) with no charset decodes with the platform default: pass StandardCharsets.UTF_8" + where
	cd3   = "CD3 new %s with no charset uses the platform default: pass StandardCharsets.UTF_8 as the second argument" + where
	cd4r  = "CD4 new FileReader has no charset parameter on Java 8: use new InputStreamReader(new FileInputStream(f), StandardCharsets.UTF_8) or Files.newBufferedReader(path, StandardCharsets.UTF_8)" + where
	cd4w  = "CD4 new FileWriter has no charset parameter on Java 8: use new OutputStreamWriter(new FileOutputStream(f), StandardCharsets.UTF_8) or Files.newBufferedWriter(path, StandardCharsets.UTF_8)" + where
)

// byteCalls are the calls whose result is a byte[] the rule recognizes as a new String argument.
var byteCalls = map[string]bool{"getBytes": true, "toByteArray": true, "readAllBytes": true}

// Check reports each pattern once per occurrence, at the line of its getBytes or new.
func (rule) Check(f *javasrc.File) ([]lint.Finding, error) {
	toks := javasrc.Tokens(f.Stripped())
	bytes := byteNames(toks)
	var out []lint.Finding
	add := func(t javasrc.Token, msg string) { out = append(out, lint.Finding{Line: t.Line + 1, Msg: msg}) }
	for k, t := range toks {
		switch {
		case t.Text == "getBytes" && k > 0 && toks[k-1].Text == "." && at(toks, k+1, "(") && at(toks, k+2, ")"):
			add(t, cd1)
		case t.Text == "new" && t.Kind == javasrc.TokIdent:
			name, open := typeName(toks, k+1)
			if open < 0 {
				continue
			}
			args := arguments(toks, open)
			switch name {
			case "String":
				if (len(args) == 1 || len(args) == 3) && isBytes(args[0], bytes) {
					add(t, cd2)
				}
			case "InputStreamReader":
				if len(args) == 1 {
					add(t, fmt.Sprintf(cd3, "InputStreamReader(in)"))
				}
			case "OutputStreamWriter":
				if len(args) == 1 {
					add(t, fmt.Sprintf(cd3, "OutputStreamWriter(out)"))
				}
			case "FileReader":
				add(t, cd4r)
			case "FileWriter":
				add(t, cd4w)
			}
		}
	}
	return out, nil
}

func at(toks []javasrc.Token, k int, text string) bool { return k < len(toks) && toks[k].Text == text }

// typeName reads the (possibly qualified) class name after new and returns its last segment and the index of the
// "(" that follows it; -1 when no "(" follows (an array creation, a generic or anything else).
func typeName(toks []javasrc.Token, k int) (string, int) {
	name := ""
	for k < len(toks) && toks[k].Kind == javasrc.TokIdent {
		name = toks[k].Text
		if !at(toks, k+1, ".") {
			break
		}
		k += 2
	}
	if name == "" || !at(toks, k+1, "(") {
		return "", -1
	}
	return name, k + 1
}

// arguments splits the argument list that opens at toks[open] ("(") into its top-level arguments.
func arguments(toks []javasrc.Token, open int) [][]javasrc.Token {
	var args [][]javasrc.Token
	depth, start := 0, open+1
	for j := open; j < len(toks); j++ {
		switch toks[j].Text {
		case "(", "[", "{":
			depth++
		case ")", "]", "}":
			depth--
			if depth == 0 {
				if j > start {
					args = append(args, toks[start:j])
				}
				return args
			}
		case ",":
			if depth == 1 {
				args = append(args, toks[start:j])
				start = j + 1
			}
		}
	}
	return nil // unbalanced: not judged
}

// isBytes: the argument is one name declared as a byte array, or ends with a call to a byte-array method.
func isBytes(arg []javasrc.Token, names map[string]bool) bool {
	if len(arg) == 1 {
		return arg[0].Kind == javasrc.TokIdent && names[arg[0].Text]
	}
	n := len(arg)
	if n >= 3 && arg[n-1].Text == ")" {
		depth := 0
		for j := n - 1; j >= 0; j-- {
			switch arg[j].Text {
			case ")":
				depth++
			case "(":
				if depth--; depth == 0 {
					return j > 0 && byteCalls[arg[j-1].Text] && (j == 1 || arg[j-2].Text == ".")
				}
			}
		}
	}
	return false
}

// byteNames are the names the file declares as byte arrays ("byte[] x", "byte... x", "byte x[]") and with no other
// type anywhere in the file: the scan is file-wide, so a name that is a byte[] in one method and a char[] in another
// is ambiguous and never judged.
func byteNames(toks []javasrc.Token) map[string]bool {
	names := map[string]bool{}
	for k, t := range toks {
		if t.Text != "byte" || t.Kind != javasrc.TokIdent {
			continue
		}
		j, arr := k+1, false
		for {
			switch {
			case at(toks, j, "[") && at(toks, j+1, "]"):
				j, arr = j+2, true
				continue
			case at(toks, j, ".") && at(toks, j+1, ".") && at(toks, j+2, "."):
				j, arr = j+3, true
				continue
			}
			break
		}
		if j >= len(toks) || toks[j].Kind != javasrc.TokIdent {
			continue
		}
		if arr || (at(toks, j+1, "[") && at(toks, j+2, "]")) {
			names[toks[j].Text] = true
		}
	}
	for n := range otherTyped(toks) {
		delete(names, n)
	}
	return names
}

// notTypes are keywords that can stand before a name the way a type does ("return x;", "throw e;").
var notTypes = map[string]bool{"return": true, "throw": true, "new": true, "case": true, "else": true, "do": true,
	"assert": true, "yield": true, "goto": true, "package": true, "import": true, "break": true, "continue": true,
	"this": true, "super": true, "default": true, "byte": true}

// otherTyped are the names declared with a type other than byte: "<Type>[ []]* name" followed by =, ;, ,, ) or :.
func otherTyped(toks []javasrc.Token) map[string]bool {
	out := map[string]bool{}
	for k := 0; k+1 < len(toks); k++ {
		t := toks[k]
		if t.Kind != javasrc.TokIdent || notTypes[t.Text] {
			continue
		}
		j := k + 1
		for at(toks, j, "[") && at(toks, j+1, "]") {
			j += 2
		}
		if j+1 >= len(toks) || toks[j].Kind != javasrc.TokIdent {
			continue
		}
		switch toks[j+1].Text {
		case "=", ";", ",", ")", ":":
			out[toks[j].Text] = true
		}
	}
	return out
}

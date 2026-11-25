package actionhandler

import (
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
)

// method is one do<Upper>... method declared directly in a class body, with or without a body.
type method struct {
	Name           string
	Line           int      // the 0-based line of the name token
	Public, Static bool     // an interface member is public unless private
	Ret            string   // the simple return type ("void", "BOrd", "String[]")
	Params         []string // the simple parameter types
}

var modifiers = map[string]bool{"public": true, "protected": true, "private": true, "static": true, "final": true,
	"abstract": true, "synchronized": true, "native": true, "strictfp": true, "default": true}

// isDo: do followed by an upper-case letter.
func isDo(name string) bool {
	return len(name) > 2 && name[:2] == "do" && name[2] >= 'A' && name[2] <= 'Z'
}

// doMethods finds the do<Upper> method declarations directly in c's body: a nested type, a method body and an
// initializer are skipped whole, and a call, a field initializer or an annotation argument is not a declaration.
func doMethods(toks []javasrc.Token, c javasrc.Class) []method {
	var out []method
	paren, brace, start := 0, 0, c.Body[0]
	for i := c.Body[0]; i < c.Body[1]; i++ {
		switch t := toks[i].Text; {
		case brace > 0:
			if t == "{" {
				brace++
			} else if t == "}" {
				if brace--; brace == 0 {
					start = i + 1
				}
			}
		case paren > 0:
			if t == "(" {
				paren++
			} else if t == ")" {
				paren--
			}
		case t == "(":
			if m, ok := declAt(toks, start, i, c); ok {
				out = append(out, m)
			}
			paren++
		case t == "{":
			brace++
		case t == ";":
			start = i + 1
		}
	}
	return out
}

// declAt reads the member whose header starts at start and whose parameter list opens at the "(" at open.
func declAt(toks []javasrc.Token, start, open int, c javasrc.Class) (method, bool) {
	name := toks[open-1]
	if open-1 <= start || name.Kind != javasrc.TokIdent || !isDo(name.Text) {
		return method{}, false
	}
	for _, t := range toks[start : open-1] {
		switch t.Text {
		case "=", "new", "return", "throw", "else":
			return method{}, false
		}
	}
	close := open
	for depth := 0; close < c.Body[1]; close++ {
		if toks[close].Text == "(" {
			depth++
		} else if toks[close].Text == ")" {
			if depth--; depth == 0 {
				break
			}
		}
	}
	if close+1 >= c.Body[1] || (toks[close+1].Text != "{" && toks[close+1].Text != ";" && toks[close+1].Text != "throws") {
		return method{}, false
	}
	mods, ret := header(toks[start : open-1])
	iface := c.Kind == "interface"
	return method{Name: name.Text, Line: name.Line, Public: mods["public"] || (iface && !mods["private"]),
		Static: mods["static"], Ret: simple(ret), Params: paramTypes(toks[open+1 : close])}, true
}

// header splits a member header into its modifiers and the return type's tokens, skipping annotations and
// type parameters.
func header(h []javasrc.Token) (map[string]bool, []javasrc.Token) {
	mods := map[string]bool{}
	i := 0
	for i < len(h) {
		switch t := h[i]; {
		case t.Text == "@":
			i = skipAnnotation(h, i)
		case t.Kind == javasrc.TokIdent && modifiers[t.Text]:
			mods[t.Text] = true
			i++
		case t.Text == "<":
			for depth := 0; i < len(h); i++ {
				if h[i].Text == "<" {
					depth++
				} else if h[i].Text == ">" {
					if depth--; depth == 0 {
						i++
						break
					}
				}
			}
		default:
			return mods, h[i:]
		}
	}
	return mods, nil
}

// skipAnnotation is the index after the annotation at h[i]: "@" Name[.Name]... [(...)].
func skipAnnotation(h []javasrc.Token, i int) int {
	for i++; i < len(h) && (h[i].Kind == javasrc.TokIdent || h[i].Text == "."); i++ {
	}
	if i < len(h) && h[i].Text == "(" {
		for depth := 0; i < len(h); i++ {
			if h[i].Text == "(" {
				depth++
			} else if h[i].Text == ")" {
				if depth--; depth == 0 {
					return i + 1
				}
			}
		}
	}
	return i
}

// paramTypes is the simple type of each parameter of a parameter list's tokens.
func paramTypes(list []javasrc.Token) []string {
	var out []string
	depth, from := 0, 0
	for j := 0; j <= len(list); j++ {
		if j < len(list) {
			switch list[j].Text {
			case "<", "(":
				depth++
			case ">", ")":
				depth--
			}
			if list[j].Text != "," || depth != 0 {
				continue
			}
		}
		var kept []javasrc.Token
		for k := from; k < j; k++ {
			switch {
			case list[k].Text == "@":
				k = skipAnnotation(list[:j], k) - 1
			case list[k].Text != "final":
				kept = append(kept, list[k])
			}
		}
		if len(kept) > 1 {
			out = append(out, simple(kept[:len(kept)-1]))
		}
		from = j + 1
	}
	return out
}

// simple is a type's last name at generic depth 0 plus one "[]" per array dimension: java.util.List<String>[]
// is "List[]".
func simple(ts []javasrc.Token) string {
	name, dims, angle := "", "", 0
	for _, t := range ts {
		switch {
		case t.Text == "<":
			angle++
		case t.Text == ">":
			angle--
		case angle == 0 && t.Text == "[":
			dims += "[]"
		case angle == 0 && t.Kind == javasrc.TokIdent:
			name = t.Text
		}
	}
	return name + dims
}

// sig writes a method's parameters as "(String, Context)".
func sig(m method) string { return m.Name + "(" + strings.Join(m.Params, ", ") + ")" }

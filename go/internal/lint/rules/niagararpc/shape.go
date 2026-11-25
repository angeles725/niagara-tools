package niagararpc

// The shape checks of one @NiagaraRpc method: its last parameter and the transports its annotation lists.

import (
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
)

// contextTail is why a method without a trailing Context is never found: NiagaraRpcUtil.rpc appends the Context
// to the arguments (:91-122) and looks the method up by their types, a Context as Context.class (:123, :189-192).
const contextTail = "the station appends its Context as the last argument and looks the method up by those types (NiagaraRpcUtil.java:91-123; types/data-access.md §8)"

// lastParam judges the parameter list whose "(" is at open: ok when its last parameter is typed Context (or a
// qualified ....Context), else why not and the 1-based line to report. It is not judged (judged false) when
// the list never closes.
func lastParam(toks []javasrc.Token, open int) (why string, line int, ok, judged bool) {
	cl := matchTok(toks, open, "(", ")")
	if cl < 0 {
		return "", 0, true, false
	}
	seg := toks[open+1 : cl]
	if len(seg) == 0 {
		return "declares no parameter, not a trailing Context", toks[open].Line + 1, false, true
	}
	depth, from := 0, 0
	for i, t := range seg {
		switch t.Text {
		case "(", "[", "{", "<":
			depth++
		case ")", "]", "}", ">":
			depth--
		case ">>":
			depth -= 2
		case ">>>":
			depth -= 3
		case ",":
			if depth == 0 {
				from = i + 1
			}
		}
	}
	if from >= len(seg) {
		return "", 0, true, false // a trailing comma: malformed source, nothing to judge
	}
	typ := paramType(seg[from:])
	line = seg[from].Line + 1
	if typ == "Context" || strings.HasSuffix(typ, ".Context") {
		return "", line, true, true
	}
	return "last parameter is " + typ + ", not Context", line, false, true
}

// paramType is the declared type of one parameter's tokens: annotations and `final` dropped, the name cut off.
func paramType(p []javasrc.Token) string {
	var typ []string
	for i := 0; i < len(p); i++ {
		switch {
		case p[i].Text == "@":
			i++ // the annotation name is one identifier, or a dotted chain: never the type after it
			for i+2 < len(p) && p[i].Kind == javasrc.TokIdent && p[i+1].Text == "." {
				i += 2
			}
			i++
			if i < len(p) && p[i].Text == "(" {
				if c := matchTok(p, i, "(", ")"); c > 0 {
					i = c
				}
			} else {
				i--
			}
		case p[i].Text == "final":
		default:
			typ = append(typ, p[i].Text)
		}
	}
	if len(typ) > 0 {
		typ = typ[:len(typ)-1] // the parameter name
	}
	return strings.Join(typ, "")
}

// transports reads the `transports = ...` of the annotation tokens [from, to): the transport names as written, the
// 1-based line of the attribute, and judged false when it is absent (no default: NiagaraRpc.java:16) or its value is
// not made of literal @Transport(type = [TransportType.]web|box|fox) entries (a constant is not judged).
func transports(toks []javasrc.Token, from, to int) (names []string, line int, judged bool) {
	for k := from; k+2 < to; k++ {
		if toks[k].Kind != javasrc.TokIdent || toks[k].Text != "transports" || toks[k+1].Text != "=" || toks[k-1].Text == "." {
			continue
		}
		line = toks[k].Line + 1
		v, end := k+2, to
		braces := toks[v].Text == "{"
		switch {
		case braces:
			c := matchTok(toks, v, "{", "}")
			if c < 0 {
				return nil, line, false
			}
			v, end = v+1, c
		case toks[v].Text == "@" && v+2 < to && toks[v+2].Text == "(": // a single @Transport(...), no braces
			c := matchTok(toks, v+2, "(", ")")
			if c < 0 || c >= to {
				return nil, line, false
			}
			end = c + 1
		default:
			return nil, line, false
		}
		names, judged = entries(toks[v:end], braces)
		return names, line, judged
	}
	return nil, 0, false
}

// entries reads the @Transport elements of a transports value; each must name one of the three wires.
func entries(v []javasrc.Token, braces bool) (names []string, judged bool) {
	if len(v) == 0 {
		return nil, braces
	}
	elems := 1
	for i, depth := 0, 0; i < len(v); i++ {
		switch v[i].Text {
		case "(", "{":
			depth++
		case ")", "}":
			depth--
		case ",":
			if depth == 0 && i+1 < len(v) {
				elems++
			}
		}
	}
	for i := 0; i+1 < len(v); i++ {
		if v[i].Text != "@" || v[i+1].Text != "Transport" || i+2 >= len(v) || v[i+2].Text != "(" {
			continue
		}
		c := matchTok(v, i+2, "(", ")")
		if c < 0 {
			return nil, false
		}
		for k := i + 3; k+1 < c; k++ {
			if v[k].Text == "type" && v[k+1].Text == "=" && (k == i+3 || v[k-1].Text == ",") {
				e := k + 2
				for e+1 < c && v[e+1].Text == "." {
					e += 2
				}
				if n := v[e].Text; n == "web" || n == "box" || n == "fox" {
					names = append(names, n)
				}
				break
			}
		}
	}
	return names, len(names) == elems
}

func hasBox(names []string) bool {
	for _, n := range names {
		if n == "box" {
			return true
		}
	}
	return false
}

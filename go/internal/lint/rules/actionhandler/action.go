package actionhandler

import (
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
)

// Action is one @NiagaraAction annotation of a class.
type Action struct {
	Name   string   // the name argument; "" when absent or not a plain literal
	Line   int      // the 0-based line of the name argument (of the annotation when there is none)
	Param  string   // parameterType, "" when none
	Return string   // returnType, "" when none (void)
	Flags  []string // every identifier of the flags expression (Flags.HIDDEN | Flags.ASYNC: Flags, HIDDEN, Flags, ASYNC)
	Opaque bool     // a parameterType or returnType that is not a plain Java class name (baja:Type, concatenation)
}

// HasFlag reports whether the flags expression names flag (a Flags.X constant, or X by static import).
func (a Action) HasFlag(flag string) bool {
	for _, f := range a.Flags {
		if f == flag {
			return true
		}
	}
	return false
}

// Actions finds the @NiagaraAction annotations among the annotations and modifiers before c's kind keyword, in
// source order. The walk back skips each annotation's arguments whole and stops at any other token.
func Actions(toks []javasrc.Token, c javasrc.Class) []Action {
	var out []Action
walk:
	for i := c.At - 1; i >= 0; i-- {
		switch t := toks[i]; {
		case t.Text == ")":
			end := i
			for depth := 0; i >= 0; i-- {
				if toks[i].Text == ")" {
					depth++
				} else if toks[i].Text == "(" {
					if depth--; depth == 0 {
						break
					}
				}
			}
			if i >= 2 && toks[i-1].Text == "NiagaraAction" && (toks[i-2].Text == "@" || toks[i-2].Text == ".") {
				out = append(out, parseAction(toks[i+1:end], toks[i-1].Line))
			}
			if i < 0 {
				break walk
			}
		case t.Kind == javasrc.TokIdent:
		case t.Text != "@" && t.Text != ".":
			break walk
		}
	}
	for l, r := 0, len(out)-1; l < r; l, r = l+1, r-1 {
		out[l], out[r] = out[r], out[l]
	}
	return out
}

// parseAction reads the `key = value, ...` arguments of one annotation.
func parseAction(args []javasrc.Token, line int) Action {
	a := Action{Line: line}
	depth := 0
	for i := 0; i < len(args); i++ {
		switch args[i].Text {
		case "(", "{":
			depth++
		case ")", "}":
			depth--
		}
		if depth != 0 || args[i].Kind != javasrc.TokIdent || i+2 >= len(args) || args[i+1].Text != "=" {
			continue
		}
		j, d := i+2, 0
		for ; j < len(args); j++ {
			switch args[j].Text {
			case "(", "{":
				d++
			case ")", "}":
				d--
			}
			if args[j].Text == "," && d == 0 {
				break
			}
		}
		val := args[i+2 : j]
		switch args[i].Text {
		case "name":
			a.Name, a.Line = plain(val), args[i].Line
		case "parameterType":
			a.Param = a.typeArg(val)
		case "returnType":
			a.Return = a.typeArg(val)
		case "flags":
			for _, v := range val {
				if v.Kind == javasrc.TokIdent {
					a.Flags = append(a.Flags, v.Text)
				}
			}
		}
		i = j
	}
	return a
}

// typeArg is a type argument's class name; anything that is not a plain Java name marks the action opaque.
func (a *Action) typeArg(val []javasrc.Token) string {
	s := plain(val)
	if s == "" || strings.ContainsAny(s, ":|+ ") {
		a.Opaque = true
	}
	return s
}

// plain is the text of a single string literal, "" for anything else.
func plain(val []javasrc.Token) string {
	if len(val) == 1 && val[0].Kind == javasrc.TokLiteral && len(val[0].Text) >= 2 && val[0].Text[0] == '"' {
		return val[0].Text[1 : len(val[0].Text)-1]
	}
	return ""
}

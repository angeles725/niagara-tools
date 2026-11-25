package bqlunbounded

// The string expressions of a Java or JavaScript file: its literals, joined where only literals are added, and
// whether a value that is not a literal takes part.

import (
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
	"github.com/angeles725/niagara-tools/go/internal/jssrc"
)

// expr is one string expression: the text of its literals, the 1-based line of the first one, and whether a
// value the lint cannot read (a variable, a call, a ${...} substitution) is added to it.
type expr struct {
	text    string
	line    int
	dynamic bool
}

// tok is the part of a token the grouping reads, the same for both languages.
type tok struct {
	lit  bool // a string literal ("..." or text block in Java; "...", '...' or `...` in JavaScript)
	text string
	line int  // 0-based
	subs bool // a JavaScript template with a ${...} substitution
	plus bool // + or += (the left side is a value that grows)
}

// exprs reads f as the language of its extension.
func exprs(f *javasrc.File) []expr {
	if strings.HasSuffix(f.Path, ".js") {
		return group(jsToks(f.Raw))
	}
	return group(javaToks(f.Stripped()))
}

func javaToks(stripped []string) []tok {
	var out []tok
	for _, t := range javasrc.Tokens(stripped) {
		switch {
		case t.Kind == javasrc.TokLiteral && strings.HasPrefix(t.Text, `"`):
			out = append(out, tok{lit: true, text: javaText(t.Text), line: t.Line})
		case t.Kind == javasrc.TokPunct && t.Text == "+":
			out = append(out, tok{text: "+", line: t.Line, plus: true})
		case t.Kind == javasrc.TokPunct && t.Text == "=" && len(out) > 0 && out[len(out)-1].plus:
			// the second byte of +=: the + before it already stands for the operator
		default:
			out = append(out, tok{text: t.Text, line: t.Line})
		}
	}
	return out
}

// javaText is the text between the quotes of a "..." literal or the delimiters of a """ text block.
func javaText(lit string) string {
	if strings.HasPrefix(lit, `"""`) {
		inner := strings.TrimPrefix(lit, `"""`)
		return javasrc.Unescape(strings.TrimSuffix(inner, `"""`))
	}
	inner := strings.TrimPrefix(lit, `"`)
	return javasrc.Unescape(strings.TrimSuffix(inner, `"`))
}

func jsToks(raw []string) []tok {
	var out []tok
	for _, t := range jssrc.Tokens(raw) {
		switch t.Kind {
		case jssrc.String:
			out = append(out, tok{lit: true, text: javasrc.Unescape(t.Text), line: t.Line})
		case jssrc.Template:
			out = append(out, tok{lit: true, text: t.Text, line: t.Line, subs: len(t.Subs) > 0})
		default:
			out = append(out, tok{text: t.Text, line: t.Line, plus: t.Kind == jssrc.Punct && (t.Text == "+" || t.Text == "+=")})
		}
	}
	return out
}

// group joins each run of literals added to one another (a + b + c, all literals) into one expression, which is
// dynamic when a + joins it to anything else on either side or a template in it has a substitution.
func group(ts []tok) []expr {
	var out []expr
	for i := 0; i < len(ts); i++ {
		if !ts[i].lit {
			continue
		}
		e := expr{line: ts[i].line + 1, dynamic: i > 0 && ts[i-1].plus}
		var b strings.Builder
		j := i
		for {
			b.WriteString(ts[j].text)
			e.dynamic = e.dynamic || ts[j].subs
			if j+2 < len(ts) && ts[j+1].plus && ts[j+2].lit {
				j += 2
				continue
			}
			break
		}
		e.dynamic = e.dynamic || (j+1 < len(ts) && ts[j+1].plus)
		e.text = b.String()
		out = append(out, e)
		i = j
	}
	return out
}

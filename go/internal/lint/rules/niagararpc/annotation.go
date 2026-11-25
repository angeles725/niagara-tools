package niagararpc

// The reading of the annotation and the method signature after it: which attributes `permissions` holds, the
// method name and where its body opens.

import (
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
)

// perm is one `permissions = <value>` of an annotation.
type perm struct {
	line int // 1-based line of the value
	val  string
	lit  bool // a string literal that is not part of a larger expression
}

// rpcAnnotation: toks[i] is the "@" of an @NiagaraRpc or @a.b.NiagaraRpc; end is the index after its argument list.
func rpcAnnotation(toks []javasrc.Token, i int) (end int, ok bool) {
	if toks[i].Text != "@" || i+1 >= len(toks) || toks[i+1].Kind != javasrc.TokIdent {
		return 0, false
	}
	j := i + 1
	for j+2 < len(toks) && toks[j+1].Text == "." && toks[j+2].Kind == javasrc.TokIdent {
		j += 2
	}
	if toks[j].Text != "NiagaraRpc" {
		return 0, false
	}
	j++
	if j < len(toks) && toks[j].Text == "(" {
		if c := matchTok(toks, j, "(", ")"); c > 0 {
			return c + 1, true
		}
		return len(toks), true
	}
	return j, true
}

// permissions reads every `permissions = <value>` between the annotation's tokens [from, to).
func permissions(toks []javasrc.Token, from, to int) []perm {
	var out []perm
	for k := from; k+2 < to; k++ {
		if toks[k].Kind != javasrc.TokIdent || toks[k].Text != "permissions" || toks[k+1].Text != "=" || (k > from && toks[k-1].Text == ".") {
			continue
		}
		v := toks[k+2]
		next := ""
		if k+3 < len(toks) {
			next = toks[k+3].Text
		}
		if v.Kind == javasrc.TokLiteral && strings.HasPrefix(v.Text, "\"") && (next == "," || next == ")" || next == "}") {
			out = append(out, perm{line: v.Line + 1, val: unquote(v.Text), lit: true})
		} else {
			out = append(out, perm{line: v.Line + 1})
		}
	}
	return out
}

func unquote(s string) string {
	s = strings.TrimPrefix(s, "\"")
	return strings.TrimSuffix(s, "\"")
}

// signature reads what follows the annotation: skips other annotations, takes the method name (the identifier before
// the first "(" outside an annotation) and stops at the first "{" or ";". name is "?" and nameAt -1 when there is none;
// sigEnd is that "{" or ";" (-1 at end of input).
func signature(toks []javasrc.Token, k int) (name string, nameAt, sigEnd int) {
	name, nameAt = "?", -1
	for k < len(toks) {
		switch t := toks[k]; {
		case t.Text == "@":
			k++
			for k < len(toks) && (toks[k].Kind == javasrc.TokIdent || toks[k].Text == ".") {
				k++
			}
			if k < len(toks) && toks[k].Text == "(" {
				c := matchTok(toks, k, "(", ")")
				if c < 0 {
					return name, nameAt, -1
				}
				k = c + 1
			}
		case t.Text == "(":
			if nameAt < 0 && k > 0 && toks[k-1].Kind == javasrc.TokIdent {
				name, nameAt = toks[k-1].Text, k-1
			}
			c := matchTok(toks, k, "(", ")")
			if c < 0 {
				return name, nameAt, -1
			}
			k = c + 1
		case t.Text == "{" || t.Text == ";":
			return name, nameAt, k
		default:
			k++
		}
	}
	return name, nameAt, -1
}

// matchTok is the index of the close token pairing with the open token at k (-1 when unbalanced).
func matchTok(toks []javasrc.Token, k int, open, close string) int {
	depth := 0
	for j := k; j < len(toks); j++ {
		if toks[j].Kind != javasrc.TokPunct {
			continue
		}
		switch toks[j].Text {
		case open:
			depth++
		case close:
			depth--
			if depth == 0 {
				return j
			}
		}
	}
	return -1
}

// closeOf is the index of the "}" closing the "{" at k, or the last token when it never closes.
func closeOf(toks []javasrc.Token, k int) int {
	if c := matchTok(toks, k, "{", "}"); c >= 0 {
		return c
	}
	return len(toks) - 1
}

// Call-argument and waiver helpers of the lint-post-permission rule.
package postpermission

import (
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
)

// nestedHandler reports whether token k is in the body of another handler declared inside decls[di], and
// where that body ends.
func nestedHandler(decls []javasrc.Decl, isHandler []bool, di, k int) (int, bool) {
	outer := decls[di]
	for i, e := range decls {
		if i != di && isHandler[i] && e.Body[0] > outer.Body[0] && e.Body[1] <= outer.Body[1] && e.Body[0] <= k && k < e.Body[1] {
			return e.Body[1], true
		}
	}
	return 0, false
}

// arguments splits the argument list opening at toks[open] ("(") on its depth-0 commas; the list is
// empty for "()". The second result is the index of the closing ")" (len(toks) when the list never closes).
func arguments(toks []javasrc.Token, open int) ([][]javasrc.Token, int) {
	var args [][]javasrc.Token
	depth, from := 0, open+1
	for j := open; j < len(toks); j++ {
		switch toks[j].Text {
		case "(", "[", "{":
			depth++
		case ")", "]", "}":
			if depth--; depth == 0 {
				if j > from {
					args = append(args, toks[from:j])
				}
				return args, j
			}
		case ",":
			if depth == 1 {
				args = append(args, toks[from:j])
				from = j + 1
			}
		}
	}
	return args, len(toks)
}

func namesStrongPermission(args [][]javasrc.Token) bool {
	for _, a := range args {
		for _, t := range a {
			if strongPermissions[t.Text] {
				return true
			}
		}
	}
	return false
}

// isRequestPost reports whether the post( at toks[k] is BComponent.post: a bare or this. call with any
// argument count, or a call on another receiver with two or more arguments and no string literal first (an
// HTTP client's post(url) and post("http://...", body) are not it).
func isRequestPost(toks []javasrc.Token, k int, args [][]javasrc.Token) bool {
	if len(args) == 0 {
		return false
	}
	if k == 0 || toks[k-1].Text != "." {
		return true
	}
	if k >= 2 && toks[k-2].Text == "this" {
		return true
	}
	return len(args) >= 2 && args[0][0].Kind != javasrc.TokLiteral
}

// waived reports whether a post-permission waiver with a reason is in the comment on the 0-based line, or
// on the line above when that line holds only a comment (a trailing waiver there belongs to its own post).
func waived(code, comments []string, line int) bool {
	return waiver.MatchString(comments[line]) ||
		(line > 0 && strings.TrimSpace(code[line-1]) == "" && waiver.MatchString(comments[line-1]))
}

// rpc reports whether @NiagaraRpc sits in d's header: between the previous ";", "{" or "}" and its body. A
// parenthesized list (the parameters, the annotation's own arguments, with their braces) is skipped whole.
func rpc(d javasrc.Decl, toks []javasrc.Token) bool {
	depth := 0
	for j := d.Body[0] - 2; j > 0; j-- {
		switch s := toks[j].Text; {
		case s == ")":
			depth++
		case s == "(":
			if depth > 0 {
				depth--
			}
		case depth > 0:
		case s == ";" || s == "{" || s == "}":
			return false
		case s == "NiagaraRpc" && toks[j-1].Text == "@":
			return true
		}
	}
	return false
}

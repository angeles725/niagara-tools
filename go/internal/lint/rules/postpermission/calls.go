// Call-argument and waiver helpers of the lint-post-permission rule.
package postpermission

import (
	"github.com/angeles725/niagara-tools/go/internal/javasrc"
)

// nestedBody reports whether token k is in the body of another declaration inside decls[di], and where
// that body ends.
func nestedBody(decls []javasrc.Decl, di, k int) (int, bool) {
	outer := decls[di]
	for i, e := range decls {
		if i != di && e.Body[0] > outer.Body[0] && e.Body[1] <= outer.Body[1] && e.Body[0] <= k && k < e.Body[1] {
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

func namesPermissions(args [][]javasrc.Token) bool {
	for _, a := range args {
		for _, t := range a {
			if t.Text == "BPermissions" {
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

// waived reports whether a post-permission waiver with a reason is in the comment on the 0-based line or
// the line above.
func waived(comments []string, line int) bool {
	return waiver.MatchString(comments[line]) || (line > 0 && waiver.MatchString(comments[line-1]))
}

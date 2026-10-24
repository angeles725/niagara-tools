package jssrc

// Match returns, for each token index, the index of the bracket that closes an opening ( [ { at that index
// (or len(toks) when it is never closed), and -1 for every other token.
func Match(toks []Token) []int {
	m := make([]int, len(toks))
	var open []int
	for i, t := range toks {
		m[i] = -1
		if t.Kind != Punct {
			continue
		}
		switch t.Text {
		case "(", "[", "{":
			open = append(open, i)
		case ")", "]", "}":
			if n := len(open); n > 0 {
				m[open[n-1]] = i
				open = open[:n-1]
			}
		}
	}
	for _, i := range open {
		m[i] = len(toks)
	}
	return m
}

// Expr returns the end (exclusive) of the expression that starts at toks[from]: the first `;` or `,` outside
// its own brackets, a closing bracket of an enclosing one, or a token on a later line that starts a new
// statement (automatic semicolon insertion: the previous token ends an expression and this one is a name or
// a literal), whichever comes first; len(toks) at the end of input.
func Expr(toks []Token, from int) int {
	depth := 0
	for i := from; i < len(toks); i++ {
		t := toks[i]
		if depth == 0 && i > from && t.Line > toks[i-1].Line && endsExpr(toks[i-1]) && t.Kind != Punct {
			return i
		}
		if t.Kind != Punct {
			continue
		}
		switch t.Text {
		case "(", "[", "{":
			depth++
		case ")", "]", "}":
			if depth == 0 {
				return i
			}
			depth--
		case ";", ",":
			if depth == 0 {
				return i
			}
		}
	}
	return len(toks)
}

// endsExpr reports a token an expression may end with.
func endsExpr(t Token) bool {
	switch t.Kind {
	case Ident:
		return !regexAfter[t.Text]
	case Punct:
		return t.Text == ")" || t.Text == "]" || t.Text == "}" || t.Text == "++" || t.Text == "--"
	}
	return true
}

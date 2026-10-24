package jssrc

// BindKind is how a Binding introduces its name.
type BindKind int

// The binding kinds.
const (
	Decl  BindKind = iota // a const / let / var declarator (Init is its initializer, empty without one)
	Param                 // a function or arrow parameter (Fn and Index locate it)
	Of                    // the name of a for (const|let|var <name> of <expr>) loop (Init is <expr>)
)

// Binding is one name a declaration introduces, with the token range it is visible in. It is lexical scope
// read from tokens, not a parser: a declarator is visible from its name to the end of the innermost enclosing
// { } (var is treated as block scoped and nothing is hoisted), a parameter in its function's body, a for-of name
// in its loop body. Destructuring patterns introduce no binding.
type Binding struct {
	Name     string
	Kind     BindKind
	Const    bool   // declared const (Decl, Of)
	At       int    // the name's token index
	From, To int    // the tokens [From, To) the name refers to this binding in
	Init     [2]int // Decl: the initializer's token range; Of: the iterated expression's
	Fn       int    // Param: the index of the token its function starts with ("function", the "(" or the name of an arrow)
	Index    int    // Param: its position in the parameter list
}

// Bindings returns every binding of toks in token order; match is Match(toks).
func Bindings(toks []Token, match []int) []Binding {
	open := make([]int, len(toks)) // the opener of each closing bracket
	for i, m := range match {
		if m >= 0 && m < len(toks) {
			open[m] = i
		}
	}
	var out []Binding
	var braces []int
	blockEnd := func() int {
		if n := len(braces); n > 0 {
			return match[braces[n-1]]
		}
		return len(toks)
	}
	is := func(i int, text string) bool { return at(toks, i, text) }
	for i := 0; i < len(toks); i++ {
		t := toks[i]
		switch {
		case t.Kind == Punct && t.Text == "{":
			braces = append(braces, i)
		case t.Kind == Punct && t.Text == "}":
			if n := len(braces); n > 0 {
				braces = braces[:n-1]
			}
		case t.Kind == Ident && (t.Text == "const" || t.Text == "let" || t.Text == "var"):
			if is(i-2, "for") && is(i-1, "(") && i+2 < len(toks) && toks[i+1].Kind == Ident && is(i+2, "of") {
				end := match[i-1]
				body := min(end+1, len(toks))
				to := Expr(toks, body)
				if is(body, "{") {
					to = match[body]
				}
				out = append(out, Binding{Name: toks[i+1].Text, Kind: Of, Const: t.Text == "const", At: i + 1,
					From: body, To: to, Init: [2]int{i + 3, end}})
				continue
			}
			for j := i + 1; j < len(toks) && toks[j].Kind == Ident; {
				b := Binding{Name: toks[j].Text, Kind: Decl, Const: t.Text == "const", At: j, From: j, To: blockEnd()}
				next := j + 1
				if is(j+1, "=") {
					b.Init = [2]int{j + 2, Expr(toks, j+2)}
					next = b.Init[1]
				}
				out = append(out, b)
				if !is(next, ",") {
					break
				}
				j = next + 1
			}
		case t.Kind == Ident && t.Text == "function":
			p := i + 1
			if p < len(toks) && toks[p].Kind == Ident {
				p++
			}
			if is(p, "(") && match[p] < len(toks) && is(match[p]+1, "{") {
				body := match[p] + 1
				out = append(out, params(toks, match, p, i, body, match[body])...)
			}
		case t.Kind == Punct && t.Text == "=>" && i > 0:
			start, body := i-1, i+1
			to := Expr(toks, body)
			if is(body, "{") {
				to = match[body]
			}
			if toks[i-1].Kind == Ident {
				out = append(out, Binding{Name: toks[i-1].Text, Kind: Param, At: i - 1, From: body, To: to, Fn: i - 1})
			} else if is(i-1, ")") {
				start = open[i-1]
				out = append(out, params(toks, match, start, start, body, to)...)
			}
		}
	}
	return out
}

// params is the bindings of the simple names in the parameter list opened at toks[lp], visible in [from, to);
// fn is the token the function starts with. A default value is skipped; a destructured parameter binds nothing.
func params(toks []Token, match []int, lp, fn, from, to int) []Binding {
	var out []Binding
	idx := 0
	for j := lp + 1; j < match[lp] && j < len(toks); idx++ {
		if toks[j].Kind == Ident {
			out = append(out, Binding{Name: toks[j].Text, Kind: Param, At: j, From: from, To: to, Fn: fn, Index: idx})
		}
		j = Expr(toks, j)
		if j >= match[lp] || !at(toks, j, ",") {
			break
		}
		j++
	}
	return out
}

// Lookup returns the binding name refers to at token index at: the innermost (latest-starting) binding of
// that name whose range holds at, or nil (a global, or a name the scan did not see declared).
func Lookup(bs []Binding, name string, at int) *Binding {
	var best *Binding
	for k := range bs {
		b := &bs[k]
		if b.Name == name && b.From <= at && at < b.To && (best == nil || b.From >= best.From) {
			best = b
		}
	}
	return best
}

// at reports that toks[i] is the name or punctuator text (never a literal that spells it).
func at(toks []Token, i int, text string) bool {
	return i >= 0 && i < len(toks) && toks[i].Text == text && (toks[i].Kind == Ident || toks[i].Kind == Punct)
}

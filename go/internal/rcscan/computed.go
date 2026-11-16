package rcscan

import "github.com/angeles725/niagara-tools/go/internal/jssrc"

// computed reports that ts[a:b] is computed only from the page's own literals: number and string literals
// (a template with no substitution), arithmetic, comparison and logic operators, Math.* calls, a member that
// literalMember clears (`u.cap` of a literal element), a const bound to such an expression, the index parameter
// of an iterator callback, and a literal-preserving method (literalMethods) on such a value. Anything else (a
// name it cannot bind, a let, a call, an index, a template with a substitution) is not proven: false.
// seen guards the const chain against a cycle.
func (s *fileScan) computed(a, b int, seen map[int]bool) bool {
	v := s.js()
	ts := v.toks
	if a >= b {
		return false
	}
	for k := a; k < b; k++ {
		t := ts[k]
		switch {
		case t.Kind == jssrc.Number || t.Kind == jssrc.String || t.Kind == jssrc.Template && len(t.Subs) == 0:
		case t.Kind == jssrc.Punct && computedOps[t.Text]:
		case t.Kind == jssrc.Punct && t.Text == ".":
			// a method on the value before it: only one that keeps literal text literal
			if k+2 >= b || ts[k+1].Kind != jssrc.Ident || !literalMethods[ts[k+1].Text] || ts[k+2].Text != "(" {
				return false
			}
			k++
		case t.Kind == jssrc.Ident && (t.Text == "true" || t.Text == "false" || t.Text == "null" || t.Text == "undefined"):
		case t.Kind == jssrc.Ident && t.Text == "Math" && k+2 < b && ts[k+1].Text == "." && ts[k+2].Kind == jssrc.Ident:
			k += 2
		case t.Kind == jssrc.Ident:
			var path []string
			e := k + 1
			for e+1 < b && ts[e].Text == "." && ts[e].Kind == jssrc.Punct && ts[e+1].Kind == jssrc.Ident &&
				!(e+2 < b && ts[e+2].Text == "(") {
				path, e = append(path, ts[e+1].Text), e+2
			}
			switch {
			case len(path) > 0:
				// Mutation: RC-computed-member -- clearing any member lets a server field into the computed value
				if !s.literalMember(t.Text, k, path) {
					return false
				}
			case !s.literalName(t.Text, k, seen):
				return false
			}
			k = e - 1
		default:
			return false
		}
	}
	return true
}

// literalName reports a bare name at stream index pos that holds a value computed from literals: a const whose
// initializer is computed, or the index parameter (the second) of a map / forEach / filter / find / some /
// every / flatMap callback whose receiver is a literal array (a number; Map.prototype.forEach's second parameter
// is a key, and a receiver the scan cannot prove an array proves nothing).
func (s *fileScan) literalName(name string, pos int, seen map[int]bool) bool {
	v := s.js()
	b := jssrc.Lookup(v.binds[name], name, pos)
	switch {
	case b == nil || seen[b.At]:
		return false
	case b.Kind == jssrc.Decl:
		seen[b.At] = true
		// Mutation: RC-computed-const -- accepting a let lets a value the page reassigns count as literal
		return b.Const && s.computed(b.Init[0], b.Init[1], seen)
	case b.Kind == jssrc.Param:
		ts, f := v.toks, b.Fn
		if !(b.Index == 1 && f >= 3 && ts[f-1].Text == "(" && ts[f-2].Kind == jssrc.Ident && iterators[ts[f-2].Text] &&
			ts[f-2].Text != "sort" && ts[f-3].Text == ".") {
			return false
		}
		// Mutation: RC-index-receiver -- any receiver's second callback parameter clears a Map key as a literal
		c := s.eval(v.chainStart(f-3), f-3)
		if c == nil || c.kind != collShape {
			return false
		}
		for _, t := range c.trees {
			if t == nil || !t.arr {
				return false
			}
		}
		return true
	}
	return false
}

// declared is the shape a call of the page's function declaration `function name(...) { ... }` returns, read
// at stream index pos: its one return statement's value, when that is a collection (collFnShape). A name with
// two declarations, or a body with any other return (nested callbacks included), proves nothing.
func (s *fileScan) declared(name string, pos int) *shape {
	v := s.js()
	ds := v.fns[name]
	if len(ds) != 1 {
		return nil
	}
	f := ds[0]
	if sh, seen := v.fnSh[f]; seen {
		return sh
	}
	v.fnSh[f] = nil
	ts, m := v.toks, v.match
	var sh *shape
	if lp := f + 2; lp < len(ts) && ts[lp].Text == "(" && m[lp]+1 < len(ts) && ts[m[lp]+1].Text == "{" {
		open := m[lp] + 1
		ret := -1
		for k := open + 1; k < m[open]; k++ {
			if ts[k].Kind == jssrc.Ident && ts[k].Text == "return" {
				if ret >= 0 {
					// Mutation: RC-computed-return -- reading the last return clears a function whose other return is server data
					ret = -2 // a second return: the value is not one expression
					break
				}
				ret = k
			}
		}
		if ret > 0 {
			if r := s.eval(ret+1, jssrc.Expr(ts, ret+1)); r != nil && r.kind == collShape {
				sh = &shape{kind: collFnShape, trees: r.trees}
			}
		}
	}
	v.fnSh[f] = sh
	return sh
}

// ternary finds the `?` and its `:` of a conditional at the top level of ts[a:b] (0, 0 when there is none).
func ternary(ts []jssrc.Token, m []int, a, b int) (int, int) {
	q, nest := 0, 0
	for k := a; k < b; k++ {
		t := ts[k]
		if t.Kind != jssrc.Punct {
			continue
		}
		switch t.Text {
		case "(", "[", "{":
			k = m[k]
		case "?":
			if q == 0 {
				q = k
			} else {
				nest++
			}
		case ":":
			if q > 0 && nest == 0 {
				return q, k
			}
			nest--
		}
	}
	return 0, 0
}

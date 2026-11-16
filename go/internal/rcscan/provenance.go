package rcscan

import "github.com/angeles725/niagara-tools/go/internal/jssrc"

// literalMember reports that base.path, read at stream index pos, is literal text of the page's own code:
// base is bound (jssrc.Bindings, lexical scope) to a const literal collection, to an element of one (a
// callback parameter of units.map(u => ...), a for (const u of units) name, units.find(...), units[i], a
// const arrow that returns units.find(...)), and every such literal holds path (issue #459 (a): a client-D
// page renders its own `const units = [{id, name, ...}]`).
func (s *fileScan) literalMember(base string, pos int, path []string) bool {
	s.markWrites()
	sh := s.resolve(base, pos)
	if sh == nil || sh.kind == elemFnShape || sh.kind == collFnShape || len(sh.trees) == 0 {
		return false
	}
	for _, t := range sh.trees {
		if !t.holds(path) {
			return false
		}
	}
	return true
}

// resolve is the shape of name at stream index pos, or nil.
func (s *fileScan) resolve(name string, pos int) *shape {
	v := s.js()
	b := jssrc.Lookup(v.binds[name], name, pos)
	if b == nil {
		return nil
	}
	if sh, seen := v.shapes[b.At]; seen {
		return sh // nil while being resolved: a cycle proves nothing
	}
	v.shapes[b.At] = nil
	var sh *shape
	switch b.Kind {
	case jssrc.Decl:
		if b.Const && b.Init[0] < b.Init[1] {
			sh = s.eval(b.Init[0], b.Init[1])
		}
	case jssrc.Of:
		if c := s.eval(b.Init[0], b.Init[1]); c != nil && c.kind == collShape {
			sh = c.elems()
		}
	case jssrc.Param:
		ts, f := v.toks, b.Fn
		if (b.Index == 0 || b.Index == 1 && f >= 2 && ts[f-2].Text == "sort") && f >= 3 && ts[f-1].Text == "(" &&
			ts[f-2].Kind == jssrc.Ident && iterators[ts[f-2].Text] && ts[f-3].Text == "." {
			if c := s.eval(v.chainStart(f-3), f-3); c != nil && c.kind == collShape {
				sh = c.elems()
			}
		}
	}
	v.shapes[b.At] = sh
	return sh
}

// eval is the shape of the expression ts[a:b], or nil.
func (s *fileScan) eval(a, b int) *shape {
	v := s.js()
	ts, m := v.toks, v.match
	if a >= b {
		return nil
	}
	if q, c := ternary(ts, m, a, b); q > 0 {
		x, y := s.eval(q+1, c), s.eval(c+1, b)
		if x == nil || y == nil || x.kind != y.kind {
			return nil
		}
		return &shape{kind: x.kind, trees: append(append([]*lit(nil), x.trees...), y.trees...)}
	}
	for k := a; k < b; k++ { // an arrow function returning an element
		if ts[k].Text == "=>" && ts[k].Kind == jssrc.Punct {
			r := s.eval(k+1, b)
			if r == nil || !(k == a+1 || ts[a].Text == "(" && m[a] == k-1) {
				return nil
			}
			switch r.kind {
			case elemShape:
				return &shape{kind: elemFnShape, trees: r.trees}
			case collShape:
				return &shape{kind: collFnShape, trees: r.trees}
			}
			return nil
		}
		if ts[k].Kind == jssrc.Punct && (ts[k].Text == "(" || ts[k].Text == "[" || ts[k].Text == "{") {
			k = m[k]
		}
	}
	if (ts[a].Text == "[" || ts[a].Text == "{") && ts[a].Kind == jssrc.Punct && m[a] == b-1 {
		if t, _ := parseLit(ts, m, a); t != nil {
			return &shape{kind: collShape, trees: []*lit{t}}
		}
		return nil
	}
	if ts[a].Kind != jssrc.Ident {
		return nil
	}
	sh := s.resolve(ts[a].Text, a)
	// a function declaration only when no local binding shadows its name (`const series = () => data`)
	// Mutation: RC-declared-shadow -- falling back past a local binding clears the shadowing const's server data
	if sh == nil && a+1 < b && ts[a+1].Text == "(" && ts[a+1].Kind == jssrc.Punct && jssrc.Lookup(v.binds[ts[a].Text], ts[a].Text, a) == nil {
		sh = s.declared(ts[a].Text, a)
	}
	for k := a + 1; k < b && sh != nil; {
		switch {
		case ts[k].Text == "." && k+2 < b && ts[k+1].Kind == jssrc.Ident && ts[k+2].Text == "(":
			name := ts[k+1].Text
			switch {
			case sh.kind == collShape && (name == "filter" || name == "slice" || name == "sort" || name == "reverse"):
			case sh.kind == collShape && (name == "find" || name == "findLast"):
				sh = sh.elems()
			case sh.kind == collShape && name == "map":
				sh = s.mapped(k+1, k+2)
			default:
				sh = nil
			}
			k = m[k+2] + 1
		case ts[k].Text == "[" && sh.kind == collShape:
			sh, k = sh.elems(), m[k]+1
		case ts[k].Text == "(" && sh.kind == elemFnShape:
			sh, k = &shape{kind: elemShape, trees: sh.trees}, m[k]+1
		case ts[k].Text == "(" && sh.kind == collFnShape:
			sh, k = &shape{kind: collShape, trees: sh.trees}, m[k]+1
		default:
			sh = nil
		}
	}
	return sh
}

// mapped is the collection that `<literal collection>.map(<callback>)` computes, the map token at ts[mapAt] and
// its "(" at ts[lp] (issue #459 tail, retro go-rc-scan Δ3: a client-D page renders `series()`, a map over its own
// `units`). The callback is an arrow with an expression body: an object literal `u => ({ k: <expr>, ... })` gives
// one object whose field k is literal text when <expr> is computed only from the page's literals (s.computed),
// unknown otherwise; any other body gives one scalar when it is computed so. The result is cached per call site,
// so markWrites and every reader see one tree (a write `tw[0].c = srv` dirties it for all). A verdict taken while
// markWrites is still marking can be stale; markWrites re-checks every site once all writes are marked (recheck).
func (s *fileScan) mapped(mapAt, lp int) *shape {
	v := s.js()
	if sh, seen := v.mapped[mapAt]; seen {
		return sh
	}
	v.mapped[mapAt] = nil
	ts, m := v.toks, v.match
	var sh *shape
	from, to := lp+1, m[lp]
	arrow := -1
	switch {
	case from+1 < to && ts[from].Kind == jssrc.Ident && ts[from+1].Text == "=>":
		arrow = from + 1
	case from < to && ts[from].Text == "(" && ts[from].Kind == jssrc.Punct && m[from]+1 < to && ts[m[from]+1].Text == "=>":
		arrow = m[from] + 1
	}
	if arrow > 0 && jssrc.Expr(ts, arrow+1) == to {
		body := arrow + 1
		if ts[body].Text == "(" && m[body] == to-1 && ts[body+1].Text == "{" && m[body+1] == to-2 {
			sh = &shape{kind: collShape, trees: []*lit{{arr: true, elems: []*lit{s.object(body + 1)}}}}
			v.sites = append(v.sites, site{sh.trees[0], body + 1, -1})
		} else if s.computed(body, to, map[int]bool{}) {
			sh = &shape{kind: collShape, trees: []*lit{{arr: true, elems: []*lit{{}}}}}
			v.sites = append(v.sites, site{sh.trees[0], body, to})
		}
	}
	v.mapped[mapAt] = sh
	return sh
}

package rcscan

import "github.com/angeles725/niagara-tools/go/internal/jssrc"

// shape is what a name's value is known to be: a literal collection (an array or object literal, one narrowed
// by filter / slice / sort / reverse, or one computed from a literal collection by .map), an element of one, a
// function that returns an element, or a function that returns a collection.
type shape struct {
	kind  int // collShape, elemShape, elemFnShape or collFnShape
	trees []*lit
}

const (
	collShape = iota
	elemShape
	elemFnShape
	collFnShape
)

// elems is the elements of a collection: an array's items, an object's field values.
func (sh *shape) elems() *shape {
	out := &shape{kind: elemShape}
	for _, t := range sh.trees {
		if t.dirty {
			out.trees = append(out.trees, nil)
		}
		out.trees = append(out.trees, t.elems...)
		for _, f := range t.fields {
			out.trees = append(out.trees, f)
		}
	}
	return out
}

// iterators are the array methods whose callback's first parameter is an element of the receiver.
var iterators = map[string]bool{"map": true, "forEach": true, "filter": true, "find": true, "findLast": true,
	"findIndex": true, "some": true, "every": true, "flatMap": true, "sort": true}

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
	if sh == nil && a+1 < b && ts[a+1].Text == "(" && ts[a+1].Kind == jssrc.Punct {
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

// mapped is the collection that `<literal collection>.map(<callback>)` computes, the .map name at ts[name] and
// its "(" at ts[lp] (issue #459 tail, retro go-rc-scan Δ3: a client-D page renders `series()`, a map over its own
// `units`). The callback is an arrow with an expression body: an object literal `u => ({ k: <expr>, ... })` gives
// one object whose field k is literal text when <expr> is computed only from the page's literals (s.computed),
// unknown otherwise; any other body gives one scalar when it is computed so. The result is cached per call site,
// so markWrites and every reader see one tree (a write `tw[0].c = srv` dirties it for all).
func (s *fileScan) mapped(name, lp int) *shape {
	v := s.js()
	if sh, seen := v.mapped[name]; seen {
		return sh
	}
	v.mapped[name] = nil
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
		} else if s.computed(body, to, map[int]bool{}) {
			sh = &shape{kind: collShape, trees: []*lit{{arr: true, elems: []*lit{{}}}}}
		}
	}
	v.mapped[name] = sh
	return sh
}

// object is the object literal opened at ts[open] with each `key: <expr>` field literal text (a scalar) when
// <expr> is computed only from the page's literals, unknown (nil) otherwise; a shorthand, spread, method or
// computed key declares no field the scan can read.
func (s *fileScan) object(open int) *lit {
	v := s.js()
	ts, end := v.toks, v.match[open]
	o := &lit{obj: true, fields: map[string]*lit{}}
	for j := open + 1; j < end; {
		if ts[j].Text == "," {
			j++
			continue
		}
		if !(j+1 < end && ts[j+1].Text == ":" && ts[j].Kind != jssrc.Template && ts[j].Kind != jssrc.Punct) {
			j = jssrc.Expr(ts, j)
			continue
		}
		key, n := ts[j].Text, jssrc.Expr(ts, j+2)
		if s.computed(j+2, n, map[int]bool{}) {
			o.fields[key] = &lit{}
		} else {
			// Mutation: RC-computed-field -- a field computed from anything else would read as literal
			o.fields[key] = nil
		}
		j = n
	}
	return o
}

// numericMethods are the methods whose result, called on literal text, is literal text too.
var numericMethods = map[string]bool{"toFixed": true, "toPrecision": true, "toString": true, "toUpperCase": true,
	"toLowerCase": true, "trim": true, "padStart": true, "padEnd": true}

// computedOps are the operators an expression computed from literals may use.
var computedOps = map[string]bool{"+": true, "-": true, "*": true, "/": true, "%": true, "**": true, "(": true,
	")": true, ",": true, "?": true, ":": true, "<": true, ">": true, "<=": true, ">=": true, "==": true, "!=": true,
	"===": true, "!==": true, "&&": true, "||": true, "!": true}

// computed reports that ts[a:b] is computed only from the page's own literals: number and string literals
// (a template with no substitution), arithmetic, comparison and logic operators, Math.* calls, a member that
// literalMember clears (`u.cap` of a literal element), a const bound to such an expression, the index parameter
// of an iterator callback, and a literal-preserving method (numericMethods) on such a value. Anything else (a
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
			if k+2 >= b || ts[k+1].Kind != jssrc.Ident || !numericMethods[ts[k+1].Text] || ts[k+2].Text != "(" {
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
// every / flatMap callback, a number.
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
		return b.Index == 1 && f >= 3 && ts[f-1].Text == "(" && ts[f-2].Kind == jssrc.Ident && iterators[ts[f-2].Text] &&
			ts[f-2].Text != "sort" && ts[f-3].Text == "."
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

// writeOps are the operators that write the member chain before them.
var writeOps = map[string]bool{"=": true, "+=": true, "-=": true, "*=": true, "/=": true, "%=": true, "**=": true,
	"&&=": true, "||=": true, "??=": true, "<<=": true, ">>=": true, ">>>=": true, "&=": true, "|=": true, "^=": true,
	"++": true, "--": true}

// mutators are the methods that change the collection they are called on.
var mutators = map[string]bool{"push": true, "unshift": true, "splice": true, "fill": true, "copyWithin": true}

// markWrites marks dirty, once per file, what the page writes into a literal: the member chain that is
// assigned (`u.model = g` marks the model field of every literal u may denote, `units[i].name = x` every
// element's name), the receiver of a mutating method (`units.push(x)` the whole array) and an Object.assign
// target. A computed key (`o[k] = x`) marks the whole object. A write through an alias
// (`const a = units; a.push(x)`) reaches the same literal through the alias's own shape.
func (s *fileScan) markWrites() {
	v := s.js()
	if v.marked {
		return
	}
	v.marked = true
	ts := v.toks
	for i, t := range ts {
		if t.Kind != jssrc.Ident || i > 0 && ts[i-1].Text == "." && ts[i-1].Kind == jssrc.Punct {
			continue
		}
		head, path := i, []string(nil)
		if t.Text == "Object" && i+4 < len(ts) && ts[i+2].Text == "assign" && ts[i+3].Text == "(" && ts[i+4].Kind == jssrc.Ident {
			head = i + 4
		} else {
			e := i + 1
			for e < len(ts) {
				if ts[e].Text == "." && ts[e].Kind == jssrc.Punct && e+1 < len(ts) && ts[e+1].Kind == jssrc.Ident {
					path, e = append(path, ts[e+1].Text), e+2
				} else if ts[e].Text == "[" && ts[e].Kind == jssrc.Punct {
					path, e = append(path, "*"), v.match[e]+1
				} else {
					break
				}
			}
			switch {
			case len(path) > 0 && e < len(ts) && ts[e].Kind == jssrc.Punct && writeOps[ts[e].Text]:
			case len(path) > 0 && mutators[path[len(path)-1]] && e < len(ts) && ts[e].Text == "(":
				path = path[:len(path)-1]
			default:
				continue
			}
		}
		if sh := s.resolve(ts[head].Text, head); sh != nil {
			for _, tr := range sh.trees {
				tr.mark(path)
			}
		}
	}
}

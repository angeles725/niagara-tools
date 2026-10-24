package rcscan

import "github.com/angeles725/niagara-tools/go/internal/jssrc"

// shape is what a name's value is known to be: a literal collection (an array or object literal, or one
// narrowed by filter / slice / sort / reverse), an element of one, or a function that returns an element.
type shape struct {
	kind  int // collShape, elemShape or elemFnShape
	trees []*lit
}

const (
	collShape = iota
	elemShape
	elemFnShape
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
// const arrow that returns units.find(...)), and every such literal holds path (issue #459 (a): Umbrella
// index.html renders its own `const units = [{id, name, ...}]`).
func (s *fileScan) literalMember(base string, pos int, path []string) bool {
	s.markWrites()
	sh := s.resolve(base, pos)
	if sh == nil || sh.kind == elemFnShape || len(sh.trees) == 0 {
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
	b := jssrc.Lookup(v.binds, name, pos)
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
			if r := s.eval(k+1, b); r != nil && r.kind == elemShape && (k == a+1 || ts[a].Text == "(" && m[a] == k-1) {
				return &shape{kind: elemFnShape, trees: r.trees}
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
	for k := a + 1; k < b && sh != nil; {
		switch {
		case ts[k].Text == "." && k+2 < b && ts[k+1].Kind == jssrc.Ident && ts[k+2].Text == "(":
			name := ts[k+1].Text
			switch {
			case sh.kind == collShape && (name == "filter" || name == "slice" || name == "sort" || name == "reverse"):
			case sh.kind == collShape && (name == "find" || name == "findLast"):
				sh = sh.elems()
			default:
				sh = nil
			}
			k = m[k+2] + 1
		case ts[k].Text == "[" && sh.kind == collShape:
			sh, k = sh.elems(), m[k]+1
		case ts[k].Text == "(" && sh.kind == elemFnShape:
			sh, k = &shape{kind: elemShape, trees: sh.trees}, m[k]+1
		default:
			sh = nil
		}
	}
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

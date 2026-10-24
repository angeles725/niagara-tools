package rcscan

import "github.com/angeles725/niagara-tools/go/internal/jssrc"

// lit is a literal value the page declares in its own code: a scalar (a string, number, template with no
// substitution, true / false / null / undefined), an array or an object. A nil *lit is a value the scan
// cannot prove literal (a name, a call, an expression).
type lit struct {
	arr, obj bool
	elems    []*lit
	fields   map[string]*lit
	dirty    bool // the page writes into it (u.name = x, units.push(...)): no longer provably literal
}

// full reports a value literal all the way down.
func (t *lit) full() bool {
	if t == nil || t.dirty {
		return false
	}
	for _, e := range t.elems {
		if !e.full() {
			return false
		}
	}
	for _, f := range t.fields {
		if !f.full() {
			return false
		}
	}
	return true
}

// holds reports that reading path from t yields literal text: a field the literal declares, down to a
// scalar or a fully literal array or object; any property of a scalar (a string's) is literal too.
func (t *lit) holds(path []string) bool {
	switch {
	case t == nil || t.dirty:
		return false
	case !t.arr && !t.obj:
		return true
	case len(path) == 0, t.arr:
		return t.full()
	}
	f, ok := t.fields[path[0]]
	return ok && f.holds(path[1:])
}

// parseLit reads the literal value at ts[i] and returns it (nil when it is not one) and the index after it.
func parseLit(ts []jssrc.Token, match []int, i int) (*lit, int) {
	if i >= len(ts) {
		return nil, i
	}
	switch t := ts[i]; {
	case t.Kind == jssrc.String || t.Kind == jssrc.Number || t.Kind == jssrc.Template && len(t.Subs) == 0,
		t.Kind == jssrc.Ident && (t.Text == "true" || t.Text == "false" || t.Text == "null" || t.Text == "undefined"):
		return &lit{}, i + 1
	case t.Kind == jssrc.Punct && t.Text == "-" && i+1 < len(ts) && ts[i+1].Kind == jssrc.Number:
		return &lit{}, i + 2
	case t.Kind == jssrc.Punct && (t.Text == "[" || t.Text == "{"):
		v, end := &lit{arr: t.Text == "[", obj: t.Text == "{", fields: map[string]*lit{}}, match[i]
		for j := i + 1; j < end; {
			if ts[j].Text == "," {
				j++
				continue
			}
			key := ""
			if v.obj {
				if j+1 < end && ts[j+1].Text == ":" && ts[j].Kind != jssrc.Template && ts[j].Kind != jssrc.Punct {
					key, j = ts[j].Text, j+2
				} else { // a shorthand, spread, method or computed key: no field the scan can read
					j = jssrc.Expr(ts, j)
					continue
				}
			}
			e, n := parseLit(ts, match, j)
			if n < end && ts[n].Text != "," || n > end {
				e, n = nil, jssrc.Expr(ts, j)
			}
			if v.obj {
				v.fields[key] = e
			} else {
				v.elems = append(v.elems, e)
			}
			j = n
		}
		return v, end + 1
	}
	return nil, jssrc.Expr(ts, i)
}

// mark marks dirty the part of t that a write to path reaches ("*" is a computed key or index).
func (t *lit) mark(path []string) {
	switch {
	case t == nil:
	case len(path) == 0, path[0] == "*" && t.obj, path[0] != "*" && t.arr:
		t.dirty = true
	case path[0] == "*":
		for _, e := range t.elems {
			e.mark(path[1:])
		}
	case t.obj:
		if f, ok := t.fields[path[0]]; ok {
			f.mark(path[1:])
		}
	}
}

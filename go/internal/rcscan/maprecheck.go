package rcscan

import "github.com/angeles725/niagara-tools/go/internal/jssrc"

// site is one cached .map result: its array tree and the callback body it was computed from (an object literal
// opened at from when to is -1, else the scalar expression [from, to)).
type site struct {
	tree     *lit
	from, to int
}

// recheck re-takes, once every write is marked, each cached .map verdict that markWrites itself may have taken
// before a later write dirtied what it reads: a field or scalar no longer computed from literals turns unknown.
// A verdict only ever narrows, so it repeats until nothing changes.
func (s *fileScan) recheck() {
	v := s.js()
	for changed := true; changed; {
		changed = false
		for _, st := range v.sites {
			el := st.tree.elems[0]
			if el == nil {
				continue
			}
			if st.to >= 0 {
				if !s.computed(st.from, st.to, map[int]bool{}) {
					st.tree.elems[0], changed = nil, true
				}
				continue
			}
			fresh := s.object(st.from)
			for k, f := range el.fields {
				if f != nil && fresh.fields[k] == nil {
					el.fields[k], changed = nil, true
				}
			}
		}
	}
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
			// Mutation: RC-object-progress -- a token that starts no expression (`({ ; c: 1 })`) loops forever
			if n := jssrc.Expr(ts, j); n > j {
				j = n
			} else {
				j++
			}
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

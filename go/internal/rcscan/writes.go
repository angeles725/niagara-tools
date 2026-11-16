package rcscan

import "github.com/angeles725/niagara-tools/go/internal/jssrc"

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
	// Mutation: RC-mapped-order -- a .map verdict taken mid-marking keeps a field a later write dirtied literal
	s.recheck()
}

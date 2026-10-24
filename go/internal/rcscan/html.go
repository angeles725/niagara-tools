package rcscan

import "github.com/angeles725/niagara-tools/go/internal/jssrc"

// innerHTMLRow decides an innerhtml-server line the line heuristic flagged (an HTML sink and a `+ a.b` /
// `${a.b` member on the same line, no esc*( call): the row stands when a sink that starts on the line
// concatenates, in its own right-hand side (an innerHTML / outerHTML `=` or `+=` value, or the
// insertAdjacentHTML arguments), a raw member that may hold server data (issue #459 (a)); `+ esc(a.b)` is no
// raw member, and an esc*( call elsewhere in the value excuses none.
// A member on the line outside every sink's value is not the sink's. A line where the tokens show no sink
// keeps the row (fail closed).
func (s *fileScan) innerHTMLRow() bool {
	v := s.js()
	from, to := v.line(s.nr)
	sinks, unsafe := 0, false
	for i := from; i < to; i++ {
		a, b, ok := htmlSink(v.toks, v.match, i)
		if !ok {
			continue
		}
		sinks++
		unsafe = unsafe || s.rawConcat(v.toks, a, b, 0, true)
	}
	return sinks == 0 || unsafe
}

// htmlSink reports an HTML sink at toks[i] and returns its value's token range.
func htmlSink(toks []jssrc.Token, match []int, i int) (int, int, bool) {
	t := toks[i]
	if t.Kind != jssrc.Ident || i+1 >= len(toks) {
		return 0, 0, false
	}
	switch {
	case (t.Text == "innerHTML" || t.Text == "outerHTML") && i > 0 && toks[i-1].Text == "." &&
		toks[i+1].Kind == jssrc.Punct && (toks[i+1].Text == "=" || toks[i+1].Text == "+="):
		return i + 2, jssrc.Expr(toks, i+2), true
	case t.Text == "insertAdjacentHTML" && toks[i+1].Text == "(" && toks[i+1].Kind == jssrc.Punct:
		return i + 2, match[i+1], true
	}
	return 0, 0, false
}

// rawConcat reports, in ts[from:to] and every template substitution in it, a member concatenated into the
// HTML (`+ a.b...`, or a substitution that starts `a.b...`) that safeMember does not clear. Names are looked
// up at their own stream index, or at the template's (at) inside a substitution (top false).
func (s *fileScan) rawConcat(ts []jssrc.Token, from, to, at int, top bool) bool {
	for k := from; k < to; k++ {
		pos := at
		if top {
			pos = k
		}
		switch t := ts[k]; {
		case t.Kind == jssrc.Punct && t.Text == "+" && isMember(ts, k+1) && !s.safeMember(ts, k+1, pos):
			return true
		case t.Kind == jssrc.Template:
			for _, sub := range t.Subs {
				if isMember(sub, 0) && !s.safeMember(sub, 0, pos) || s.rawConcat(sub, 0, len(sub), pos, false) {
					return true
				}
			}
		}
	}
	return false
}

// isMember reports a member read `a.b` starting at ts[j], the shape the line heuristic matched.
func isMember(ts []jssrc.Token, j int) bool {
	return j+2 < len(ts) && ts[j].Kind == jssrc.Ident && ts[j+1].Text == "." && ts[j+1].Kind == jssrc.Punct &&
		ts[j+2].Kind == jssrc.Ident
}

// safeMember reports a member read (ts[j] the base name) that holds no server text: a Math.* value, a
// .length, a number formatted by .toFixed( / .toPrecision(, or literal text of the page's own code.
func (s *fileScan) safeMember(ts []jssrc.Token, j, pos int) bool {
	base := ts[j].Text
	var path []string
	k := j + 1
	for ; k+1 < len(ts) && ts[k].Text == "." && ts[k].Kind == jssrc.Punct && ts[k+1].Kind == jssrc.Ident; k += 2 {
		path = append(path, ts[k+1].Text)
	}
	last := path[len(path)-1]
	call := k < len(ts) && ts[k].Text == "("
	return base == "Math" || last == "length" || call && (last == "toFixed" || last == "toPrecision") ||
		s.literalMember(base, pos, path)
}

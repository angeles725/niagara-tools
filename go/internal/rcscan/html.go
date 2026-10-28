package rcscan

import (
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/jssrc"
)

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

// statusOKMsg is the status-default-ok row text.
const statusOKMsg = `a missing status read as "ok" (fail-open) -- render an unknown state when the status is absent: ` +
	`trust a value only when st === "ok"`

// statusDefaultOK reports, among the tokens that start on the line, a status read that falls back to "ok":
// `<x>.st || "ok"`, `?? 'ok'`, `||= "ok"`, `??= "ok"`, where the name before the operator is a status (st,
// stat, sts, status, or a camelCase name ending in Status). A missing status then reads as healthy, so an
// unlinked or uncommissioned point shows a trusted value (audit 2026-10-03 A7 D2; the 2026-09-24
// phase-2 alarms retro, Δ5: `st = p.st || "ok"` showed an unlinked phase flag as active). Text in a string or a comment is no token.
func (s *fileScan) statusDefaultOK() bool {
	v := s.js()
	from, to := v.line(s.nr)
	ts := v.toks
	for i := from; i < to; i++ {
		t := ts[i]
		if t.Kind != jssrc.Punct || i == 0 || i+1 >= len(ts) {
			continue
		}
		switch t.Text {
		case "||", "??", "||=", "??=":
		default:
			continue
		}
		// Mutation: RC-status-name -- any left operand flags `p.name || "ok"`, a label default, not a status
		if r := ts[i+1]; r.Kind == jssrc.String && asciiLower(r.Text) == "ok" && ts[i-1].Kind == jssrc.Ident && statusName(ts[i-1].Text) {
			return true
		}
	}
	return false
}

// statusName reports a name that holds a point status: st, stat, sts, status (any case), or a camelCase name
// ending in Status (pointStatus).
func statusName(n string) bool {
	switch asciiLower(n) {
	case "st", "stat", "sts", "status":
		return true
	}
	return len(n) > len("Status") && strings.HasSuffix(n, "Status")
}

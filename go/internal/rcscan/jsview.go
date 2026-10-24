package rcscan

import (
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/jssrc"
)

// jsView is a file's JavaScript as jssrc tokens (a .js file, or the <script> elements of an .html file), built
// on first use; a check that cannot find its shape in it keeps the line-heuristic verdict (fail closed).
type jsView struct {
	toks  []jssrc.Token
	first []int // the first token index of each 0-based line (len(toks) past the last token)
	match []int // jssrc.Match(toks)
	binds []jssrc.Binding
}

func (s *fileScan) js() *jsView {
	if s.view != nil {
		return s.view
	}
	lines := s.src.lines
	if s.src.ext == "html" {
		lines = jssrc.HTMLScripts(lines)
	}
	v := &jsView{toks: jssrc.Tokens(lines), first: make([]int, len(lines)+1)}
	v.match = jssrc.Match(v.toks)
	v.binds = jssrc.Bindings(v.toks, v.match)
	k := 0
	for n := range v.first {
		for k < len(v.toks) && v.toks[k].Line < n {
			k++
		}
		v.first[n] = k
	}
	s.view = v
	return v
}

// line is the token range [from, to) of the tokens that start on 1-based line nr.
func (v *jsView) line(nr int) (int, int) {
	if nr < 1 || nr >= len(v.first) {
		return 0, 0
	}
	return v.first[nr-1], v.first[nr]
}

// receiver is the plain name chain (`b`, `this.saveBtn`) before the "." at toks[dot], or "" when the chain is
// not plain (a call or an index result such as `$('x').disabled`).
func receiver(toks []jssrc.Token, dot int) string {
	j := dot - 1
	if j < 0 || toks[j].Kind != jssrc.Ident {
		return ""
	}
	for j >= 2 && toks[j-1].Text == "." && toks[j-2].Kind == jssrc.Ident {
		j -= 2
	}
	if j > 0 && (toks[j-1].Text == "." || toks[j-1].Text == ")" || toks[j-1].Text == "]" || toks[j-1].Text == "?.") {
		return ""
	}
	var b strings.Builder
	for k := j; k < dot; k++ {
		b.WriteString(toks[k].Text)
	}
	return b.String()
}

// memberSet reports `<recv>.<prop> = <value>` at toks[dot] and returns the value's token range.
func memberSet(toks []jssrc.Token, dot int, prop string) (int, int, bool) {
	if dot+2 >= len(toks) || toks[dot].Text != "." || toks[dot+1].Kind != jssrc.Ident || toks[dot+1].Text != prop ||
		toks[dot+2].Kind != jssrc.Punct || toks[dot+2].Text != "=" {
		return 0, 0, false
	}
	return dot + 3, jssrc.Expr(toks, dot+3), true
}

// statedReason reports that every control the line disables states a reason that is not a gate (issue #459
// (b): `b.disabled = true; b.title = "Sin salida cableada";`): each `.disabled = <not false>` on the line has a
// plain receiver whose `.title` the same line sets to a value holding a string literal, and no literal of that
// value mentions a login / auth / step-up gate. A control disabled for a wiring, interlock or phase reason is
// meant to be inert, so the gate's lost click is no defect. A setAttribute("disabled" call or a disabled
// attribute on the line has no reason the scan can read, and neither has a line it finds no assignment on.
func (s *fileScan) statedReason() bool {
	if disabledAttr.MatchString(s.line) || disabledTag.MatchString(s.line) {
		return false
	}
	v := s.js()
	from, to := v.line(s.nr)
	toks, found := v.toks, false
	for i := from; i < to; i++ {
		a, b, ok := memberSet(toks, i, "disabled")
		if !ok || a < b && (toks[a].Text == "false" && toks[a].Kind == jssrc.Ident || toks[a].Text == "0" && toks[a].Kind == jssrc.Number) {
			continue
		}
		if recv := receiver(toks, i); recv == "" || !titled(toks, from, to, recv) {
			return false
		}
		found = true
	}
	return found
}

// titled reports a `<recv>.title = <value>` in toks[from:to] whose value holds a string literal and no literal
// that mentions a gate.
func titled(toks []jssrc.Token, from, to int, recv string) bool {
	for i := from; i < to; i++ {
		a, b, ok := memberSet(toks, i, "title")
		if !ok || receiver(toks, i) != recv {
			continue
		}
		lit := false
		for _, t := range toks[a:b] {
			if t.Kind == jssrc.String || t.Kind == jssrc.Template {
				if mentionsGate(t.Text) {
					return false
				}
				lit = true
			}
		}
		if lit {
			return true
		}
	}
	return false
}

package servlet

import (
	"regexp"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
)

// returnOnly is a catch body whose only statement is `return <expr>;`.
var returnOnly = regexp.MustCompile(`^return[[:space:]]+([^;]*);$`)

// rereads reports whether catch c of try tr re-reads the value its parse failed on instead of substituting a
// default: c's body is a single `return <expr>;`, expr holds no "?" and passes to a call, as an argument, a
// simple name that a parse call (anyParse) inside tr received (`try { return r.get((int)
// Double.parseDouble(tag), false); } catch (NumberFormatException e) { return r.get(tag); }`). What it cannot
// prove: that the second read rejects a bad value (BEnumRange.get(String) throws on an unknown tag; a lookup
// that maps any text to a value would not). It only proves the fallback is a function of the input, not a
// constant or a value unrelated to the request.
func rereads(text string, tr javasrc.Try, c javasrc.Catch) bool {
	m := returnOnly.FindStringSubmatch(strings.TrimSpace(text[c.Body.Open+1 : min(c.Body.Close, len(text))]))
	if m == nil || strings.Contains(m[1], "?") {
		return false
	}
	passed := map[string]bool{} // the names expr passes to a call: name(..., x, ...) with x bare or x.member
	ret := javasrc.Tokens([]string{m[1]})
	for k := 1; k+1 < len(ret); k++ {
		prev, next := ret[k-1].Text, ret[k+1].Text
		if ret[k].Kind == javasrc.TokIdent && (prev == "(" && k >= 2 && ret[k-2].Kind == javasrc.TokIdent || prev == ",") &&
			(next == ")" || next == "," || next == ".") {
			passed[ret[k].Text] = true
		}
	}
	for _, loc := range anyParse.FindAllStringIndex(text, -1) {
		if !(tr.At < loc[0] && loc[0] < tr.Body.Close) {
			continue
		}
		arg := javasrc.Tokens([]string{text[loc[1]:closeParen(text, loc[1]-1)]})
		for k, t := range arg {
			if t.Kind == javasrc.TokIdent && passed[t.Text] && (k == 0 || arg[k-1].Text != ".") &&
				(k+1 == len(arg) || arg[k+1].Text != "(") {
				return true
			}
		}
	}
	return false
}

// closeParen is the offset of the ")" matching the "(" at open (len(s) when it never closes).
func closeParen(s string, open int) int {
	depth := 0
	for k := open; k < len(s); k++ {
		switch s[k] {
		case '(':
			depth++
		case ')':
			if depth--; depth == 0 {
				return k
			}
		}
	}
	return len(s)
}

package servlet

import (
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
)

// fingerprintParams are the query parameters that carry an asset fingerprint (?v=<build or version>).
var fingerprintParams = map[string]bool{`"v"`: true, `"ver"`: true, `"version"`: true}

// fingerprinted reports whether every max-age>0 literal on 0-based line i is the then-branch of a ternary
// whose condition proves the request is fingerprinted (kit #459 cache-nofinger): the condition is a
// fingerprint test (fingerprintTest) itself, or a boolean parameter of the private method holding the line
// that every call of that name in the file passes a fingerprint test (an overload's calls are checked too,
// which can only keep the row). A max-age it cannot tie to such a condition leaves the row in place.
func fingerprinted(toks []javasrc.Token, decls []javasrc.Decl, i int) bool {
	seen := false
	for k, t := range toks {
		if t.Line != i || t.Kind != javasrc.TokLiteral || !maxAge.MatchString(t.Text) {
			continue
		}
		seen = true
		if k < 2 || toks[k-1].Text != "?" {
			return false
		}
		cond := condition(toks, k-1)
		if !fingerprintTest(cond) && !(len(cond) == 1 && cond[0].Kind == javasrc.TokIdent && fingerprintParam(toks, decls, k, cond[0].Text)) {
			return false
		}
	}
	return seen
}

// condition is the ternary condition before the "?" at q: the tokens back to the "(", "," or "=" that opens
// the expression at depth 0.
func condition(toks []javasrc.Token, q int) []javasrc.Token {
	depth := 0
	j := q - 1
	for ; j >= 0; j-- {
		switch toks[j].Text {
		case ")", "]":
			depth++
		case "(", "[":
			if depth == 0 {
				return toks[j+1 : q]
			}
			depth--
		case ",", "=", ";", "{", "}", "?", ":":
			if depth == 0 {
				if toks[j].Text == "=" && (j > 0 && strings.Contains("!=<>", toks[j-1].Text) || j+1 < q && toks[j+1].Text == "=") {
					continue // an "=" of !=, ==, <= or >=
				}
				return toks[j+1 : q]
			}
		}
	}
	return toks[j+1 : q]
}

// fingerprintTest reports whether expr tests that the request carries a fingerprint parameter:
// getParameter("v") != null (or ver / version), with no "||" or "?" that could make it true without one.
func fingerprintTest(expr []javasrc.Token) bool {
	found := false
	for j := range expr {
		if expr[j].Text == "?" || j+1 < len(expr) && expr[j].Text == "|" && expr[j+1].Text == "|" {
			return false
		}
		if j+6 < len(expr) && expr[j].Text == "getParameter" && expr[j+1].Text == "(" && fingerprintParams[expr[j+2].Text] &&
			expr[j+3].Text == ")" && expr[j+4].Text == "!" && expr[j+5].Text == "=" && expr[j+6].Text == "null" {
			found = true
		}
	}
	return found
}

// fingerprintParam reports whether p is a boolean parameter of the private method whose body holds token k
// and every call of its name in the file (at least one) passes a fingerprint test in p's position.
func fingerprintParam(toks []javasrc.Token, decls []javasrc.Decl, k int, p string) bool {
	var d javasrc.Decl
	found := false
	for _, x := range decls {
		if x.Body[0] <= k && k < x.Body[1] && (!found || x.Body[0] > d.Body[0]) {
			d, found = x, true
		}
	}
	if !found || !d.Private {
		return false
	}
	open := -1
	for j := d.Body[0] - 1; j > 0; j-- {
		if toks[j-1].Text == d.Name && toks[j].Text == "(" {
			open = j
			break
		}
	}
	if open < 0 {
		return false
	}
	pos := -1
	for n, arg := range args(toks, open, true) {
		if len(arg) >= 2 && arg[len(arg)-1].Text == p && arg[len(arg)-2].Text == "boolean" {
			pos = n
		}
	}
	if pos < 0 {
		return false
	}
	calls := 0
	for j := 0; j+1 < len(toks); j++ {
		if toks[j].Text != d.Name || toks[j+1].Text != "(" || j+1 == open ||
			j > 0 && toks[j-1].Text == "." && !(j > 1 && toks[j-2].Text == "this") {
			continue
		}
		a := args(toks, j+1, false)
		if pos >= len(a) || !fingerprintTest(a[pos]) {
			return false
		}
		calls++
	}
	return calls > 0
}

// args splits the argument list (or, with generics, the parameter list, whose "<" and ">" nest) whose "("
// is toks[open] at its top-level commas.
func args(toks []javasrc.Token, open int, generics bool) [][]javasrc.Token {
	var out [][]javasrc.Token
	depth, from := 0, open+1
	for j := open; j < len(toks); j++ {
		t := toks[j].Text
		if !generics && (t == "<" || t == ">") {
			continue
		}
		switch t {
		case "(", "[", "{", "<":
			depth++
		case ")", "]", "}", ">":
			if depth--; depth == 0 {
				if j > from {
					out = append(out, toks[from:j])
				}
				return out
			}
		case ",":
			if depth == 1 {
				out = append(out, toks[from:j])
				from = j + 1
			}
		}
	}
	return out
}

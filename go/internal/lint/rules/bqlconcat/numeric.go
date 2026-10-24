package bqlconcat

import "github.com/angeles725/niagara-tools/go/internal/javasrc"

// numeric are the types whose string form can carry no quote and no BQL syntax: a value of one of them
// concatenated into a query is not an injection. char is left out: a char can be a quote.
var numeric = map[string]bool{"byte": true, "short": true, "int": true, "long": true, "float": true,
	"double": true, "boolean": true, "Byte": true, "Short": true, "Integer": true, "Long": true, "Float": true,
	"Double": true, "Boolean": true}

// notType are the keywords that can stand right before a name without declaring it.
var notType = map[string]bool{"return": true, "throw": true, "case": true, "new": true, "assert": true,
	"yield": true, "else": true, "instanceof": true}

// numericOnly reports whether every "+" on 0-based line i joins only literals, numbers, true/false and simple
// names declared with a numeric type (and no other type) in the innermost method or constructor holding the
// line, its parameters included. What it cannot prove stays a finding: a field, a call, a field access, an
// array element, a cast or a parenthesized expression, "++"/"+=", a name declared outside the method.
func numericOnly(toks []javasrc.Token, decls []javasrc.Decl, i int) bool {
	seen := false
	for k, t := range toks {
		if t.Line != i || t.Kind != javasrc.TokPunct || t.Text != "+" {
			continue
		}
		seen = true
		if k == 0 || k+1 >= len(toks) || toks[k-1].Text == "+" || toks[k+1].Text == "+" || toks[k+1].Text == "=" {
			return false
		}
		from, to, ok := scope(toks, decls, k)
		if !ok {
			return false
		}
		left := k-2 < 0 || toks[k-2].Text != "."
		right := k+2 >= len(toks) || (toks[k+2].Text != "(" && toks[k+2].Text != "." && toks[k+2].Text != "[")
		if !safe(toks, from, to, toks[k-1], left) || !safe(toks, from, to, toks[k+1], right) {
			return false
		}
	}
	return seen
}

// safe reports whether operand t (a simple name only when bare is set) is a literal, a number, a boolean
// constant or a name of numeric type in toks[from:to].
func safe(toks []javasrc.Token, from, to int, t javasrc.Token, bare bool) bool {
	switch {
	case t.Kind == javasrc.TokLiteral || t.Kind == javasrc.TokNumber:
		return true
	case t.Kind != javasrc.TokIdent || !bare:
		return false
	case t.Text == "true" || t.Text == "false":
		return true
	}
	declared := false
	for j := from; j+2 < to; j++ {
		if toks[j+1].Text != t.Text || toks[j+1].Kind != javasrc.TokIdent {
			continue
		}
		switch toks[j+2].Text {
		case "=", ";", ",", ")", ":":
		default:
			continue
		}
		typ := toks[j]
		if numeric[typ.Text] {
			declared = true
		} else if (typ.Kind == javasrc.TokIdent && !notType[typ.Text]) || typ.Text == ">" || typ.Text == "]" {
			return false // also declared with a type that is not numeric
		}
	}
	return declared
}

// scope is the token range of the innermost method or constructor holding token k, from its name (so its
// parameters are inside) to its closing brace.
func scope(toks []javasrc.Token, decls []javasrc.Decl, k int) (from, to int, ok bool) {
	var in javasrc.Decl
	for _, d := range decls {
		if d.Body[0] <= k && k < d.Body[1] && (!ok || d.Body[0] > in.Body[0]) {
			in, ok = d, true
		}
	}
	if !ok {
		return 0, 0, false
	}
	from = in.Body[0]
	for j := from - 1; j >= 0; j-- {
		if toks[j].Text == in.Name && toks[j].Line == in.Line {
			from = j
			break
		}
	}
	return from, in.Body[1], true
}

package rcscan

import (
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/jssrc"
)

// jsRefs are the URLs JavaScript names, from its tokens (comments are no token): the string a reference form
// takes (urlArg), and the src= / href= attributes and url() a string or template holds as markup. A
// require/define string is a module id and is no form; a template with a substitution is not read.
func jsRefs(toks []jssrc.Token) []Ref {
	var refs []Ref
	for i, t := range toks {
		if t.Kind != jssrc.String && t.Kind != jssrc.Template {
			continue
		}
		text := strings.NewReplacer(`\"`, `"`, `\'`, `'`).Replace(t.Text)
		if urlArg(toks, i) {
			refs = append(refs, Ref{Line: t.Line + 1, URL: text, col: 1 << 30})
			continue
		}
		for _, r := range textRefs(text, t.Line+1) {
			r.col = 1 << 30
			refs = append(refs, r)
		}
	}
	return refs
}

// urlArg reports a string token that is the whole URL a call or statement takes: fetch("..."), new URL("...",
// base), import("..."), import "...", from "...", and an assignment to .src or .href. A string that goes on
// (`"a" + b`, `"a".x`) is only the start of a URL and is no reference.
func urlArg(toks []jssrc.Token, i int) bool {
	if i+1 < len(toks) && toks[i+1].Kind == jssrc.Punct && (toks[i+1].Text == "+" || toks[i+1].Text == ".") {
		return false
	}
	at := func(k int) string {
		if k < 0 {
			return ""
		}
		return toks[k].Text
	}
	switch p := at(i - 1); {
	case p == "(":
		return isFetch(toks, i-2) || at(i-2) == "URL" && at(i-3) == "new" || at(i-2) == "import"
	case p == "import" || p == "from":
		return toks[i-1].Kind == jssrc.Ident
	case p == "=":
		return (at(i-2) == "src" || at(i-2) == "href") && at(i-3) == "."
	}
	return false
}

// isFetch reports that toks[k] is the global fetch (a bare fetch, or window/self/globalThis.fetch), not the
// fetch method of another object.
func isFetch(toks []jssrc.Token, k int) bool {
	if k < 0 || toks[k].Text != "fetch" {
		return false
	}
	if k == 0 || toks[k-1].Text != "." {
		return true
	}
	return k > 1 && (toks[k-2].Text == "window" || toks[k-2].Text == "self" || toks[k-2].Text == "globalThis")
}

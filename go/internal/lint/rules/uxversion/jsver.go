package uxversion

import (
	"path/filepath"
	"regexp"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/jssrc"
)

// A `?v=` built in JavaScript: a string literal ending in `?v=` or `&v=` followed by `+ NAME` (or `+
// encodeURIComponent(NAME)`). NAME is resolved from a `var|let|const NAME = "<ver>"` in the same file and the
// result is judged like a literal ?v=; a name it cannot resolve is a dynamic-version WARN.

// scriptTokens is the JavaScript tokens of a file: a script's own, an HTML page's inline scripts, none for CSS.
func scriptTokens(path string, lines []string) []jssrc.Token {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".js":
		return jssrc.Tokens(lines)
	case ".css":
		return nil
	}
	return jssrc.Tokens(jssrc.HTMLScripts(strings.Split(blank(htmlComment, strings.Join(lines, "\n")), "\n")))
}

var wrapperFn = regexp.MustCompile(`^(encodeURIComponent|encodeURI)$`)

func isPunct(toks []jssrc.Token, i int, text string) bool {
	return i < len(toks) && toks[i].Kind == jssrc.Punct && toks[i].Text == text
}

func isIdent(toks []jssrc.Token, i int) bool { return i < len(toks) && toks[i].Kind == jssrc.Ident }

// jsVersionRefs are the references of toks whose ?v= is concatenated: resolved ones carry the constant's value,
// unresolved ones are dynamic.
func jsVersionRefs(toks []jssrc.Token) []ref {
	var refs []ref
	for i, t := range toks {
		if t.Kind != jssrc.String || !(strings.HasSuffix(t.Text, "?v=") || strings.HasSuffix(t.Text, "&v=")) ||
			strings.HasPrefix(t.Text, "//") || scheme.MatchString(t.Text) || !isPunct(toks, i+1, "+") || !isIdent(toks, i+2) {
			continue
		}
		at, wrapped := i+2, false
		if wrapperFn.MatchString(toks[at].Text) && isPunct(toks, at+1, "(") && isIdent(toks, at+2) && isPunct(toks, at+3, ")") {
			at, wrapped = at+2, true
		}
		name, end := toks[at].Text, at+1
		expr := name
		for isPunct(toks, end, ".") && isIdent(toks, end+1) { // a member (cfg.ver) is not a constant
			expr, end = expr+"."+toks[end+1].Text, end+2
		}
		if wrapped {
			expr = toks[i+2].Text + "(" + expr + ")"
		}
		r := ref{line: t.Line + 1, tag: t.Line + 1, dynamic: t.Text + " + " + expr}
		if ver, ok := constValue(toks, name); ok && end == at+1 && !isPunct(toks, end, "[") && !isPunct(toks, end, "(") {
			r.dynamic, r.text, r.chk = "", t.Text+ver, t.Text+ver
			if !strings.Contains(t.Text, "?") { // "&v=": a query appended to a URL built elsewhere
				r.chk = "?" + r.text
			}
		}
		refs = append(refs, r)
	}
	return refs
}

// constValue is the string a `var|let|const name = "<ver>"` of toks gives name; not ok when there is none or
// the declarations disagree.
func constValue(toks []jssrc.Token, name string) (string, bool) {
	val, found := "", false
	for i := 0; i+3 < len(toks); i++ {
		if toks[i].Kind != jssrc.Ident || !strings.Contains(" var let const ", " "+toks[i].Text+" ") ||
			toks[i+1].Text != name || !isPunct(toks, i+2, "=") || toks[i+3].Kind != jssrc.String {
			continue
		}
		if found && val != toks[i+3].Text {
			return "", false
		}
		val, found = toks[i+3].Text, true
	}
	return val, found
}

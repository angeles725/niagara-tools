package subscribeleakjs

// Whether a file loads bajaux/mixin/subscriberMixIn as a module id in code.

import "github.com/angeles725/niagara-tools/go/internal/jssrc"

// mixInID is the module id a view loads to get the Subscriber teardown of its widget destroy.
const mixInID = "bajaux/mixin/subscriberMixIn"

// loadsMixIn: the file names mixInID as a module id in code, never in a comment or an unrelated string: an element of
// the dependency array of define([...]), define('name', [...]), require([...]) or requirejs([...]), the argument of
// require('id'), or the source of an `import ... from 'id'` / `import 'id'` / `import('id')`. The mixin's destroy
// detaches the widget's Subscriber and calls unsubscribeAll() on it (subscriberMixIn.js:304-317), so the view needs no
// call of its own. Comments are no tokens, and the id must be the whole string.
func loadsMixIn(raw []string) bool {
	toks := jssrc.Tokens(raw)
	for i, t := range toks {
		if t.Kind != jssrc.Ident {
			continue
		}
		switch t.Text {
		case "define", "require", "requirejs":
			if i+1 < len(toks) && toks[i+1].Text == "(" && callLoads(toks, i+2, t.Text == "require") {
				return true
			}
		case "import":
			if importLoads(toks, i+1) {
				return true
			}
		}
	}
	return false
}

func isID(t jssrc.Token) bool { return t.Kind == jssrc.String && t.Text == mixInID }

// callLoads reads the arguments of define( / require( from toks[k]: an optional name string and a comma, then the
// dependency array; a lone string is a module id only for require.
func callLoads(toks []jssrc.Token, k int, isRequire bool) bool {
	if k < len(toks) && toks[k].Kind == jssrc.String {
		if isRequire && isID(toks[k]) {
			return true
		}
		if k+1 < len(toks) && toks[k+1].Text == "," {
			k += 2 // define('name', [...])
		}
	}
	if k >= len(toks) || toks[k].Text != "[" {
		return false
	}
	for depth := 0; k < len(toks); k++ {
		switch {
		case toks[k].Text == "[" && toks[k].Kind == jssrc.Punct:
			depth++
		case toks[k].Text == "]" && toks[k].Kind == jssrc.Punct:
			if depth--; depth == 0 {
				return false
			}
		case depth == 1 && isID(toks[k]):
			return true
		}
	}
	return false
}

// importLoads: the first string after `import` (toks[k]) before a `;`, within 40 tokens, is mixInID.
func importLoads(toks []jssrc.Token, k int) bool {
	for n := 0; k < len(toks) && n < 40; k, n = k+1, n+1 {
		switch {
		case toks[k].Kind == jssrc.String:
			return isID(toks[k])
		case toks[k].Kind == jssrc.Punct && toks[k].Text == ";":
			return false
		}
	}
	return false
}

package lint

import (
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/ktssrc"
)

// dependencyCalls are the gradle.kts configurations that declare a module dependency.
var dependencyCalls = map[string]bool{"api": true, "nre": true, "implementation": true}

// DeclaredDeps are the module names a gradle.kts token stream declares, in script order: every api, nre or
// implementation call whose first argument is a ":X" string literal or project(":X"). The script is read
// through ktssrc, so comments and string contents are no code, and a call named xapi() or a method call
// (x.api(...)) declares nothing. It is the reader verify-module's phantom-dep check uses
// (go/internal/verifymodule declaredDeps holds the same rule until it calls this one).
func DeclaredDeps(toks []ktssrc.Token) []string {
	var out []string
	at := func(i int, k ktssrc.Kind, text string) bool {
		return i < len(toks) && toks[i].Kind == k && (text == "" || toks[i].Text == text)
	}
	dep := func(i int) (string, bool) {
		if at(i, ktssrc.String, "") && strings.HasPrefix(toks[i].Value, ":") && !toks[i].Template {
			return toks[i].Value[1:], true
		}
		return "", false
	}
	for i, t := range toks {
		if t.Kind != ktssrc.Ident || !dependencyCalls[t.Text] || !at(i+1, ktssrc.Punct, "(") || (i > 0 && at(i-1, ktssrc.Punct, ".")) {
			continue
		}
		if name, ok := dep(i + 2); ok {
			out = append(out, name)
		} else if at(i+2, ktssrc.Ident, "project") && at(i+3, ktssrc.Punct, "(") {
			if name, ok := dep(i + 4); ok {
				out = append(out, name)
			}
		}
	}
	return out
}

package servletwriteguard

import (
	"regexp"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
)

// helperName is the template's write-gate helper (types/servlet-write.md).
const helperName = "requireTargetWrite"

// helperParam: the declaration of the helper and the name of its first parameter.
var helperParam = regexp.MustCompile(helperName + `\(` + sp + `(final[[:space:]]+)?[A-Za-z0-9_$.<>\[\]]+[[:space:]]+(` + ident + `)`)

// helperOK reports whether a requireTargetWrite(...) call counts as a target check: the helper is not declared
// in this file (the lint reads one file, so a helper elsewhere is trusted by name), or every declaration here
// checks a permission on its first parameter, in a condition or an assignment, before it returns.
func helperOK(methods []javasrc.Method, lines []string, al map[int]aliases) bool {
	for _, m := range methods {
		if m.Name != helperName {
			continue
		}
		param := ""
		for i := max(1, m.Start-backScan); i <= m.Start && i <= len(lines); i++ {
			if g := helperParam.FindStringSubmatch(lines[i-1]); g != nil {
				param = g[2]
			}
		}
		if param == "" || !checksName(m, lines, al, param) {
			return false
		}
	}
	return true
}

// backScan is how many lines above a body's "{" the helper's declaration may start.
const backScan = 5

// checksName reports whether body m holds a used, non-self target check naming variable name.
func checksName(m javasrc.Method, lines []string, al map[int]aliases, name string) bool {
	var prev byte
	for i := m.Start; i <= m.End && i <= len(lines); i++ {
		ln := lines[i-1]
		if !isSelf(ln, al[i]) {
			for _, c := range checksOn(ln, prev, false) {
				if c.names[name] {
					return true
				}
			}
		}
		if lc := lastCode(ln); lc != 0 {
			prev = lc
		}
	}
	return false
}

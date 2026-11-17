package servletwriteguard

import (
	"regexp"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
)

var (
	// userDecl: a local or parameter declared BUser (BUser caller = …, (BUser who)).
	userDecl = regexp.MustCompile(`(^|[^A-Za-z0-9_$.])BUser` + sp + `(` + ident + `)` + sp + `[=;,)]`)
	// selfAssign: a local assigned this, super or the user (x = this; x = (BUser) op.getUser();).
	selfAssign = regexp.MustCompile(`(^|[^A-Za-z0-9_$.])(` + ident + `)` + sp + `=` + sp +
		`(\(` + sp + ident + sp + `\)` + sp + `)?(this|super|` + userExpr + `)` + sp + `;`)
)

// aliases are the names a method binds to the user or to the servlet itself.
type aliases map[string]bool

// self reports whether s reads a permission off an alias: alias.getPermissions(cx) or getPermissionsFor(alias).
func (al aliases) self(s string) bool {
	for name := range al {
		q := regexp.QuoteMeta(name)
		if regexp.MustCompile(notQual+q+sp+`\.getPermissions`+anArg).MatchString(s) ||
			regexp.MustCompile(`getPermissionsFor\(`+sp+q+sp+`\)`).MatchString(s) {
			return true
		}
	}
	return false
}

// aliasesByLine maps each 1-based line of a method body to the aliases bound anywhere in that body (a line in
// two bodies, a nested class, gets both sets). Lines outside every method have none.
func aliasesByLine(methods []javasrc.Method, lines []string) map[int]aliases {
	out := map[int]aliases{}
	for _, m := range methods {
		al := aliases{}
		for i := m.Start; i <= m.End && i <= len(lines); i++ {
			for _, g := range userDecl.FindAllStringSubmatch(lines[i-1], -1) {
				al[g[2]] = true
			}
			for _, g := range selfAssign.FindAllStringSubmatch(lines[i-1], -1) {
				al[g[2]] = true
			}
		}
		for i := m.Start; i <= m.End && i <= len(lines); i++ {
			if out[i] == nil {
				out[i] = aliases{}
			}
			for k := range al {
				out[i][k] = true
			}
		}
	}
	return out
}

// isIdentByte reports whether c can be part of a Java identifier.
func isIdentByte(c byte) bool {
	return c == '_' || c == '$' || c >= '0' && c <= '9' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z'
}

// varNames are the variable-like identifiers of expression e: not a method name (followed by '('), not a type
// (an upper-case first letter), not this or super.
func varNames(e string) map[string]bool {
	out := map[string]bool{}
	for _, loc := range identRe.FindAllStringIndex(e, -1) {
		name := e[loc[0]:loc[1]]
		next := strings.TrimLeft(e[loc[1]:], " \t")
		if name == "this" || name == "super" || name == "new" || name[0] >= 'A' && name[0] <= 'Z' ||
			strings.HasPrefix(next, "(") {
			continue
		}
		out[name] = true
	}
	return out
}

var identRe = regexp.MustCompile(ident)

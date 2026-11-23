package servletwriteguard

import (
	"path/filepath"
	"regexp"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
)

// A servlet that is a thin adapter over its dispatch class (types/dashboard.md DUX1) may leave the CSRF token
// check to that class (retro kit-p1-security-guards Δ4). The dispatch class is a type the servlet names in
// code, declared in the servlet's own directory by a file named after it (the DUX1 route/ package) that
// calls verifyCsrfToken( or names CsrfProtectedFilter in code. The lint reads no import and no type
// resolution, so a dispatch class in another directory is not seen, and the row stays.

// csrfTypes maps a directory to the types (file names without .java) of its files that check a CSRF token.
func csrfTypes(files []*javasrc.File) map[string]map[string]bool {
	out := map[string]map[string]bool{}
	for _, f := range files {
		lines, _ := javasrc.BlankLiterals(f.Stripped(), nil)
		if !anyLine(lines, csrfCall.MatchString) {
			continue
		}
		dir := filepath.Dir(f.Path)
		if out[dir] == nil {
			out[dir] = map[string]bool{}
		}
		out[dir][strings.TrimSuffix(filepath.Base(f.Path), ".java")] = true
	}
	return out
}

// namesAny reports whether some line of code names one of the types as an identifier.
func namesAny(lines []string, types map[string]bool) bool {
	for t := range types {
		re := regexp.MustCompile(notQual + regexp.QuoteMeta(t) + `([^A-Za-z0-9_$]|$)`)
		if anyLine(lines, re.MatchString) {
			return true
		}
	}
	return false
}

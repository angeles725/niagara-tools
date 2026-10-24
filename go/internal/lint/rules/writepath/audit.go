package writepath

import (
	"regexp"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/lint"
	"github.com/angeles725/niagara-tools/go/internal/lint/rules/linescan"
)

var (
	handler = regexp.MustCompile(`void[[:space:]]+(do(Post|Put|Delete|Patch))[[:space:]]*\(`)
	callee  = regexp.MustCompile(`([A-Za-z_][A-Za-z0-9_]*)[[:space:]]*\(`)
	getter  = regexp.MustCompile(`^(get|is)[A-Z]`)
)

const auditMsg = ": no audit call (record user, ord, old, new, source before answering success)"

// audit is the audit-call pass over one profile's files: a file that declares a servlet write handler
// (void doPost/doPut/doDelete/doPatch, read in code: comments and literal contents removed) must call an
// identifier containing "audit" (not a get*/is* getter) somewhere in it, or it gets one WARN row at its
// first handler. Contract: types/security.md § Servlet write audit.
func (s *scan) audit(files []string) ([]lint.Finding, error) {
	var out []lint.Finding
	for _, p := range files {
		f, err := s.file(p)
		if err != nil {
			return out, err
		}
		at, method, audited := 0, "", false
		for i, ln := range linescan.Code(f) {
			if m := handler.FindStringSubmatch(ln); at == 0 && m != nil {
				at, method = i+1, m[1]
			}
			for _, c := range callee.FindAllStringSubmatch(ln, -1) {
				audited = audited || strings.Contains(strings.ToLower(c[1]), "audit") && !getter.MatchString(c[1])
			}
		}
		if at > 0 && !audited {
			out = append(out, lint.Finding{Severity: lint.Warn, Path: p, Line: at, Msg: "write handler " + method + auditMsg})
		}
	}
	return out, nil
}

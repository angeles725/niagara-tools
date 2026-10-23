// Package servlet is the lint-servlet rule: a static security lint over the BWebServlet sources of a
// module (Campaign 8 PR12, corpus B813). Port of the awk in toolbelt/lint-servlet.sh (the wrapper keeps the
// user-facing contract and its header documents every check and its mutation pins). Per-file rule: a
// file is a servlet when a line extends BWebServlet or declares a doGet/doPost/service handler, and every
// row is read from that one file, in the awk's order: per handler auth, unbounded-set, log-in-handler and
// api-response-headers; then catch-no-400, input-400, cache-nofinger and csrf-xrw-only over the file.
package servlet

import (
	"regexp"
	"strings"
)

var (
	servletDecl = regexp.MustCompile(`extends[[:space:]]+BWebServlet`)
	handler     = regexp.MustCompile(`void[[:space:]]+(doGet|doPost|service)[[:space:]]*\(`)
	csrf        = regexp.MustCompile(`CsrfUtil|x-niagara-csrfToken|csrfToken`)
	xcto        = regexp.MustCompile(`X-Content-Type-Options|setApiHeaders[[:space:]]*\(|applyHeaders[[:space:]]*\(`)
	write       = regexp.MustCompile(`\.(setDouble|setFloat|setInt|setLong|setRelTime|setString)\(|[^a-zA-Z0-9_]set\(`)
	auth        = regexp.MustCompile(`getRemoteUser\(\)|\.getUser\(\)`)
	typedSet    = regexp.MustCompile(`\.(setDouble|setFloat|setInt|setLong|setRelTime)\(`)
	clamp       = regexp.MustCompile(`getMin\(\)|getMax\(\)|getFacets\(\)|MIN|MAX|clamp`)
	respWrite   = regexp.MustCompile(`getWriter\(\)|getOutputStream\(\)`)
	tryOpen     = regexp.MustCompile(`(^|[[:space:]])try[[:space:]]*\{`)
	anyParse    = regexp.MustCompile(`Double\.parseDouble\(|Integer\.parseInt\(|Long\.parseLong\(|Double\.valueOf\(`)
	catchOpen   = regexp.MustCompile(`catch[[:space:]]*\(`)
	badRequest  = regexp.MustCompile(`[^0-9]400[^0-9]|SC_BAD_REQUEST|sendError\(|setStatus\(|jsonError\(|sendJson\(`)
	parse       = regexp.MustCompile(`Double\.parseDouble\(|Integer\.parseInt\(`)
	maxAge      = regexp.MustCompile(`max-age=[1-9]`)
)

const (
	msgAuth     = "handler writes without getRemoteUser()/getUser() auth gate"
	msgUnbound  = "typed set* in handler without MIN/MAX facet enforcement"
	msgLog      = "LOG.info inside request handler (per-request log spam)"
	msgHeaders  = "response handler sets no headers; the station header providers apply (security.md §7)"
	msgCatch    = "catch around parse without 400 response (silent default may write bad value)"
	msgInput    = "parseDouble/parseInt without enclosing try/catch -> may not return 400 on bad input"
	msgCache    = "Cache-Control max-age>0 on rc asset without fingerprint"
	msgCsrfOnly = "X-Requested-With guard without CsrfUtil/csrfToken (use Niagara CSRF token) [ev: corpus B813]"
)

// cut drops a line's first "//" and everything after it (the awk's line-comment strip).
func cut(s string) string {
	if i := strings.Index(s, "//"); i >= 0 {
		return s[:i]
	}
	return s
}

// block is the text of the brace block that opens at or after line start (0-based): the cut lines, each
// prefixed by " ", from start to the line where the depth counted over the raw lines returns to 0, or to
// the end of the file. needOpen ignores a "}" before the first "{" counted (the catch scan) instead of
// only a "}" at depth 0 (the handler and try scans). end is the closing line, or start when the block
// never closes (the awk's try_end kept its initial value).
func block(lines []string, start int, needOpen bool) (text string, end int) {
	var b strings.Builder
	depth, opened := 0, false
	for j := start; j < len(lines); j++ {
		b.WriteString(" ")
		b.WriteString(cut(lines[j]))
		for _, ch := range []byte(lines[j]) {
			switch {
			case ch == '{':
				depth++
				opened = true
			case ch == '}' && depth > 0 && (opened || !needOpen):
				depth--
				if depth == 0 {
					return b.String(), j
				}
			}
		}
	}
	return b.String(), start
}

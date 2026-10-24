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

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
	"github.com/angeles725/niagara-tools/go/internal/lint"
)

const id = "lint-servlet"

// Spec registers the rule with the argument contract of toolbelt/lint-servlet.sh: the root first, then
// only --strict; rows name the file relative to the root; --strict prints WARN rows as FAIL.
func Spec() lint.Spec {
	return lint.Spec{
		Rule:        rule{},
		Mode:        lint.Trailing,
		Usage:       "usage: lint-servlet.sh <src-dir> [--strict]",
		Select:      lint.Selection{Names: []string{"*.java"}},
		StrictFails: true,
		RelPaths:    true,
		// The K20 row of lib/scan-files.sh scan_files_nonempty_or_exit, which the bash lint printed.
		EmptyError: lint.EmptyRow{Msg: "no files to scan (K20: wrong path or empty scaffold)"},
	}
}

type rule struct{}

func (rule) ID() string              { return id }
func (rule) Severity() lint.Severity { return lint.Warn }

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

// block is the text of the brace block that opens at or after line start (0-based): the lines, each
// prefixed by " ", from start to the line where the depth counted over them returns to 0, or to the end
// of the file. needOpen ignores a "}" before the first "{" counted (the catch scan) instead of
// only a "}" at depth 0 (the handler and try scans). end is the closing line, or start when the block
// never closes (the awk's try_end kept its initial value); closed tells the two apart.
func block(lines []string, start int, needOpen bool) (text string, end int, closed bool) {
	var b strings.Builder
	depth, opened := 0, false
	for j := start; j < len(lines); j++ {
		b.WriteString(" ")
		b.WriteString(lines[j])
		for _, ch := range []byte(lines[j]) {
			switch {
			case ch == '{':
				depth++
				opened = true
			case ch == '}' && depth > 0 && (opened || !needOpen):
				depth--
				if depth == 0 {
					return b.String(), j, true
				}
			}
		}
	}
	return b.String(), start, false
}

// inTry reports whether line i (0-based) lies inside a "try {" block opened on it or above it within the
// same method or class-level lambda body (javasrc.Scopes): a try of another method, or one that closed
// before line i, guards nothing; one that never closes (an unbalanced file) still counts. A line in no
// body has no try.
func inTry(lines []string, scopes []javasrc.Method, i int) bool {
	m, ok := javasrc.Enclosing(scopes, i+1)
	if !ok {
		return false
	}
	for k := i; k >= m.Start-1; k-- {
		if tryOpen.MatchString(lines[k]) {
			if _, end, closed := block(lines, k, false); end >= i || !closed {
				return true // it encloses line i, or stays open to the end of the file
			}
		}
	}
	return false
}

func (rule) Check(f *javasrc.File) ([]lint.Finding, error) {
	// Every check reads code, never a comment: code is the file with comments blanked and literal text
	// kept (the header names and the Cache-Control value are string literals); lines also has the literal
	// contents removed, for the code shapes (calls, braces, try/catch, handler and class declarations), so
	// "}" or "Integer.parseInt(" inside a string is no code either.
	code := f.Stripped()
	lines, _ := javasrc.BlankLiterals(code, nil)
	isServlet := false
	for _, ln := range lines {
		if servletDecl.MatchString(ln) || handler.MatchString(ln) {
			isServlet = true
			break
		}
	}
	if !isServlet {
		return nil, nil
	}
	// The file-level flags and the csrf-xrw-only anchor: a comment (a Javadoc block included) documents,
	// it guards nothing (truth audit 2026-10-22 T8: a Javadoc line was reported as the guard).
	hasXHR, hasCsrf, hasXCTO := false, false, false
	for _, sl := range code {
		hasXHR = hasXHR || strings.Contains(sl, "X-Requested-With")
		hasCsrf = hasCsrf || csrf.MatchString(sl)
		hasXCTO = hasXCTO || xcto.MatchString(sl)
	}

	var out []lint.Finding
	add := func(sev lint.Severity, check string, line int, msg string) {
		out = append(out, lint.Finding{Check: check, Severity: sev, Line: line, Msg: msg})
	}
	for i, ln := range lines {
		if !handler.MatchString(ln) {
			continue
		}
		body, _, _ := block(lines, i, false)
		if write.MatchString(body) && !auth.MatchString(body) {
			add(lint.Fail, "auth", i+1, msgAuth)
		}
		if typedSet.MatchString(body) && !clamp.MatchString(body) {
			add(lint.Unset, "unbounded-set", i+1, msgUnbound)
		}
		if strings.Contains(body, "LOG.info(") {
			add(lint.Unset, "log-in-handler", i+1, msgLog)
		}
		if respWrite.MatchString(body) && !hasXCTO {
			add(lint.Info, "api-response-headers", i+1, msgHeaders)
		}
	}

	for i, ln := range lines {
		if !tryOpen.MatchString(ln) {
			continue
		}
		body, end, _ := block(lines, i, false)
		if !anyParse.MatchString(body) {
			continue
		}
		catch := -1
		// The try's closing line counts: "} catch (E e) {" is the usual shape (the awk began one line
		// later and never read it).
		for k := end; k < len(lines) && k <= end+4; k++ {
			if catchOpen.MatchString(lines[k]) {
				catch = k
				break
			}
		}
		if catch < 0 {
			continue
		}
		if cbody, _, _ := block(lines, catch, true); !badRequest.MatchString(cbody) {
			add(lint.Unset, "catch-no-400", catch+1, msgCatch)
		}
	}

	scopes := f.Scopes()
	for i, ln := range lines {
		if parse.MatchString(ln) && !inTry(lines, scopes, i) {
			add(lint.Fail, "input-400", i+1, msgInput)
		}
	}

	for i, ln := range code {
		if strings.Contains(ln, "Cache-Control") && maxAge.MatchString(ln) {
			add(lint.Unset, "cache-nofinger", i+1, msgCache)
		}
	}

	if hasXHR && !hasCsrf {
		for i, sl := range code {
			if strings.Contains(sl, "X-Requested-With") {
				add(lint.Unset, "csrf-xrw-only", i+1, msgCsrfOnly)
				break
			}
		}
	}
	return out, nil
}

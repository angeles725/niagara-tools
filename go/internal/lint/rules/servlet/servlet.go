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
	anyParse    = regexp.MustCompile(`Double\.parseDouble\(|Integer\.parseInt\(|Long\.parseLong\(|Double\.valueOf\(`)
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

// block is the text of the handler body that opens at or after line start (0-based): the lines, each
// prefixed by " ", from start to the line where the brace depth counted over them returns to 0 (a "}" at
// depth 0 is skipped), or to the end of the file.
func block(lines []string, start int) string {
	var b strings.Builder
	depth := 0
	for j := start; j < len(lines); j++ {
		b.WriteString(" ")
		b.WriteString(lines[j])
		for _, ch := range []byte(lines[j]) {
			switch {
			case ch == '{':
				depth++
			case ch == '}' && depth > 0:
				if depth--; depth == 0 {
					return b.String()
				}
			}
		}
	}
	return b.String()
}

// rethrow is a throw statement in a catch body.
var rethrow = regexp.MustCompile(`(^|[^A-Za-z0-9_$])throw[[:space:]]`)

// parseFailure is the exceptions a parse call throws on bad input (NumberFormatException) and its supertypes.
var parseFailure = map[string]bool{"NumberFormatException": true, "IllegalArgumentException": true,
	"RuntimeException": true, "Exception": true, "Throwable": true}

// decidingCatches are the catch clauses that decide catch-no-400, each once, in try order: for every parse
// call (anyParse) in a try, the first clause catching its failure (parseFailure, a type's last name) on the
// innermost enclosing try that has one, the failure propagating outward past a try without one and past a
// clause that throws (it rethrows or replaces the failure, so an enclosing catch answers); when no enclosing
// try catches it, the first clause of the innermost enclosing try that has clauses, unless a clause threw it
// on. A parse in no try with a catch decides nothing (input-400 owns it).
func decidingCatches(t javasrc.Tries) []javasrc.Catch {
	type key struct{ try, clause int }
	chosen := map[key]bool{}
	for _, loc := range anyParse.FindAllStringIndex(t.Text, -1) {
		first, found, thrown := key{-1, 0}, false, false
		for i := len(t.List) - 1; i >= 0 && !found; i-- { // innermost first: a nested try comes after its outer one
			tr := t.List[i]
			if !(tr.At < loc[0] && loc[0] < tr.Body.Close) || len(tr.Catches) == 0 {
				continue
			}
			if first.try < 0 {
				first = key{i, 0}
			}
			for j, c := range tr.Catches {
				if !catches(c) {
					continue
				}
				if rethrow.MatchString(t.Text[c.Body.Open:min(c.Body.Close+1, len(t.Text))]) {
					thrown = true // the failure leaves this try: an enclosing catch answers it
				} else {
					chosen[key{i, j}], found = true, true
				}
				break
			}
		}
		if !found && !thrown && first.try >= 0 {
			chosen[first] = true
		}
	}
	var out []javasrc.Catch
	for i, tr := range t.List {
		for j, c := range tr.Catches {
			if chosen[key{i, j}] {
				out = append(out, c)
			}
		}
	}
	return out
}

// owner is the try statement clause c is attached to.
func owner(t javasrc.Tries, c javasrc.Catch) javasrc.Try {
	for _, tr := range t.List {
		for _, x := range tr.Catches {
			if x.At == c.At {
				return tr
			}
		}
	}
	return javasrc.Try{}
}

// catches reports whether clause c catches a parse failure.
func catches(c javasrc.Catch) bool {
	for _, ty := range c.Types {
		if parseFailure[ty[strings.LastIndexByte(ty, '.')+1:]] {
			return true
		}
	}
	return false
}

// inTry reports whether offset off (in t.Text) lies inside a try statement, its resource specification or its
// block (javasrc.TryBlocks), whose try keyword is in the same method or class-level lambda body as off
// (javasrc.Scopes, 1-based line): a try of another method, or one that closed before off, guards nothing; one
// that never closes (an unbalanced file) still counts. A line in no body has no try.
func inTry(t javasrc.Tries, scopes []javasrc.Method, line, off int) bool {
	m, ok := javasrc.Enclosing(scopes, line)
	if !ok {
		return false
	}
	start := t.Offset(m.Start)
	for _, tr := range t.List {
		if tr.At >= start && tr.At < off && off < tr.Body.Close {
			return true
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
		body := block(lines, i)
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

	tries := javasrc.TryBlocks(code)
	for _, c := range decidingCatches(tries) {
		if !badRequest.MatchString(tries.Text[c.Body.Open:min(c.Body.Close+1, len(tries.Text))]) &&
			!rereads(tries.Text, owner(tries, c), c) {
			add(lint.Unset, "catch-no-400", tries.Line(c.At), msgCatch)
		}
	}

	scopes := f.Scopes()
	for i, ln := range lines {
		if loc := parse.FindStringIndex(ln); loc != nil && !inTry(tries, scopes, i+1, tries.Offset(i+1)+loc[0]) {
			add(lint.Fail, "input-400", i+1, msgInput)
		}
	}

	toks := javasrc.Tokens(code)
	decls := javasrc.Decls(toks)
	for i, ln := range code {
		if strings.Contains(ln, "Cache-Control") && maxAge.MatchString(ln) && !fingerprinted(toks, decls, i) {
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

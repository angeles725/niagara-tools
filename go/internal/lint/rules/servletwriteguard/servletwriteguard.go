// Package servletwriteguard is the lint-servlet-write-guards rule: the write guards of a servlet's write
// path that lint-servlet does not read (retro 2026-11-10-mcpn4-history-batch-tools Δ6). Over web code only
// (a file naming BWebServlet, HttpServlet, WebOp or HttpServletRequest):
//
//   - self-permission (FAIL): a permission read off the servlet itself (getPermissionsFor(this),
//     this.getPermissions(cx), super.getPermissions(cx), an unqualified getPermissions(cx)) or off the user
//     itself (user.getPermissionsFor(user), getPermissionsFor(cx.getUser()), user.getPermissions(cx)), also
//     through a local of the method bound to either (BUser x, x = op.getUser(), x = this; alias.go): none is
//     the caller's permission on the written target (BIProtected.getPermissions(cx) = cx.getUser().getPermissionsFor(this)).
//   - no-target-permission (WARN): a component write (scope.go: not a JDBC setInt, a List.set, an
//     AtomicReference.set) or invoke(...) with no target check earlier in the same
//     method that names the written receiver and whose result is used (checks.go); a requireTargetWrite
//     declared in the file counts only when it checks its first parameter (helper.go).
//   - no-csrf (WARN): a BWebServlet that writes and calls no verifyCsrfToken( (nor names CsrfProtectedFilter,
//     and has no X-Requested-With guard, which lint-servlet csrf-xrw-only reports, and names no dispatch class of
//     its directory that does: dispatch.go), once per file at the class line.
//
// The toolbelt/lint-servlet-write-guards.sh header is the user-facing contract. Mutation pins of the golden
// cases (testdata/): dropping `this` from selfPermission (truth-self-permission), dropping the user-receiver
// pattern (truth-self-permission), counting a self-permission line as a target check (truth-self-permission),
// ignoring the order on the write's own line (truth-no-target-permission), not accepting requireTargetWrite(
// (truth-target-checks, truth-template-clean), not deferring to X-Requested-With (truth-no-csrf) and reading
// non-web code (truth-scope) each turn a case red; so do dropping `super` or the aliases (truth-self-alias),
// matching a check to any receiver, counting a discarded check, trusting a declared helper by name or not
// treating invoke as a write (truth-target-receiver), and accepting a csrfToken name (truth-no-csrf). The write
// scope (scope.go) is pinned by truth-write-scope: dropping the numeric first argument, the argument count, the
// plain receiver types, the mixed-declaration exclusion, the multi-line unknown or invoke turns it red; the dispatch
// class (dispatch.go) by truth-dispatch-csrf: dropping the same-directory match or the name in code turns it red.
package servletwriteguard

import (
	"path/filepath"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
	"github.com/angeles725/niagara-tools/go/internal/lint"
)

// ID is the lint name.
const ID = "lint-servlet-write-guards"

// Spec registers the rule with the argument contract of toolbelt/lint-servlet-write-guards.sh.
func Spec() lint.Spec {
	return lint.Spec{
		Tree:   rule{},
		Mode:   lint.One,
		Usage:  "usage: lint-servlet-write-guards.sh [--strict] <src-root>",
		Select: lint.Selection{Names: []string{"*.java"}},
	}
}

type rule struct{}

func (rule) ID() string              { return ID }
func (rule) Severity() lint.Severity { return lint.Warn }

// CheckTree checks each file in walk order; the files are read together only for the dispatch classes
// that check the CSRF token for a servlet of their directory (dispatch.go).
func (rule) CheckTree(files []*javasrc.File) ([]lint.Finding, error) {
	csrf := csrfTypes(files)
	var out []lint.Finding
	for _, f := range files {
		for _, fd := range checkFile(f, csrf[filepath.Dir(f.Path)]) {
			fd.Path = f.Path
			out = append(out, fd)
		}
	}
	return out, nil
}

// checkFile is the findings of one file; csrfDispatch are the types of its directory that check the token.
func checkFile(f *javasrc.File, csrfDispatch map[string]bool) []lint.Finding {
	code := f.Stripped()                         // comments blanked, literals kept
	lines, _ := javasrc.BlankLiterals(code, nil) // literal contents blanked too: the code shapes
	if !anyLine(lines, web.MatchString) {
		return nil
	}
	methods := f.Methods()
	al := aliasesByLine(methods, lines)
	var out []lint.Finding
	for i, ln := range lines {
		if isSelf(ln, al[i+1]) {
			out = append(out, lint.Finding{Check: "self-permission", Severity: lint.Fail, Line: i + 1, Msg: msgSelf})
		}
	}
	plain := plainReceivers(lines)
	out = append(out, unguardedWrites(methods, lines, al, plain, helperOK(methods, lines, al))...)
	if row, ok := noCsrf(code, lines, plain, csrfDispatch); ok {
		out = append(out, row)
	}
	return out
}

// unguardedWrites is one no-target-permission row per write line with no target check naming its receiver on
// an earlier line, or earlier on the same line, of the same method (checks.go). A line in two bodies (a nested
// class) reports once.
func unguardedWrites(methods []javasrc.Method, lines []string, al map[int]aliases, plain map[string]bool, helper bool) []lint.Finding {
	var out []lint.Finding
	seen := map[int]bool{}
	for _, m := range methods {
		var seenChecks []check
		var prev byte
		for i := m.Start; i <= m.End && i <= len(lines); i++ {
			ln := lines[i-1]
			var here []check
			if !isSelf(ln, al[i]) {
				here = checksOn(ln, prev, helper)
			}
			if loc := firstWrite(ln, plain); loc != nil && !seen[i] {
				w := varNames(ln[receiverStart(ln, loc[0]):loc[0]])
				if !guarded(seenChecks, here, loc[0], w) {
					seen[i] = true
					out = append(out, lint.Finding{Check: "no-target-permission", Line: i, Msg: msgNoCheck})
				}
			}
			seenChecks = append(seenChecks, here...)
			if lc := lastCode(ln); lc != 0 {
				prev = lc
			}
		}
	}
	return out
}

// guarded reports whether a write at byte at of its line, on receiver variables w, has a check before it:
// on an earlier line of the method, or earlier on its own line.
func guarded(earlier, here []check, at int, w map[string]bool) bool {
	for _, c := range earlier {
		if c.guards(w) {
			return true
		}
	}
	for _, c := range here {
		if c.at < at && c.guards(w) {
			return true
		}
	}
	return false
}

// noCsrf is the no-csrf row: a BWebServlet file with a write and no CSRF check, at its class line.
func noCsrf(code, lines []string, plain, csrfDispatch map[string]bool) (lint.Finding, bool) {
	if anyLine(lines, csrfCall.MatchString) || anyLine(code, xrw.MatchString) || !hasWrite(lines, plain) ||
		namesAny(lines, csrfDispatch) {
		return lint.Finding{}, false
	}
	for i, ln := range lines {
		if bwebServlet.MatchString(ln) {
			return lint.Finding{Check: "no-csrf", Line: i + 1, Msg: msgNoCsrf}, true
		}
	}
	return lint.Finding{}, false
}

// anyLine reports whether some line matches.
func anyLine(lines []string, match func(string) bool) bool {
	for _, ln := range lines {
		if match(ln) {
			return true
		}
	}
	return false
}

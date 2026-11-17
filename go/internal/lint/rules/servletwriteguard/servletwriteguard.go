// Package servletwriteguard is the lint-servlet-write-guards rule: the write guards of a servlet's write
// path that lint-servlet does not read (retro 2026-11-10-mcpn4-history-batch-tools Δ6). Over web code only
// (a file naming BWebServlet, HttpServlet, WebOp or HttpServletRequest):
//
//   - self-permission (FAIL): a permission read off the servlet itself (getPermissionsFor(this),
//     this.getPermissions(cx), an unqualified getPermissions(cx)) or off the user itself
//     (user.getPermissionsFor(user), getPermissionsFor(cx.getUser()), user.getPermissions(cx)): none is the
//     caller's permission on the written target (BIProtected.getPermissions(cx) = cx.getUser().getPermissionsFor(this)).
//   - no-target-permission (WARN): a slot write with no target check earlier in the same method.
//   - no-csrf (WARN): a BWebServlet that writes and checks no CSRF token (and has no X-Requested-With guard,
//     which lint-servlet csrf-xrw-only reports), once per file at the class line.
//
// The toolbelt/lint-servlet-write-guards.sh header is the user-facing contract. Mutation pins of the golden
// cases (testdata/): dropping `this` from selfPermission (truth-self-permission), dropping the user-receiver
// pattern (truth-self-permission), counting a self-permission line as a target check (truth-self-permission),
// ignoring the order on the write's own line (truth-no-target-permission), not accepting requireTargetWrite(
// (truth-target-checks, truth-template-clean), not deferring to X-Requested-With (truth-no-csrf) and reading
// non-web code (truth-scope) each turn a case red.
package servletwriteguard

import (
	"github.com/angeles725/niagara-tools/go/internal/javasrc"
	"github.com/angeles725/niagara-tools/go/internal/lint"
)

// ID is the lint name.
const ID = "lint-servlet-write-guards"

// Spec registers the rule with the argument contract of toolbelt/lint-servlet-write-guards.sh.
func Spec() lint.Spec {
	return lint.Spec{
		Rule:   rule{},
		Mode:   lint.One,
		Usage:  "usage: lint-servlet-write-guards.sh [--strict] <src-root>",
		Select: lint.Selection{Names: []string{"*.java"}},
	}
}

type rule struct{}

func (rule) ID() string              { return ID }
func (rule) Severity() lint.Severity { return lint.Warn }

func (rule) Check(f *javasrc.File) ([]lint.Finding, error) {
	code := f.Stripped()                         // comments blanked, literals kept
	lines, _ := javasrc.BlankLiterals(code, nil) // literal contents blanked too: the code shapes
	if !anyLine(lines, web.MatchString) {
		return nil, nil
	}
	var out []lint.Finding
	for i, ln := range lines {
		if isSelf(ln) {
			out = append(out, lint.Finding{Check: "self-permission", Severity: lint.Fail, Line: i + 1, Msg: msgSelf})
		}
	}
	out = append(out, unguardedWrites(f.Methods(), lines)...)
	if row, ok := noCsrf(code, lines); ok {
		out = append(out, row)
	}
	return out, nil
}

// unguardedWrites is one no-target-permission row per write line with no target check on an earlier line,
// or earlier on the same line, of the same method. A line in two bodies (a nested class) reports once.
func unguardedWrites(methods []javasrc.Method, lines []string) []lint.Finding {
	var out []lint.Finding
	seen := map[int]bool{}
	for _, m := range methods {
		guarded := false
		for i := m.Start; i <= m.End && i <= len(lines); i++ {
			ln := lines[i-1]
			loc := write.FindStringIndex(ln)
			if checkAt := targetCheck.FindStringIndex(ln); checkAt != nil && !isSelf(ln) &&
				(loc == nil || checkAt[0] < loc[0]) {
				guarded = true
			}
			if loc != nil && !guarded && !seen[i] {
				seen[i] = true
				out = append(out, lint.Finding{Check: "no-target-permission", Line: i, Msg: msgNoCheck})
			}
		}
	}
	return out
}

// noCsrf is the no-csrf row: a BWebServlet file with a write and no CSRF check, at its class line.
func noCsrf(code, lines []string) (lint.Finding, bool) {
	if anyLine(code, csrf.MatchString) || !anyLine(lines, write.MatchString) {
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

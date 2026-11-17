package servletwriteguard

import "regexp"

// The code shapes, matched on lines whose comments and string/char literal contents are blanked, except
// csrf, which is matched on lines that keep the literals (a header name is a string literal).
const (
	ident     = `[A-Za-z_][A-Za-z0-9_]*`
	sp        = `[[:space:]]*`
	userExpr  = `(` + ident + `(\.` + ident + `)*\.)?getUser\(\)` // op.getUser(), cx.getUser(), getUser()
	userVar   = `[A-Za-z0-9_]*[Uu]ser`                            // user, currentUser, u.user
	notQual   = `(^|[^A-Za-z0-9_$.])`                             // no identifier character or '.' before
	anArg     = `\(` + sp + `[A-Za-z_]`                           // a call with a first argument
	stmtStart = `(^|[=(!,&|?:;{}]|(^|[^A-Za-z0-9_$])return)` + sp // where an unqualified call is a call, not a declaration
)

var (
	// web: the file is web code (a servlet, a route handler taking a WebOp, a web-module servlet).
	web = regexp.MustCompile(`(^|[^A-Za-z0-9_$])(BWebServlet|HttpServlet|WebOp|HttpServletRequest)([^A-Za-z0-9_$]|$)`)
	// bwebServlet: the class declaration line that the no-csrf row anchors at.
	bwebServlet = regexp.MustCompile(`extends[[:space:]]+BWebServlet([^A-Za-z0-9_$]|$)`)

	// selfPermission: a permission read off the servlet itself or off the user itself (BUser is a component
	// too: BUser.getPermissions(cx) is the caller's permission ON THE USER, BUser.java:810).
	selfPermission = []*regexp.Regexp{
		regexp.MustCompile(`getPermissionsFor\(` + sp + `(this|` + userExpr + `|` + userVar + `)` + sp + `\)`),
		regexp.MustCompile(stmtStart + `getPermissions` + anArg),
		regexp.MustCompile(`(^|[^A-Za-z0-9_$.])(this|super)\.getPermissions` + anArg),
		regexp.MustCompile(`(` + userExpr + `|` + notQual + userVar + `)\.getPermissions` + anArg),
	}

	// targetCall: a permission check that can name the written target (unless it is a selfPermission): the
	// receiver of getPermissions(cx) / canWrite(), the argument of getPermissionsFor(x), or the first argument
	// of requireTargetWrite(x, ...) is what it checks (checks.go).
	targetCall = regexp.MustCompile(`getPermissionsFor\(|\.getPermissions` + anArg + `|\.canWrite\(` + sp + `\)|` +
		notQual + `requireTargetWrite\(`)

	// write: a component slot write, or an action invoked on a component.
	write = regexp.MustCompile(`\.(set|setBoolean|setDouble|setFloat|setInt|setLong|setRelTime|setString|invoke)\(`)

	// csrfCall: a CSRF token check in code (literals blanked): a verifyCsrfToken( call or the framework filter.
	// A name alone (a local called csrfToken) is no check.
	csrfCall = regexp.MustCompile(`verifyCsrfToken` + sp + `\(|CsrfProtectedFilter`)
	// xrw: the X-Requested-With guard that lint-servlet csrf-xrw-only already reports (a string literal).
	xrw = regexp.MustCompile(`X-Requested-With`)
)

const (
	doc        = " (types/servlet-write.md)"
	msgSelf    = "permission read off the servlet or the user itself, not the written target: use target.getPermissions(cx) or cx.getUser().getPermissionsFor(target)" + doc
	msgNoCheck = "write with no permission check on its target earlier in this method: check target.getPermissions(cx).hasOperatorWrite() (or getPermissionsFor(target), ordTarget.canWrite(), requireTargetWrite) before the write" + doc
	msgNoCsrf  = "servlet writes but verifies no CSRF token: call CsrfUtil.verifyCsrfToken(op.getRequest()) before the write, or map CsrfProtectedFilter on its write verbs" + doc
)

// isSelf reports whether line s reads a permission off the servlet or the user itself, directly or through
// one of the method's aliases (alias.go).
func isSelf(s string, al aliases) bool {
	for _, re := range selfPermission {
		if re.MatchString(s) {
			return true
		}
	}
	return al.self(s)
}

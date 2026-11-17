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
		regexp.MustCompile(`(^|[^A-Za-z0-9_$.])this\.getPermissions` + anArg),
		regexp.MustCompile(`(` + userExpr + `|` + notQual + userVar + `)\.getPermissions` + anArg),
	}

	// targetCheck: a permission check that can name the written target (unless it is a selfPermission).
	targetCheck = regexp.MustCompile(`getPermissionsFor\(|\.getPermissions` + anArg + `|\.canWrite\(` + sp + `\)|` +
		notQual + `requireTargetWrite\(`)

	// write: a component slot write.
	write = regexp.MustCompile(`\.(set|setBoolean|setDouble|setFloat|setInt|setLong|setRelTime|setString)\(`)

	// csrf: a CSRF token check (CsrfUtil, the token header or parameter, the framework filter) or the
	// X-Requested-With guard that lint-servlet csrf-xrw-only already reports.
	csrf = regexp.MustCompile(`CsrfUtil|x-niagara-csrfToken|csrfToken|CsrfProtectedFilter|X-Requested-With`)
)

const (
	doc        = " (types/servlet-write.md)"
	msgSelf    = "permission read off the servlet or the user itself, not the written target: use target.getPermissions(cx) or cx.getUser().getPermissionsFor(target)" + doc
	msgNoCheck = "write with no permission check on its target earlier in this method: check target.getPermissions(cx).hasOperatorWrite() (or getPermissionsFor(target), ordTarget.canWrite(), requireTargetWrite) before the write" + doc
	msgNoCsrf  = "servlet writes but verifies no CSRF token: call CsrfUtil.verifyCsrfToken(op.getRequest()) before the write, or map CsrfProtectedFilter on its write verbs" + doc
)

// isSelf reports whether line s reads a permission off the servlet or the user itself.
func isSelf(s string) bool {
	for _, re := range selfPermission {
		if re.MatchString(s) {
			return true
		}
	}
	return false
}

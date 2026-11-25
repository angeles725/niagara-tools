// Package postpermission is the lint-post-permission rule: a request, RPC or action handler that calls
// post( with no permission check before it in the same method. BComponent.post(action, arg, cx) queues the
// action and the queued invocation runs with a null Context, so the caller's permission is never checked
// (types/security.md section 2.4). The wrapper toolbelt/lint-post-permission.sh documents the rule, the
// accepted guard forms, the waiver and the limits.
package postpermission

import (
	"regexp"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
	"github.com/angeles725/niagara-tools/go/internal/lint"
)

const id = "lint-post-permission"

// Spec registers the rule with the argument contract of toolbelt/lint-post-permission.sh (the shape of
// lint-servlet: a source directory, rows relative to it, --strict accepted and without effect on a FAIL).
func Spec() lint.Spec {
	return lint.Spec{
		Rule:       rule{},
		Mode:       lint.Trailing,
		Usage:      "usage: lint-post-permission.sh <src-dir> [--strict]",
		Select:     lint.Selection{Names: []string{"*.java"}},
		RelPaths:   true,
		EmptyError: lint.EmptyRow{Msg: "no files to scan (K20: wrong path or empty scaffold)"},
	}
}

type rule struct{}

func (rule) ID() string              { return id }
func (rule) Severity() lint.Severity { return lint.Fail }

// servletMethods are the servlet entry points, handlers by name alone.
var servletMethods = map[string]bool{"doGet": true, "doPost": true, "doPut": true, "doDelete": true,
	"doHead": true, "doOptions": true, "service": true}

// requestTypes: a parameter of one of these types makes a method a request handler.
var requestTypes = map[string]bool{"WebOp": true, "HttpServletRequest": true, "HttpServletResponse": true}

// doAction is an action body: the framework calls do<Action>(..., Context) for an invoke.
var doAction = regexp.MustCompile(`^do[A-Z]`)

// guards are the permission checks that exist in the extracted baja API (the wrapper header cites each).
var guards = map[string]bool{"checkInvoke": true, "checkWrite": true, "getPermissionsFor": true, "canInvoke": true,
	"hasOperatorInvoke": true, "hasAdminInvoke": true}

// permissionArgGuards are guards only when their argument list names an invoke or write permission
// (strongPermissions): BUser.check(target, BPermissions) and BPermissions.has(BPermissions|int). A bare
// check(x) or has(x) is any method of that name, and a read permission is no authority to invoke.
var permissionArgGuards = map[string]bool{"check": true, "has": true}

// strongPermissions are the BPermissions instances (BPermissions.java:24-29: operatorInvoke, adminInvoke,
// operatorWrite, adminWrite) and int masks (:18-23: OPERATOR_INVOKE, ADMIN_INVOKE, OPERATOR_WRITE,
// ADMIN_WRITE) that grant invoke or write. operatorRead, adminRead, OPERATOR_READ and ADMIN_READ are absent.
var strongPermissions = map[string]bool{"operatorInvoke": true, "adminInvoke": true, "operatorWrite": true,
	"adminWrite": true, "OPERATOR_INVOKE": true, "ADMIN_INVOKE": true, "OPERATOR_WRITE": true, "ADMIN_WRITE": true}

// waiver is the comment that accepts one post: on its line, or on a comment-only line above it, with a reason.
var waiver = regexp.MustCompile(`post-permission:[ \t]*ok[ \t]+[^ \t*/]`)

const message = "runs the queued action with no RBAC Context and no permission check comes before it in this " +
	"method: check the caller first (checkInvoke, checkWrite, getPermissionsFor, canInvoke, hasOperatorInvoke, " +
	"hasAdminInvoke; types/security.md section 2.4), or waive it with a '// post-permission: ok <reason>' comment"

func (rule) Check(f *javasrc.File) ([]lint.Finding, error) {
	code, comments, _ := javasrc.StripComments(f.Raw)
	toks := javasrc.Tokens(code)
	if !hasPost(toks) {
		return nil, nil
	}
	decls := javasrc.Decls(toks)
	isHandler := make([]bool, len(decls))
	for i, d := range decls {
		isHandler[i] = handler(d, toks)
	}
	var out []lint.Finding
	for di, d := range decls {
		if !isHandler[di] {
			continue
		}
		guarded := false
		for k := d.Body[0]; k < d.Body[1]; k++ {
			// A handler of an inner class is judged on its own; any other inner method (an anonymous
			// Runnable's run()) is part of this handler: the checks before it guard it.
			if end, nested := nestedHandler(decls, isHandler, di, k); nested {
				k = end - 1
				continue
			}
			t := toks[k]
			if t.Kind != javasrc.TokIdent || k+1 >= d.Body[1] || toks[k+1].Text != "(" {
				continue
			}
			args, _ := arguments(toks, k+1)
			switch {
			case guards[t.Text], permissionArgGuards[t.Text] && namesStrongPermission(args):
				guarded = true
			case t.Text == "post" && isRequestPost(toks, k, args):
				if !guarded && !waived(code, comments, t.Line) {
					out = append(out, lint.Finding{Line: t.Line + 1, Msg: "post() in '" + d.Name + "' " + message})
				}
			}
		}
	}
	return out, nil
}

func hasPost(toks []javasrc.Token) bool {
	for _, t := range toks {
		if t.Kind == javasrc.TokIdent && t.Text == "post" {
			return true
		}
	}
	return false
}

// handler reports whether d serves a request, an RPC or an action invoked with a Context: the callers whose
// permission the queued action would skip.
func handler(d javasrc.Decl, toks []javasrc.Token) bool {
	if servletMethods[d.Name] || rpc(d, toks) {
		return true
	}
	hasContext := false
	for _, p := range d.Params {
		switch simple(p) {
		case "Context":
			hasContext = true
		default:
			if requestTypes[simple(p)] {
				return true
			}
		}
	}
	return hasContext && doAction.MatchString(d.Name)
}

// simple is the last segment of a type name, generics dropped ("javax.baja.sys.Context" is "Context").
func simple(typ string) string {
	if i := strings.IndexByte(typ, '<'); i >= 0 {
		typ = typ[:i]
	}
	return typ[strings.LastIndexByte(typ, '.')+1:]
}

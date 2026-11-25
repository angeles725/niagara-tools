package actionhandler

import (
	"strings"
	"unicode"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
	"github.com/angeles725/niagara-tools/go/internal/lint"
)

const (
	cite    = "ComplexIntrospector.java"
	noDoErr = "-- the station refuses the type with \"Missing action do method\" (" + cite + ":316-343, types/actions.md § 1)"
	statErr = "must not be static -- the station refuses the type with \"Do action method must be not be static\" (" + cite + ":339)"
	retErr  = "-- the station refuses the type with \"Action method and do method have mismatched signature\" (" + cite + ":341)"
)

// withAncestors adds the do methods of c's superclasses declared in the same file to ms.
func withAncestors(toks []javasrc.Token, classes []javasrc.Class, c javasrc.Class, ms []method) []method {
	out := append([]method(nil), ms...)
	seen := map[string]bool{c.Name: true}
	for up := c.Extends; up != "" && !seen[up]; {
		seen[up] = true
		next := ""
		for _, k := range classes {
			if k.Name == up {
				out = append(out, doMethods(toks, k)...)
				next = k.Extends
				break
			}
		}
		up = next
	}
	return out
}

// shape reports whether m takes the parameters the action declares, with a trailing Context when ctx.
func shape(m method, a Action, ctx bool) bool {
	n := 0
	if a.Param != "" {
		n = 1
	}
	if ctx {
		n++
	}
	if len(m.Params) != n || (ctx && m.Params[n-1] != "Context") {
		return false
	}
	return a.Param == "" || a.Opaque || m.Params[0] == simpleName(a.Param)
}

// simpleName is the last dotted segment of a class name written in an annotation.
func simpleName(s string) string { return s[strings.LastIndex(s, ".")+1:] }

// handlerRows checks each named action of c against the do methods in ms (the class's, then its ancestors').
func handlerRows(c javasrc.Class, acts []Action, ms []method) []lint.Finding {
	var out []lint.Finding
	for _, a := range acts {
		if a.Name == "" {
			continue
		}
		do := "do" + string(unicode.ToUpper(rune(a.Name[0]))) + a.Name[1:]
		var named []method
		for _, m := range ms {
			if m.Name == do {
				named = append(named, m)
			}
		}
		var got *method
		for _, ctx := range []bool{false, true} {
			for i, m := range named {
				if got == nil && m.Public && shape(m, a, ctx) {
					got = &named[i]
				}
			}
		}
		row := func(line int, msg string) { out = append(out, lint.Finding{Line: line + 1, Msg: msg}) }
		switch {
		case got == nil:
			row(a.Line, missing(a, do, named))
		case got.Static:
			row(got.Line, "@NiagaraAction \""+a.Name+"\": "+do+" "+statErr)
		case !a.Opaque && got.Ret != retOf(a):
			row(got.Line, "@NiagaraAction \""+a.Name+"\": "+do+" returns "+got.Ret+" but the action returns "+retOf(a)+" "+retErr)
		}
	}
	return out
}

// retOf is the return type an action's wrapper declares.
func retOf(a Action) string {
	if a.Return == "" {
		return "void"
	}
	return simpleName(a.Return)
}

// missing words the row for an action with no usable do method, listing the same-named methods it ignored.
func missing(a Action, do string, named []method) string {
	arg, found := "", ""
	if a.Param != "" {
		arg = simpleName(a.Param) + " arg"
	}
	if len(named) > 0 {
		var fs []string
		for _, m := range named {
			f := sig(m)
			if !m.Public {
				f += " not public"
			}
			fs = append(fs, f)
		}
		found = " (found: " + strings.Join(fs, ", ") + ")"
	}
	ctx := "Context cx"
	if arg != "" {
		ctx = arg + ", Context cx"
	}
	return "@NiagaraAction \"" + a.Name + "\" has no matching do method" + found + ": declare public " + retOf(a) + " " +
		do + "(" + ctx + "), or " + do + "(" + arg + ") " + noDoErr
}

// pairRows reports each doX(..., Context) beside a no-Context doX(...) of the same other parameters.
func pairRows(ms []method) []lint.Finding {
	var out []lint.Finding
	for _, m := range ms {
		n := len(m.Params)
		if n == 0 || m.Params[n-1] != "Context" {
			continue
		}
		for _, o := range ms {
			if o.Name == m.Name && strings.Join(o.Params, ",") == strings.Join(m.Params[:n-1], ",") {
				out = append(out, lint.Finding{Severity: lint.Warn, Line: m.Line + 1, Msg: m.Name + "(..., Context) never runs: the station binds the action to the no-Context " +
					m.Name + "(...) it finds first; keep one handler (types/actions.md § 1)"})
				break
			}
		}
	}
	return out
}

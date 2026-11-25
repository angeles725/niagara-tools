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

// framework are the javax.baja base classes a module's own action never has its handler in: a class that ends its
// superclass chain at one of them is judged, any other end of the chain may hold the handler.
var framework = map[string]bool{"BObject": true, "BValue": true, "BSimple": true, "BComplex": true, "BStruct": true,
	"BComponent": true, "BAbstractService": true, "BFolder": true, "BUnrestrictedFolder": true}

// withAncestors adds the do methods of c's superclasses declared in the same file to ms, and names the first
// superclass of the chain that is neither declared in the file nor a framework base ("" when the chain ends
// at one, or at Object).
func withAncestors(toks []javasrc.Token, classes []javasrc.Class, c javasrc.Class, ms []method) ([]method, string) {
	out := append([]method(nil), ms...)
	seen := map[string]bool{c.Name: true}
	for up := c.Extends; up != "" && !seen[up]; {
		seen[up] = true
		next, found := "", false
		for _, k := range classes {
			if k.Name == up {
				out = append(out, doMethods(toks, k)...)
				next, found = k.Extends, true
				break
			}
		}
		if !found {
			if framework[up] {
				return out, ""
			}
			return out, up
		}
		up = next
	}
	return out, ""
}

// shape reports whether m takes the parameters the action declares, with a trailing Context when ctx. An action
// whose parameterType is unknown (a constant, a baja: spec) accepts the four shapes: no parameter or one,
// with or without the Context.
func shape(m method, a Action, ctx bool) bool {
	n := len(m.Params)
	if ctx && (n == 0 || m.Params[n-1] != "Context") {
		return false
	}
	if ctx {
		n--
	}
	switch {
	case a.ParamUnknown:
		return n <= 1
	case a.Param == "":
		return n == 0
	}
	return n == 1 && m.Params[0] == simpleName(a.Param)
}

// simpleName is the last dotted segment of a class name written in an annotation.
func simpleName(s string) string { return s[strings.LastIndex(s, ".")+1:] }

// handlerRows checks each named action of c against the do methods in ms (the class's, then its ancestors').
// An action with no usable do method is a FAIL, or a WARN when the chain ends at a superclass declared in another
// file (far, its name), which may hold the handler.
func handlerRows(acts []Action, ms []method, far string) []lint.Finding {
	var out []lint.Finding
	for _, a := range acts {
		if a.Name == "" {
			continue
		}
		do := doName(a.Name)
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
		case got == nil && far != "":
			out = append(out, lint.Finding{Severity: lint.Warn, Line: a.Line + 1, Msg: "@NiagaraAction \"" + a.Name +
				"\" has no do method in this file: handler may be inherited from " + far + ": not judged (the station needs a public " +
				do + "(...) in the class or a superclass; " + cite + ":316-343)"})
		case got == nil:
			row(a.Line, missing(a, do, named))
		case got.Static:
			row(got.Line, "@NiagaraAction \""+a.Name+"\": "+do+" "+statErr)
		case !a.RetOpaque && got.Ret != retOf(a):
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
	} else if a.ParamUnknown {
		arg = "<parameterType> arg"
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

// pairRows reports each public doX(..., Context) beside a public no-Context doX(...) of the same other
// parameters, for an action X declared on the class (the introspector sees public methods only).
func pairRows(acts []Action, ms []method) []lint.Finding {
	declared := map[string]bool{}
	for _, a := range acts {
		if a.Name != "" {
			declared[doName(a.Name)] = true
		}
	}
	var out []lint.Finding
	for _, m := range ms {
		n := len(m.Params)
		if !m.Public || !declared[m.Name] || n == 0 || m.Params[n-1] != "Context" {
			continue
		}
		for _, o := range ms {
			if o.Public && o.Name == m.Name && strings.Join(o.Params, ",") == strings.Join(m.Params[:n-1], ",") {
				out = append(out, lint.Finding{Severity: lint.Warn, Line: m.Line + 1, Msg: m.Name + "(..., Context) never runs: the station binds the action to the no-Context " +
					m.Name + "(...) it finds first; keep one handler (types/actions.md § 1)"})
				break
			}
		}
	}
	return out
}

// doName is the handler name of the action name: reset -> doReset.
func doName(name string) string { return "do" + string(unicode.ToUpper(rune(name[0]))) + name[1:] }

package asyncblock

import (
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
	"github.com/angeles725/niagara-tools/go/internal/lint/rules/actionhandler"
)

// simple is a parameter type's last dotted name without generics or array marks: javax.baja.sys.Context is
// "Context", java.util.List<String> is "List".
func simple(t string) string {
	if i := strings.IndexByte(t, '<'); i >= 0 {
		t = t[:i]
	}
	return t[strings.LastIndex(t, ".")+1:]
}

// shape reports whether d takes the parameters the action declares, with a trailing Context when ctx; the same
// choice as the actionhandler rule: an action whose parameterType is unknown accepts no parameter or one.
func shape(d javasrc.Decl, a actionhandler.Action, ctx bool) bool {
	ps := d.Params
	if ctx {
		if len(ps) == 0 || simple(ps[len(ps)-1]) != "Context" {
			return false
		}
		ps = ps[:len(ps)-1]
	}
	switch {
	case a.ParamUnknown:
		return len(ps) <= 1
	case a.Param == "":
		return len(ps) == 0
	}
	return len(ps) == 1 && simple(ps[0]) == simple(a.Param)
}

// handler picks, among the same-named do methods ds, the overload the introspector runs for a: the no-Context
// shapes first (doX() or doX(P) is found before doX(Context) or doX(P, Context), ComplexIntrospector G1), then
// the Context ones. A private method is never found; none matching reports false (the actionhandler rule FAILs it).
func handler(ds []javasrc.Decl, a actionhandler.Action) (javasrc.Decl, bool) {
	for _, ctx := range []bool{false, true} {
		for _, d := range ds {
			if !d.Private && shape(d, a, ctx) {
				return d, true
			}
		}
	}
	return javasrc.Decl{}, false
}

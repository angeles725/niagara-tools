package writeops

import (
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// tplModulesInstalled refuses, before a token, a deploy that adds a type the station's type
// registry does not describe (left out of the reply, or answered {"isErr": true} by 4.15): its
// module is not installed, so the station would refuse the add halfway. The refusal names the
// module, the type, where the template uses it and the manifest dependency that names the
// module.
func tplModulesInstalled(tl plan.TypeLoader, body *pyjson.Object, deps []string) error {
	specs, where := tplTypes(body)
	reply, err := tl.LoadTypes(specs)
	if err != nil {
		return err
	}
	got, _ := reply.(*pyjson.Object)
	for _, spec := range specs {
		var entry *pyjson.Object
		if got != nil {
			entry = asObj(get(got, spec))
		}
		if entry != nil && get(entry, "isErr") != true {
			continue
		}
		module, _, _ := strings.Cut(spec, ":")
		msg := "the station does not have module " + pyjson.StrRepr(module) + " installed: type " + spec +
			" of " + where[spec] + " is unknown to its type registry"
		if dep := depOf(module, deps); dep != "" {
			msg += " (manifest dependency " + dep + ")"
		}
		return plan.ToolError(msg + "; install the module, then deploy again")
	}
	return nil
}

// depOf is the manifest dependency of module (a part such as control-rt, or the module itself).
func depOf(module string, deps []string) string {
	for _, d := range deps {
		part, _, _ := strings.Cut(d, " ")
		if part == module || strings.HasPrefix(part, module+"-") {
			return d
		}
	}
	return ""
}

package writeops

import (
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// repoint is a copy of body whose ConfigBindings name, in targetOrd, the slot ORD of the component
// inside the copy (targets: the binding's name -> that ORD). A plain copy keeps the golden's
// "h:<n>" handle, which resolves to the golden's own component: a config set on the copy would
// change the golden. A binding the body lacks (the strip left it out) is refused, never skipped.
func repoint(body *pyjson.Object, targets map[string]string) (*pyjson.Object, error) {
	out, done := cloneObject(body), 0
	for _, b := range objects(get(kidNamed(out, "templateConfig"), "s")) {
		if to, ok := targets[name(b)]; ok && get(b, "t") == configBindingType {
			setChild(b, "targetOrd", "baja:Ord", to)
			done++
		}
	}
	if done != len(targets) {
		return nil, plan.ToolError("the copy body lacks a ConfigBinding of the golden's templateConfig (the copy leaves " +
			"out slots flagged readonly or transient); its binding would keep pointing at the golden")
	}
	for _, b := range objects(get(kidNamed(out, "templateConfig"), "s")) {
		if t := kidNamed(b, "targetOrd"); t != nil && strings.HasPrefix(pyjson.Str(get(t, "v")), "h:") {
			return nil, plan.ToolError("the golden's templateConfig child " + pyjson.StrRepr(name(b)) + " (type " +
				pyjson.Str(get(b, "t")) + ") names a target by handle and is no " + configBindingType +
				": a copy of it would keep pointing at the golden")
		}
	}
	return out, nil
}

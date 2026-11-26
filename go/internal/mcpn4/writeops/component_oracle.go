package writeops

import (
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// registryOracle classifies the types of bodies (snapshot bodies of the re-add ops) by the station's
// type registry, reading them in one call (B1200-G3: a struct is a slot value, not a component). A
// client that cannot read the registry channel gets nil, which leaves every type to the naming rule.
func registryOracle(c plan.Client, bodies ...any) *plan.RegistryOracle {
	tl, ok := c.(plan.TypeLoader)
	if !ok {
		return nil
	}
	o := plan.NewRegistryOracle(plan.NewContracts(tl))
	var specs []string
	for _, b := range bodies {
		collectBodyTypes(b, &specs)
	}
	o.Preload(specs)
	return o
}

// collectBodyTypes appends the type of every node of a snapshot body, the typed leaves included.
func collectBodyTypes(node any, specs *[]string) {
	n, ok := node.(*pyjson.Object)
	if !ok {
		return
	}
	if t, ok := get(n, "t").(string); ok && t != "" {
		*specs = append(*specs, t)
	}
	kids, _ := get(n, "s").([]any)
	for _, k := range kids {
		collectBodyTypes(k, specs)
	}
}

// opBodies are the bodies of the add ops of ops.
func opBodies(ops []any) []any {
	var out []any
	for _, op := range ops {
		if o, ok := op.(*pyjson.Object); ok && get(o, "nm") == "a" {
			out = append(out, get(o, "b"))
		}
	}
	return out
}

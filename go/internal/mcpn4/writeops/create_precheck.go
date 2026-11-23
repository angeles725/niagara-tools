package writeops

import (
	"errors"
	"fmt"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// typesKnown refuses, before a token, a type spec (the component's, its proxy_ext's) that the
// station's type registry does not describe, so an unknown type names itself instead of the
// station's generic "Unable to process request". A client without the registry, or a registry
// call that fails (an older station may lack the channel), leaves the check to the station. The
// contracts it read are returned for further checks (nil without a registry).
func typesKnown(c plan.Client, specs ...string) (*plan.Contracts, error) {
	tl, ok := c.(plan.TypeLoader)
	if !ok {
		return nil, nil
	}
	contracts := plan.NewContracts(tl)
	for _, spec := range specs {
		if spec == "" {
			continue
		}
		var unknown *plan.UnansweredType
		errored, err := contracts.Errored(spec)
		if errors.As(err, &unknown) || errored {
			return nil, plan.ToolError(fmt.Sprintf(msgUnknownType, spec) + unknownTypeHint(spec))
		}
	}
	return contracts, nil
}

// unknownTypeHint is the extra sentence for an unknown type spec that names an alarm extension
// ("alarm:OutOfRangeAlarmExt"): the palette has alarm:AlarmSourceExt only. "" for any other spec,
// and for alarm:AlarmSourceExt itself (an unknown real type means a stale registry).
func unknownTypeHint(spec string) string {
	module, name, ok := strings.Cut(spec, ":")
	if ok && module == "alarm" && name != "AlarmSourceExt" && strings.HasSuffix(name, "Ext") {
		return msgAlarmExtHint
	}
	return ""
}

// childNames is the set of the names of a loaded component's children.
func childNames(nodes *pyjson.Object) map[string]bool {
	out := map[string]bool{}
	for _, name := range box.Children(nodes, "").Keys() {
		out[name] = true
	}
	return out
}

// createUnapplied is the read-back after the station refused the add: true when the parent has
// no child it did not have when planned.
func createUnapplied(c plan.Client, args *pyjson.Object, p *plan.Planned) (bool, error) {
	_, nodes, err := plan.Handle(c, str(args, "parent_ord"), 1)
	if err != nil {
		return false, err
	}
	before := p.Data.(createData).children
	for name := range childNames(nodes) {
		if !before[name] {
			return false, nil
		}
	}
	return true, nil
}

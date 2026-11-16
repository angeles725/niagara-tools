package writeops

import (
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// computedFlags are the declared flags that make an output runtime state: r readonly (the
// component computes it, a NumericWritable's out) or t transient (never persisted).
const computedFlags = "rt"

// keepsOutput says whether child, a runtimeOutputSlots Status child of node, is configuration:
// node's type contract declares it without r or t (a constant block's out, such as
// kitControl:NumericConst's, flags 8 summary only). Without contracts, a type the contracts do
// not answer or a slot they do not declare, the output stays left out as before.
func (s *snapshot) keepsOutput(node, child *pyjson.Object) bool {
	if s.ct == nil {
		return false
	}
	typ, _ := get(node, "t").(string)
	slot, err := s.ct.Declared(typ, child)
	if err != nil || slot == nil {
		return false
	}
	flags, _ := get(slot, "f").(string)
	return !strings.ContainsAny(flags, computedFlags)
}

// snapshotContracts are the contracts of every type in root, read as a copy reads them, for a
// snapshot that does not refuse: nil (outputs left out as before) when the client cannot read
// the registry channel or the station does not describe one of the types; then note is the plan
// note that says a constant's out is not carried ("" for a client with no registry channel).
func snapshotContracts(c plan.Client, root *pyjson.Object) (ct *plan.Contracts, note string) {
	tl, ok := c.(plan.TypeLoader)
	if !ok {
		return nil, ""
	}
	ct, err := treeContracts(tl, root)
	if err != nil { // an unanswered type or a failed registry call: the snapshot as before contracts
		return nil, msgOutNotCarried + plan.PyStr(err)
	}
	return ct, ""
}

// msgOutNotCarried starts the note of a snapshot taken without type contracts.
const msgOutNotCarried = "The type contracts could not be read, so a constant block's configured out is not " +
	"in this snapshot and a rollback re-creates it at its default: "

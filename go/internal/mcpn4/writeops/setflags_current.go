package writeops

import (
	"errors"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// currentFlags is the flags slot holds on the component loaded as nodes (its direct slots), as
// the station encodes them. The station writes "f" on a slot only when its flags differ from the
// declared ones, or on a dynamic slot with flags (box-rt ValueDocEncoder.encodeSlot); so:
//   - "f" on the loaded slot is the answer, an empty string included;
//   - else the flags the type contract declares for a frozen slot, listed or omitted;
//   - else none for a listed slot no contract declares (a dynamic slot);
//   - a slot neither listed nor declared does not exist.
//
// Without "f" and without a readable contract the flags are unknown: a refusal, never a guess.
func currentFlags(c plan.Client, nodes *pyjson.Object, ord, slot string) (string, error) {
	node := box.Node(nodes, slot)
	if node != nil && has(node, "f") {
		s, ok := get(node, "f").(string)
		if !ok {
			return "", plan.ToolError(flagsMalformedMsg(slot, ord))
		}
		return s, nil
	}
	typ, _ := get(box.Node(nodes, ""), "t").(string)
	declared, why, err := declaredFlagsSlot(c, typ, slot)
	switch {
	case err != nil:
		return "", err
	case why != "":
		return "", plan.ToolError(flagsUnknownMsg(slot, ord, why))
	case declared != nil:
		f, _ := get(declared, "f").(string) // no "f" in a contract: declared without flags
		return f, nil
	case node != nil:
		return "", nil
	}
	return "", plan.ToolError(flagsNoSlotMsg(slot, ord, typ))
}

// declaredFlagsSlot is the contract slot named slot of type typ (nil: none declares it); why says
// why the contract cannot be read, when it cannot.
func declaredFlagsSlot(c plan.Client, typ, slot string) (declared *pyjson.Object, why string, err error) {
	tl, ok := c.(plan.TypeLoader)
	if !ok {
		return nil, flagsNoRegistry, nil
	}
	if typ == "" {
		return nil, flagsNoType, nil
	}
	declared, err = plan.NewContracts(tl).AnySlot(typ, slot)
	var unanswered *plan.UnansweredType
	if errors.As(err, &unanswered) {
		return nil, unanswered.Error(), nil
	}
	return declared, "", err
}

// undescribedEnd is, for a listed slot that carries no flags and that no contract declares (taken
// for a dynamic slot with no flags by currentFlags), the type the contract walk ended at when the
// station answered it with no contract; "" when the slot carries its flags, the walk ended at
// described types or the registry cannot be read. Retro 2026-11-14 row 10; a baja type answered with {} is not flagged (see Contracts.UndescribedEnd).
func undescribedEnd(c plan.Client, nodes *pyjson.Object, slot string) string {
	node := box.Node(nodes, slot)
	tl, ok := c.(plan.TypeLoader)
	if node == nil || has(node, "f") || !ok {
		return ""
	}
	typ, _ := get(box.Node(nodes, ""), "t").(string)
	if declared, _, err := declaredFlagsSlot(c, typ, slot); err != nil || declared != nil || typ == "" {
		return ""
	}
	end, err := plan.NewContracts(tl).UndescribedEnd(typ)
	if err != nil {
		return ""
	}
	return end
}

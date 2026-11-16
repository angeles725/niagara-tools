package writeops

import (
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// The rollback of an n4_set_flags batch: its journaled inverse is a set-flags op ({nm: f, h, n,
// f}), sent as it is; this file holds its compare-before-restore check and its read-back.

// flagsOpParts are the slot name and the flags of a journaled set-flags op; ok is false when
// either is not a string (a journal line this server did not write).
func flagsOpParts(op *pyjson.Object) (slot, flags string, ok bool) {
	slot, okN := get(op, "n").(string)
	flags, okF := get(op, "f").(string)
	return slot, flags, okN && okF
}

// writtenFlags is the flags the batch set on slot of the component with handle h, from its ops.
func writtenFlags(view *pyjson.Object, h any, slot string) (string, bool) {
	ops, _ := get(view, "ops").([]any)
	for _, e := range ops {
		op, ok := e.(*pyjson.Object)
		if !ok || get(op, "nm") != "f" || !box.PyEqual(get(op, "h"), h) {
			continue
		}
		if n, flags, ok := flagsOpParts(op); ok && n == slot {
			return flags, true
		}
	}
	return "", false
}

// flagsUnchangedSince is unchangedSince for flags: refuse to restore the flags of a slot that no
// longer holds the set the batch gave it; a later change is the operator's to undo, by hand.
func flagsUnchangedSince(c plan.Client, view *pyjson.Object, ops []any, targets *pyjson.Object) error {
	for _, e := range ops {
		op := e.(*pyjson.Object)
		if get(op, "nm") != "f" {
			continue
		}
		slot, _, ok := flagsOpParts(op)
		if !ok {
			return plan.ToolError("batch records a malformed set-flags inverse: refusing")
		}
		wrote, ok := writtenFlags(view, get(op, "h"), slot)
		if !ok {
			continue
		}
		where := ordOf(targets, get(op, "h"))
		nodes, err := c.LoadTree(where, flagsDepth, box.LoadOptions{})
		if err != nil {
			return err
		}
		now, err := currentFlags(c, nodes, where, slot)
		if err != nil {
			return err
		}
		if !sameFlags(now, wrote) {
			return plan.ToolError(flagsChangedSinceMsg(slot, where, get(view, "batch_id"), wrote, now))
		}
	}
	return nil
}

// flagsRestored is the read-back of a replayed set-flags op: whether the slot of the component
// at where (loaded as nodes) now holds the flags the op restored.
func flagsRestored(c plan.Client, where string, nodes, op *pyjson.Object) (bool, error) {
	slot, flags, ok := flagsOpParts(op)
	if !ok {
		return false, nil
	}
	now, err := currentFlags(c, nodes, where, slot)
	return err == nil && sameFlags(now, flags), err
}

package writeops

// The frozen-config run of a rollback: after the parent's add, the planned "frozen_config" of a
// frozen component is written by set ops on the parent's handle (the swap of a component slot
// gives it a new handle; the parent's stays). Only what the confirmed plan holds is sent, and
// only the slots the fresh component does not already hold.

import (
	"errors"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/session"
)

// restoreFrozen writes spec's planned frozen_config under the component at parent (the frozen
// one is parent/n, at path). A set the station refuses is kept in d.frozenRefused for the
// read-back; any other failure is returned (the caller makes the batch in-doubt).
func restoreFrozen(b *plan.Batch, d *rollbackData, spec *pyjson.Object, parent, n, path string) error {
	cfg := asObj(get(spec, "frozen_config"))
	if cfg == nil {
		return nil
	}
	if err := b.State.Scope.Check(path); err != nil {
		return err
	}
	parentH, nodes, err := plan.Handle(b.Client, parent, FrozenCheckDepth)
	if err != nil {
		return err
	}
	live := box.Node(nodes, n)
	if live == nil { // gone: the read-back reports it missing
		return nil
	}
	body := asObj(get(spec, "b"))
	swap := get(cfg, "swap") == true // a frozen child at its declared type loads without one
	if t := get(live, "t"); t != nil {
		swap = !box.PyEqual(t, get(cfg, "type"))
	}
	differ := map[string]bool{}
	if !swap {
		names, err := differs(nodes, n, body)
		if err != nil {
			return err
		}
		for _, s := range names {
			differ[s.(string)] = true
		}
	}
	var ops []any
	if swap {
		ops = append(ops, obj("nm", "s", "h", parentH, "n", n, "b", obj("nm", "p", "t", get(cfg, "type"), "v", "")))
	}
	sets, _ := get(cfg, "sets").([]any)
	for _, s := range sets {
		set := s.(*pyjson.Object)
		if slot := get(set, "n").(string); swap || differ[slot] {
			ops = append(ops, obj("nm", "s", "h", parentH, "n", n+"/"+slot, "b", get(set, "b")))
		}
	}
	if len(ops) == 0 {
		return nil
	}
	if b.State.Journal.Append(obj("batch_id", b.ID, "ts", session.Now(), "phase", "frozen-config-intent",
		"ops", ops)) != nil {
		return plan.InDoubtBecause(b.ID, "the frozen-config intent could not be written, nothing more was sent")
	}
	var refused []any
	for i, op := range ops {
		_, err := b.Client.Sync(op)
		var be *box.Error
		if errors.As(err, &be) {
			refused = append(refused, obj("slot", slotOf(op.(*pyjson.Object), n), "error", be.Msg))
			if swap && i == 0 { // the slots below belong to the type that was not swapped in
				break
			}
			continue
		}
		if err != nil {
			return err
		}
	}
	if len(refused) > 0 {
		if d.frozenRefused == nil {
			d.frozenRefused = map[string][]any{}
		}
		d.frozenRefused[path] = refused
	}
	return nil
}

// slotOf is the slot a frozen-config op writes, relative to the frozen component ("type" for
// the swap of its type).
func slotOf(op *pyjson.Object, n string) string {
	name := pyjson.Str(get(op, "n"))
	if name == n {
		return "type"
	}
	return name[len(n)+1:]
}

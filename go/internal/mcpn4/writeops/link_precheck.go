package writeops

import (
	"fmt"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// linkPrecheck refuses, before a token, a link the station would refuse: a slot missing on its
// component, or a link identical to one the target already has. "The parent accepts it" cannot
// be checked offline; the station's refusal of it is journaled as refused.
func linkPrecheck(c plan.Client, args, srcNodes, tgtNodes *pyjson.Object, d linkData, ss, ts string) error {
	if err := slotExists(c, srcNodes, "source", str(args, "source_ord"), ss); err != nil {
		return err
	}
	if err := slotExists(c, tgtNodes, "target", str(args, "target_ord"), ts); err != nil {
		return err
	}
	return noIdenticalLink(c, args, d, ss, ts)
}

// slotExists refuses a slot that is neither a child of the loaded component nor declared (a
// property, action or topic) by its type or a supertype. It refuses only on a contract the
// station described in full; an undescribed type or a failed registry read leaves it to the
// station.
func slotExists(c plan.Client, nodes *pyjson.Object, end, ord, slot string) error {
	if box.Node(nodes, slot) != nil {
		return nil
	}
	tl, ok := c.(plan.TypeLoader)
	typ, _ := get(box.Node(nodes, ""), "t").(string)
	if !ok || typ == "" {
		return nil
	}
	contracts := plan.NewContracts(tl)
	if described, err := contracts.Described(typ); err != nil || !described {
		return nil
	}
	if declared, err := contracts.AnySlot(typ, slot); err != nil || declared != nil {
		return nil
	}
	return plan.ToolError(fmt.Sprintf(msgSlotMissing, end, pyjson.StrRepr(slot), ord, typ))
}

// noIdenticalLink refuses a link whose source handle and slots match a link the target already
// has (read with its slot values, depth 2). A read that fails leaves it to the station.
func noIdenticalLink(c plan.Client, args *pyjson.Object, d linkData, ss, ts string) error {
	if len(d.existing) == 0 {
		return nil
	}
	tgtOrd := str(args, "target_ord")
	_, nodes, err := plan.Handle(c, tgtOrd, 2)
	want, werr := handleOrd(d.srcH)
	if err != nil || werr != nil {
		return nil
	}
	for _, name := range links(nodes) {
		f := linkSlots(nodes, name)
		if tail, err := sourceTail(f); err == nil && tail == want && box.PyEqual(get(f, "sourceSlotName"), ss) &&
			box.PyEqual(get(f, "targetSlotName"), ts) {
			return plan.ToolError(fmt.Sprintf(msgIdenticalLink, tgtOrd, name, pyjson.StrRepr(ss), pyjson.StrRepr(ts)))
		}
	}
	return nil
}

// linkUnapplied is the read-back after the station refused checkLinks: true when the target
// has no link it did not have when planned.
func linkUnapplied(c plan.Client, args *pyjson.Object, p *plan.Planned) (bool, error) {
	_, nodes, err := plan.Handle(c, str(args, "target_ord"), 1)
	if err != nil {
		return false, err
	}
	existing := p.Data.(linkData).existing
	for _, name := range links(nodes) {
		if !existing[name] {
			return false, nil
		}
	}
	return true, nil
}

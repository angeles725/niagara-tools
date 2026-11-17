package writeops

import (
	"strconv"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// A slot that is the target of a link is written again by that link at its next update, so a
// value n4_set_slot writes there does not stay (backlog OD001, retro hist-alarms M9). The dry
// run lists those links; the write itself is not refused (writing a fallback or a linked input
// on purpose is legitimate), but the plan says plainly that the value will be overwritten.

// incomingLinks are the links on the component (its direct link children in the load) whose
// target slot is the top slot of slot: {source_ord, source_slot, target_slot, enabled}.
func incomingLinks(nodes *pyjson.Object, slot string) []any {
	top, _, _ := strings.Cut(slot, "/")
	out := []any{}
	for _, k := range box.Children(nodes, "").Keys() {
		if !isLink(get(box.Node(nodes, k), "t")) {
			continue
		}
		kids := box.Children(nodes, k)
		if box.ChildValue(kids, "targetSlotName", nil) != top {
			continue
		}
		out = append(out, obj("source_ord", box.ChildValue(kids, "sourceOrd", nil),
			"source_slot", box.ChildValue(kids, "sourceSlotName", nil), "target_slot", top,
			"enabled", box.ChildValue(kids, "enabled", nil) != "false"))
	}
	return out
}

// incomingNote is the plan note of links into the written slot ("" when there are none).
func incomingNote(links []any, slot string) string {
	if len(links) == 0 {
		return ""
	}
	return strconv.Itoa(len(links)) + " link(s) write " + pyjson.StrRepr(slot) + " (incoming_links): " +
		"the value written here is overwritten at the link's next update; to keep it, remove the link " +
		"(n4_list_links shows it) or write the link's source instead"
}

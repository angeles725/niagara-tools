package readtools

import (
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// An alarm or history extension reads its parent point's out without a link (the extension
// executes against its parent), so a point whose only consumer is such an extension is not
// dangling (backlog OD007, retro hist-alarms M8: 13 false positives on client A).

// extConsumer is the type of a child of p that consumes p's out: an alarm source extension
// (alarm:...AlarmExt) or a history extension (history:...HistoryExt); "" when none is loaded.
func extConsumer(nodes *pyjson.Object, p string) string {
	kids := box.Children(nodes, p) // keyed by name, not path
	for _, k := range kids.Keys() {
		s, _ := lookup(box.Node(kids, k), "t").(string)
		if (strings.HasPrefix(s, "alarm:") && strings.HasSuffix(s, "AlarmExt")) ||
			(strings.HasPrefix(s, "history:") && strings.HasSuffix(s, "HistoryExt")) {
			return s
		}
	}
	return ""
}

// Reasons an out nothing links is not dangling (retro 2026-11-15 row 14: terminal points and
// calendar schedules were listed as dangling).
const (
	reasonProxyTerminal = "its out drives a device through its proxy extension"
	reasonCalendar      = "a calendar schedule is read by other schedules by ORD, never by a link"
)

// terminalReason says why the out of p (of type kind) has no link by design, "" when it has no
// such reason: a writable point with a proxy extension child (not the default control:NullProxyExt every control point holds) (its out is written to a device), or
// a schedule:CalendarSchedule (other schedules refer to it by ORD).
func terminalReason(nodes *pyjson.Object, p string, kind any) string {
	t, _ := kind.(string)
	if t == "schedule:CalendarSchedule" {
		return reasonCalendar
	}
	if !strings.HasPrefix(t, "control:") || !strings.HasSuffix(t, "Writable") {
		return ""
	}
	kids := box.Children(nodes, p)
	for _, k := range kids.Keys() {
		if s, _ := lookup(box.Node(kids, k), "t").(string); strings.HasSuffix(s, "ProxyExt") && s != nullProxyExt {
			return reasonProxyTerminal
		}
	}
	return ""
}

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

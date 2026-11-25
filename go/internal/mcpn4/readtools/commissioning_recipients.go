package readtools

import (
	"fmt"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// classRecipients lists in c the classes no link reads the alarm topic of, among the links the
// scan of the AlarmService sees. While a link that may read an alarm topic has a source the
// scan cannot name (unresolved, or a handle outside the loaded levels), no class is reported:
// the missing link may be that one.
func (r *commRun) classRecipients(c *commCheck, classes []*pyjson.Object) error {
	r.requests++
	_, links, err := r.t.scanOrd(alarmServiceOrd, alarmScanDepth)
	if err != nil {
		c.why, err = r.failure("the link scan of "+alarmServiceOrd+" failed", err)
		return err
	}
	linked, unseen := map[string]bool{}, 0
	for _, l := range links {
		if !routesAlarm(l.sourceSlot, l.targetSlot, true) {
			continue
		}
		if p, ok := l.sourcePath.(string); ok && l.resolved {
			linked[p] = true
		} else {
			unseen++
		}
	}
	if unseen > 0 {
		c.why = fmt.Sprintf(whyUnresolved, unseen)
		return nil
	}
	for _, row := range classes {
		path := cellText(row, "Slot Path")
		if !linked[strings.TrimPrefix(path, slotPrefix)] {
			c.add(path, cellText(row, "Name"), fmt.Sprintf(factNoRecipient, alarmScanDepth+1))
		}
	}
	return nil
}

package readtools

import (
	"fmt"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// The class side of n4_list_alarm_classes audit=true (station-audits S5): the classes no
// recipient's routeAlarm link reads, and the counts of the audit. While a routeAlarm link has a
// source no listed class answers, or recipients were left unread, a class listed as unrouted may
// be routed by that link: the note says so (the commissioning report withholds the list then;
// here the operator asked for the audit, so the list is given with its caveat).

// Notes of the routing audit, as data.
const (
	noteUnrouted = "%d routeAlarm link(s) have a source that is not a listed class and %d recipient(s) were not read: " +
		"a class listed in unrouted_classes may be routed by one of them"
)

// addRoutingAudit adds unrouted_classes, audit_counts and, when the scan is incomplete,
// unrouted_note to the reply out.
func addRoutingAudit(out *pyjson.Object, classes []*pyjson.Object, routes *alarmRouting) {
	unrouted := []any{}
	for _, c := range classes {
		if r, _ := get(c, "recipients").([]any); len(r) == 0 {
			unrouted = append(unrouted, obj("path", get(c, "path"), "name", get(c, "name")))
		}
	}
	withProblems := 0
	for _, r := range routes.recipients {
		if p, _ := get(r.(*pyjson.Object), "problems").([]any); len(p) > 0 {
			withProblems++
		}
	}
	out.Set("unrouted_classes", unrouted)
	out.Set("audit_counts", obj("classes", len(classes), "unrouted_classes", len(unrouted),
		"recipients_read", len(routes.recipients), "recipients_with_problems", withProblems))
	if unread := routes.total - len(routes.recipients); len(routes.unresolved) > 0 || unread > 0 {
		out.Set("unrouted_note", fmt.Sprintf(noteUnrouted, len(routes.unresolved), unread))
	}
}

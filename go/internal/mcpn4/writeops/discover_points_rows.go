package writeops

import (
	"strconv"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// pointsUnread is the objectName a row keeps when the station could not read the object (a
// device added before its I-Am was heard gives such partial jobs that still end success).
const pointsUnread = "--"

// pointsReadback waits for the point discovery job, then lists its rows: each object marked with
// the point already proxying it under the extension (in_database) or null, and unread rows counted.
func pointsReadback(c plan.Client, _ *pyjson.Object, p *plan.Planned, replies, _ []any) (any, any, any, string, error) {
	d := p.Data.(pointsData)
	requested := obj("action", pointsAction, "points_ord", d.ext)
	job := jobOrd(replies[0])
	if job == "" {
		return requested, replies[0], obj("hint", "the station returned no job ORD"), "unverified", nil
	}
	state, nodes, err := waitJob(c, job, discoverData{wait: d.wait, w: d.w})
	if err != nil {
		return nil, nil, nil, "", err
	}
	observed := obj("job", job, "job_state", state)
	switch {
	case state == "success":
	case jobComplete(state):
		return requested, replies[0], observed, "mismatch", nil
	default:
		observed.Set("hint", "the point discovery job is still "+pyjson.Str(state)+" after "+strconv.Itoa(d.wait)+
			" s plus "+strconv.Itoa(int(discoverSlack))+" s; read it later under "+job)
		return requested, replies[0], observed, "unverified", nil
	}
	known, err := proxiedObjects(c, d.ext)
	if err != nil {
		return nil, nil, nil, "", err
	}
	rows, unread, differ := []any{}, 0, 0
	var hardware any
	for _, name := range box.Children(nodes, "").Keys() {
		if !strings.Contains(pyjson.Str(get(box.Node(nodes, name), "t")), "DiscoveryPoint") {
			continue
		}
		val := func(slot string) any { return box.ChildValue(box.Children(nodes, name), slot, nil) }
		id := strOr(val("objectId"))
		row := obj("object_name", strOr(val("objectName")), "object_id", id, "value", val("value"),
			"description", cleanDescription(strOr(val("description"))), "in_database", nil)
		if hw := spyderHardware(strOr(val("objectName")), id, val("value")); hw != nil {
			hardware = hw
		}
		if path, ok := known[id]; ok {
			// #820: the objectName is only on the device; name_differs compares it with the point's name
			point := unescapeSlot(path[strings.LastIndex(path, "/")+1:])
			row.Set("in_database", path)
			row.Set("name_differs", point != strOr(val("objectName")))
			if point != strOr(val("objectName")) {
				differ++
			}
		}
		if strOr(val("objectName")) == pointsUnread {
			row.Set("unread", true)
			unread++
		}
		rows = append(rows, row)
	}
	observed.Set("found", len(rows))
	observed.Set("unread", unread)
	observed.Set("names_differ", differ)
	if unread > 0 {
		observed.Set("hint", "unread rows: the device may not have sent an I-Am yet; run n4_discover_devices, then discover the points again")
	}
	observed.Set("hardware", hardware)
	observed.Set("points", rows)
	return requested, replies[0], observed, "verified", nil
}

// proxiedObjects maps each objectId already proxied under the extension (folders included) to the
// point's path relative to the extension.
func proxiedObjects(c plan.Client, ext string) (map[string]string, error) {
	nodes, err := c.LoadTree(ext, 6, box.LoadOptions{})
	if err != nil {
		return nil, err
	}
	const suffix = "/proxyExt/objectId"
	out := map[string]string{}
	for _, k := range nodes.Keys() {
		if strings.HasSuffix(k, suffix) {
			out[pyjson.Str(get(box.Node(nodes, k), "v"))] = strings.TrimSuffix(k, suffix)
		}
	}
	return out, nil
}

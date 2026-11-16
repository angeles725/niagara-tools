package blackbox

import (
	"reflect"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest/boxstation"
)

// The queries of n4_commissioning_report by the text that tells them apart (the network needle
// first: the device needle is a prefix of it), each with the header ITableToCsv writes for it.
var comQueries = []struct{ table, needle, header string }{
	{"networks", "driver:DeviceNetwork", "Slot Path,Name,Type,Bits,Enabled"},
	{"devices", "driver:Device", "Slot Path,Name,Type,Bits,Enabled"},
	{"points", "control:ControlPoint", "Slot Path,Name,Display Name,Type,Bits"},
	{"histories", "history:HistoryExt", "Slot Path,Name,Type,Enabled,Bits,Id,Timestamp"},
	{"alarms", "alarm:AlarmSourceExt", "Slot Path,Name,Type,Alarm Class"},
	{"classes", "alarm:AlarmClass", "Slot Path,Name,Open Alarm Count,Unacked Alarm Count"},
}

// comFailed is a table whose query the station refuses (no route: HTTP 404).
const comFailed = "\x00failed"

// comRoutes answers each query with its header and the rows given for its table (none: an
// empty table); a table set to comFailed is not answered.
func comRoutes(sv *boxstation.Servlets, tables map[string][]string) {
	routes := []boxstation.OrdRoute{}
	for _, q := range comQueries {
		rows := tables[q.table]
		if len(rows) == 1 && rows[0] == comFailed {
			continue
		}
		body := q.header + "\n"
		if len(rows) > 0 {
			body += strings.Join(rows, "\n") + "\n"
		}
		routes = append(routes, boxstation.OrdRoute{Needle: q.needle, Body: []byte(body)})
	}
	routeOrds(sv, routes...)
}

// comRig is a connected servlet rig whose station answers the report's queries from tables.
func comRig(t *testing.T, tables map[string][]string) (*Rig, *boxstation.Servlets) {
	t.Helper()
	r, sv := servletRig(t, nil)
	comRoutes(sv, tables)
	return r, sv
}

// comSection is one section of a report over args.
func comSection(t *testing.T, r *Rig, name string, args Args) map[string]any {
	t.Helper()
	if args == nil {
		args = Args{}
	}
	if args["sections"] == nil {
		args["sections"] = []any{name}
	}
	return shxSection(t, r.Srv.OK("n4_commissioning_report", args), name)
}

// comCheck is one check of a section.
func comCheck(t *testing.T, section map[string]any, name string) map[string]any {
	t.Helper()
	checks, _ := section["checks"].(map[string]any)
	c, ok := checks[name].(map[string]any)
	if !ok {
		t.Fatalf("check %s = %v (checks %v)", name, checks[name], checks)
	}
	return c
}

// comWant fails unless check ran with count findings and lists exactly the paths given.
func comWant(t *testing.T, section map[string]any, name string, count float64, paths ...string) map[string]any {
	t.Helper()
	c := comCheck(t, section, name)
	got := []string{}
	for _, p := range field(c["items"], "path") {
		s, _ := p.(string)
		got = append(got, s)
	}
	if c["count"] != count || !reflect.DeepEqual(got, append([]string{}, paths...)) {
		t.Errorf("%s = %v, want count %v and paths %q", name, c, count, paths)
	}
	return c
}

// comFact is the fact of the item of check at path.
func comFact(c map[string]any, path string) string {
	for _, item := range objects(c["items"]) {
		if item["path"] == path {
			s, _ := item["fact"].(string)
			return s
		}
	}
	return ""
}

// comSummary is the summary line of section in out ("" when absent).
func comSummary(out map[string]any, section string) string {
	for _, line := range out["summary"].([]any) {
		if s, _ := line.(string); strings.HasPrefix(s, section+": ") {
			return s
		}
	}
	return ""
}

// comServices gives the station a Services container and returns it (lock held by the caller).
func comServices(st *boxstation.Station) *boxstation.Node {
	if n := st.Root().Child("Services"); n != nil {
		return n
	}
	return st.Component(st.Root(), "Services", "baja:ServiceContainer", "")
}

// comAlarmService gives the station an AlarmService with the classes named (each an
// alarm:AlarmClass) and one recipient; it returns their handles by name.
func comAlarmService(r *Rig, classes ...string) map[string]string {
	handles := map[string]string{}
	r.Station.Do(func(st *boxstation.Station) {
		svc := st.Component(comServices(st), "AlarmService", "alarm:AlarmService", "")
		for _, name := range classes {
			handles[name] = st.Component(svc, name, "alarm:AlarmClass", "").Handle
		}
		handles["Console"] = st.Component(svc, "Console", "alarm:ConsoleRecipient", "").Handle
	})
	return handles
}

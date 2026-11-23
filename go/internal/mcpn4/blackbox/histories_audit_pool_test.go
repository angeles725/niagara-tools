package blackbox

// n4_histories_audit reads each history's record count (and, with flat=true, its window) with a
// small fixed pool of requests at once (retro R8 delta N5: 223 requests one by one took 140 s);
// the reply is what the one-by-one reads gave.

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest/boxstation"
)

const poolHistories = 10

// poolRig is a rig of poolHistories recording histories H0..H9 (capped by 1000 records) whose
// station holds each answer back 40 ms; counts are the records per history, a history with no
// entry has no route (its read fails).
func poolRig(t *testing.T, counts map[int]string, flat bool) (*Rig, *boxstation.Servlets) {
	t.Helper()
	var points, exts []string
	for i := 0; i < poolHistories; i++ {
		p := fmt.Sprintf("slot:/Pool/P%d", i)
		points = append(points, p)
		exts = append(exts, strings.Join([]string{p + "/hist", "history:NumericIntervalHistoryExt", "true",
			fmt.Sprintf("/FakeStation/H%d", i), "1:1000", "roll", "false:300000", haAgo(time.Minute)}, ","))
	}
	r, sv := servletRig(t, nil)
	routes := []boxstation.OrdRoute{
		{Needle: "control:ControlPoint", Body: []byte("Slot Path\n" + strings.Join(points, "\n") + "\n")},
		{Needle: "history:HistoryExt", Body: []byte("Slot Path,Type,Enabled,Id,Encode To String,Encode To String," +
			"Encode To String,Encode To String\n" + strings.Join(exts, "\n") + "\n")}}
	delay := 40 * time.Millisecond
	for i := 0; i < poolHistories; i++ {
		id := fmt.Sprintf("history:/FakeStation/H%d", i)
		if flat {
			routes = append(routes,
				boxstation.OrdRoute{Needle: id + "?start=", Delay: delay, Body: []byte("min(value),max(value)\n0.0,0.0\n")},
				boxstation.OrdRoute{Needle: id + "?end=", Delay: delay, Body: []byte("count(toString)\n" + counts[i] + "\n")})
			continue
		}
		if n, ok := counts[i]; ok {
			routes = append(routes, boxstation.OrdRoute{Needle: id + "|bql:select count(*)", Delay: delay,
				Body: []byte("count(toString)\n" + n + "\n")})
		}
	}
	routeOrds(sv, routes...)
	return r, sv
}

func maxInFlight(sv *boxstation.Servlets) (n int) {
	sv.Do(func(sv *boxstation.Servlets) { n = sv.MaxOrdInFlight })
	return n
}

func TestHistoriesAuditReadsRecordCountsFourAtATimeAndKeepsTheReply(t *testing.T) {
	counts := map[int]string{}
	for i := 0; i < poolHistories; i++ {
		counts[i] = "10"
	}
	counts[3], counts[7] = "950", "990"
	delete(counts, 5) // no route: that count fails
	delete(counts, 8)
	r, sv := poolRig(t, counts, false)
	out := r.Srv.OK("n4_histories_audit", Args{})
	if got := haPaths(out["near_capacity"], "history_id"); !reflect.DeepEqual(got, []string{"/FakeStation/H3", "/FakeStation/H7"}) {
		t.Errorf("near_capacity = %q, want the ext order H3, H7", got)
	}
	if out["count_reads"] != 10.0 || out["requests"] != 12.0 || len(ordRequests(sv)) != 12 {
		t.Errorf("count_reads %v requests %v station requests %d", out["count_reads"], out["requests"], len(ordRequests(sv)))
	}
	notes, _ := out["notes"].([]any)
	if len(notes) != 1 || !strings.HasPrefix(notes[0].(string), "2 record count read(s) failed, those histories are not in "+
		"near_capacity; first: /FakeStation/H5: ") {
		t.Errorf("notes = %q, want the first failure in extension order (H5, not H8)", notes)
	}
	if n := maxInFlight(sv); n < 2 || n > 4 {
		t.Errorf("most count reads at once = %d, want 2 to 4", n)
	}
}

func TestHistoriesAuditFlatReadsFourHistoriesAtATimeAndKeepsTheReply(t *testing.T) {
	counts := map[int]string{}
	for i := 0; i < poolHistories; i++ {
		counts[i] = "600"
	}
	counts[2] = "0" // younger than the window: not flat
	r, sv := poolRig(t, counts, true)
	out := r.Srv.OK("n4_histories_audit", Args{"flat": true})
	var ids []string
	for _, f := range objects(out["flat"]) {
		ids = append(ids, f["history_id"].(string))
	}
	want := []string{}
	for i := 0; i < poolHistories; i++ {
		if i != 2 {
			want = append(want, fmt.Sprintf("/FakeStation/H%d", i))
		}
	}
	if !reflect.DeepEqual(ids, want) {
		t.Errorf("flat = %q, want the ext order %q", ids, want)
	}
	// each history: one window read and one count before it; the plain counts answer 404
	if out["flat_reads"] != 20.0 {
		t.Errorf("flat_reads = %v, want 20", out["flat_reads"])
	}
	if n := maxInFlight(sv); n < 2 || n > 4 {
		t.Errorf("most flat reads at once = %d, want 2 to 4", n)
	}
}

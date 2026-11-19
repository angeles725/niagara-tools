package blackbox

// n4_histories_audit flat=true (mcp-n4 delta M3): a history holding one value for more than
// flat_days days (default 2) is listed as a possible device placeholder, with its point and value.

import (
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest/boxstation"
)

const flatDev = "slot:/Unit/"

// flatRig is a connected rig whose station holds four recording histories: Flat (one value over
// the window, older records before it), Moving (its value changes), Young (one value, but no
// record before the window) and Empty (never recorded).
func flatRig(t *testing.T) (*Rig, *boxstation.Servlets) {
	t.Helper()
	ext := func(name, last string) string {
		return flatDev + name + "/hist,history:NumericIntervalHistoryExt,true,/FakeStation/" + name +
			",0:0,roll,false:300000," + last
	}
	exts := []string{ext("Flat", haAgo(time.Minute)), ext("Moving", haAgo(time.Minute)),
		ext("Young", haAgo(time.Minute)), ext("Empty", "null")}
	points := []string{flatDev + "Flat", flatDev + "Moving", flatDev + "Young", flatDev + "Empty"}
	minMax := func(lo, hi string) []byte { return []byte("min(value),max(value)\n" + lo + "," + hi + "\n") }
	count := func(n string) []byte { return []byte("count(toString)\n" + n + "\n") }
	r, sv := servletRig(t, nil)
	routeOrds(sv,
		boxstation.OrdRoute{Needle: "control:ControlPoint", Body: []byte("Slot Path\n" + strings.Join(points, "\n") + "\n")},
		boxstation.OrdRoute{Needle: "history:HistoryExt", Body: []byte("Slot Path,Type,Enabled,Id,Encode To String," +
			"Encode To String,Encode To String,Encode To String\n" + strings.Join(exts, "\n") + "\n")},
		boxstation.OrdRoute{Needle: "history:/FakeStation/Flat?start=", Body: minMax("0.0", "0.0")},
		boxstation.OrdRoute{Needle: "history:/FakeStation/Flat?end=", Body: count("600")},
		boxstation.OrdRoute{Needle: "history:/FakeStation/Moving?start=", Body: minMax("20.1", "23.4")},
		boxstation.OrdRoute{Needle: "history:/FakeStation/Moving?end=", Body: count("600")},
		boxstation.OrdRoute{Needle: "history:/FakeStation/Young?start=", Body: minMax("0.0", "0.0")},
		boxstation.OrdRoute{Needle: "history:/FakeStation/Young?end=", Body: count("0")})
	return r, sv
}

func TestHistoriesAuditFlagsAHistoryFlatForMoreThanTwoDays(t *testing.T) {
	r, sv := flatRig(t)
	out := r.Srv.OK("n4_histories_audit", Args{"flat": true})
	flat := objects(out["flat"])
	if len(flat) != 1 {
		t.Fatalf("flat = %v, want only /FakeStation/Flat", out["flat"])
	}
	f := flat[0]
	if f["history_id"] != "/FakeStation/Flat" || f["point"] != flatDev+"Flat" || f["value"] != "0.0" ||
		f["ext"] != flatDev+"Flat/hist" || !strings.Contains(f["note"].(string), "possible device placeholder") {
		t.Errorf("flat item = %v", f)
	}
	counts, _ := out["counts"].(map[string]any)
	if counts["flat"] != 1.0 || out["flat_days"] != 2.0 {
		t.Errorf("counts.flat %v flat_days %v", counts["flat"], out["flat_days"])
	}
	// Empty never recorded: no read. Flat, Moving, Young: one window read; Flat and Young (one
	// value) also one count of the records before the window.
	if out["flat_reads"] != 5.0 || out["requests"] != 7.0 || len(ordRequests(sv)) != 7 {
		t.Errorf("flat_reads %v requests %v station requests %q", out["flat_reads"], out["requests"], ordRequests(sv))
	}
	for _, req := range ordRequests(sv) {
		if strings.Contains(req, "Empty") {
			t.Errorf("a never-recorded history was read: %s", req)
		}
	}
}

func TestHistoriesAuditFlatHonoursACustomNumberOfDays(t *testing.T) {
	r, sv := flatRig(t)
	before := time.Now().UTC()
	out := r.Srv.OK("n4_histories_audit", Args{"flat": true, "flat_days": 5})
	if out["flat_days"] != 5.0 || !strings.Contains(objects(out["flat"])[0]["note"].(string), "5d") {
		t.Errorf("flat_days %v flat %v", out["flat_days"], out["flat"])
	}
	var start string
	for _, req := range ordRequests(sv) {
		if i := strings.Index(req, "history:/FakeStation/Flat?start="); i >= 0 {
			start, _, _ = strings.Cut(req[i+len("history:/FakeStation/Flat?start="):], "|")
		}
	}
	start, _ = url.QueryUnescape(start)
	at, err := time.Parse("2006-01-02T15:04:05.000Z", start)
	if want := before.Add(-5 * 24 * time.Hour); err != nil || at.Sub(want).Abs() > time.Minute {
		t.Errorf("window start %q (%v), want about %s", start, err, want.Format(time.RFC3339))
	}
}

func TestHistoriesAuditWithoutFlatAnswersAsBefore(t *testing.T) {
	r, sv := flatRig(t)
	out := r.Srv.OK("n4_histories_audit", Args{})
	for _, k := range []string{"flat", "flat_days", "flat_reads"} {
		if _, has := out[k]; has {
			t.Errorf("%s present without flat=true: %v", k, out[k])
		}
	}
	counts, _ := out["counts"].(map[string]any)
	want := map[string]any{"points": 4.0, "history_exts": 4.0, "no_history": 0.0, "disabled": 0.0, "stale": 1.0,
		"near_capacity": 0.0, "colliding_ids": 0.0}
	if !reflect.DeepEqual(counts, want) || out["requests"] != 2.0 || len(ordRequests(sv)) != 2 {
		t.Errorf("counts %v requests %v station requests %d", counts, out["requests"], len(ordRequests(sv)))
	}
}

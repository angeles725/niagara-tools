package readtools

// The pure rules of n4_histories_audit (station-audits S1): which points have no history,
// which ids collide, which histories are stale and which are close to their capacity.

import (
	"reflect"
	"testing"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

var haNow = time.Date(2026, 11, 17, 12, 0, 0, 0, time.UTC)

var haCfg = histAuditCfg{staleFactor: 3, covMaxAge: 24 * time.Hour, warn: 0.9, now: haNow}

func haExt(path, typ, id, capacity, policy, interval, last string) histExt {
	return histExt{path: path, typ: typ, id: id, capacity: capacity, policy: policy, interval: interval,
		last: last, enabled: true}
}

func TestPointsWithoutHistoryAreThoseWithNoExtensionRightBelow(t *testing.T) {
	points := []string{"slot:/A/P1", "slot:/A/P2", "slot:/A/P2x"}
	exts := []histExt{{path: "slot:/A/P1/NumericInterval"}, {path: "slot:/A/P2x/sub/Cov"}}
	if got := pointsWithoutHistory(points, exts); !reflect.DeepEqual(got, []string{"slot:/A/P2", "slot:/A/P2x"}) {
		t.Errorf("pointsWithoutHistory = %q", got)
	}
}

func TestCollidingIdsGroupsExtensionsSharingAnId(t *testing.T) {
	exts := []histExt{{path: "slot:/D1/T/h", id: "/S/T"}, {path: "slot:/D2/T/h", id: "/S/T"},
		{path: "slot:/D3/U/h", id: "/S/U"}, {path: "slot:/D4/X/h", id: ""}, {path: "slot:/D5/Y/h", id: ""}}
	got := pyjson.Repr(list(collidingIds(exts)))
	if want := "[{'history_id': '/S/T', 'exts': ['slot:/D1/T/h', 'slot:/D2/T/h']}]"; got != want {
		t.Errorf("collidingIds = %s, want %s", got, want)
	}
}

func TestStaleIntervalHistoryIsOlderThanFactorTimesInterval(t *testing.T) {
	fresh := haExt("slot:/P/h", "history:NumericIntervalHistoryExt", "/S/P", "1:1000", "roll", "false:300000",
		"2026-11-17T05:50:00.000-06:00") // 10 min old, threshold 15 min
	if item, why := haCfg.stale(fresh); item != nil || why != "" {
		t.Errorf("fresh = %v, %q", pyjson.Repr(item), why)
	}
	old := fresh
	old.last = "2026-11-17T05:40:00.000-06:00" // 20 min old
	item, _ := haCfg.stale(old)
	want := "{'history_id': '/S/P', 'ext': 'slot:/P/h', 'last_record': '2026-11-17T05:40:00.000-06:00', " +
		"'interval': '5m', 'threshold': '15m', 'age': '20m'}"
	if pyjson.Repr(item) != want {
		t.Errorf("stale = %s\nwant  %s", pyjson.Repr(item), want)
	}
}

func TestStaleCovHistoryIsOlderThanCovMaxAge(t *testing.T) {
	cov := haExt("slot:/B/h", "history:BooleanCovHistoryExt", "/S/B", "1:500", "roll", "true:0",
		"2026-11-15T12:00:00.000Z")
	item, _ := haCfg.stale(cov)
	want := "{'history_id': '/S/B', 'ext': 'slot:/B/h', 'last_record': '2026-11-15T12:00:00.000Z', " +
		"'interval': 'cov', 'threshold': '1d', 'age': '2d'}"
	if pyjson.Repr(item) != want {
		t.Errorf("stale = %s\nwant  %s", pyjson.Repr(item), want)
	}
	cov.last = "2026-11-17T00:00:00.000Z"
	if item, _ := haCfg.stale(cov); item != nil {
		t.Errorf("12 h old COV = %s", pyjson.Repr(item))
	}
}

func TestStaleNamesNeverRecordedAndUnreadableCells(t *testing.T) {
	never := haExt("slot:/N/h", "history:NumericIntervalHistoryExt", "/S/N", "1:1000", "roll", "false:60000", "null")
	item, _ := haCfg.stale(never)
	if lookup(item, "last_record") != nil || lookup(item, "age") != nil || lookup(item, "note") != "never recorded" {
		t.Errorf("never = %s", pyjson.Repr(item))
	}
	bad := never
	bad.last = "17-Nov-26 6:00 AM CST"
	if item, why := haCfg.stale(bad); item != nil || why != "last_record" {
		t.Errorf("unreadable last = %v, %q", pyjson.Repr(item), why)
	}
	bad = never
	bad.last, bad.interval = "2026-11-17T05:40:00.000-06:00", ""
	if item, why := haCfg.stale(bad); item != nil || why != "interval" {
		t.Errorf("unreadable interval = %v, %q", pyjson.Repr(item), why)
	}
	bad.typ = "history:NumericCovHistoryExt" // the type alone tells a COV extension
	if item, why := haCfg.stale(bad); item != nil || why != "" {
		t.Errorf("COV by type = %v, %q", pyjson.Repr(item), why)
	}
}

func TestNearCapacitySeparatesStopFromRoll(t *testing.T) {
	stop := haExt("slot:/S/h", "history:NumericIntervalHistoryExt", "/S/S", "1:1000", "stop", "false:60000", "")
	want := "{'history_id': '/S/S', 'ext': 'slot:/S/h', 'records': 950, 'capacity': 1000, 'fraction': 0.95, " +
		"'full_policy': 'stop', 'severity': 'warning', 'effect': 'recording stops when the history is full'}"
	if got := pyjson.Repr(haCfg.nearCapacity(stop, 950)); got != want {
		t.Errorf("stop = %s\nwant %s", got, want)
	}
	if got := lookup(haCfg.nearCapacity(stop, 1000), "effect"); got != "full: the history has stopped recording" {
		t.Errorf("full stop effect = %v", got)
	}
	roll := stop
	roll.policy = "roll"
	if item := haCfg.nearCapacity(roll, 1000); lookup(item, "severity") != "info" ||
		lookup(item, "effect") != "rollover: the oldest records are overwritten" {
		t.Errorf("roll = %s", pyjson.Repr(item))
	}
	if item := haCfg.nearCapacity(stop, 899); item != nil {
		t.Errorf("89.9%% = %s", pyjson.Repr(item))
	}
}

func TestNeedsCountOnlyForRecordingRecordCappedHistories(t *testing.T) {
	base := haExt("slot:/C/h", "history:NumericIntervalHistoryExt", "/S/C", "1:1000", "stop", "false:60000",
		"2026-11-17T05:40:00.000-06:00")
	if !needsCount(base) {
		t.Error("record-capped enabled history not counted")
	}
	for name, mutate := range map[string]func(*histExt){
		"disabled":  func(e *histExt) { e.enabled = false },
		"no id":     func(e *histExt) { e.id = "" },
		"unlimited": func(e *histExt) { e.capacity = "0:0" },
		"by bytes":  func(e *histExt) { e.capacity = "2:4096" },
		"never":     func(e *histExt) { e.last = "null" },
	} {
		e := base
		mutate(&e)
		if needsCount(e) {
			t.Errorf("%s history counted", name)
		}
	}
}

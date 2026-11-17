package blackbox

// n4_read_history cov_hint (station-audits S3): a COV changeTolerance suggested from a history's
// real data, report only, from a fake station's answers.

import (
	"reflect"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest/boxstation"
)

// covRig is a connected rig whose station answers the records, the count and the
// machine-encoded summary read of /FakeStation/Noisy (values oldest first) and of
// /FakeStation/Door (a boolean history).
func covRig(t *testing.T) (*Rig, *boxstation.Servlets) {
	t.Helper()
	noisy := []string{"20.0", "20.3", "20.1", "20.2", "20.0", "20.3"}
	var b strings.Builder
	b.WriteString("Encode To String,Encode To String,Status\n")
	for i := len(noisy) - 1; i >= 0; i-- {
		b.WriteString("2026-11-16T02:2" + string(rune('0'+i)) + ":00.000-06:00," + noisy[i] + ",{ok}\n")
	}
	r, sv := servletRig(t, nil)
	routeOrds(sv,
		boxstation.OrdRoute{Needle: "history:/FakeStation/Noisy|bql:select timestamp.encodeToString", Body: []byte(b.String())},
		boxstation.OrdRoute{Needle: "history:/FakeStation/Door|bql:select timestamp.encodeToString",
			Body: []byte("Encode To String,Encode To String,Status\n2026-11-16T02:21:00.000-06:00,true,{ok}\n" +
				"2026-11-16T02:20:00.000-06:00,false,{ok}\n")},
		boxstation.OrdRoute{Needle: "select count(*)", Body: []byte("count(toString)\n6\n")},
		boxstation.OrdRoute{Needle: "order by timestamp desc", Body: []byte("Timestamp,Value,Status\n16 nov 2026 02:25:00 CST,\"20,3\",{ok}\n")})
	return r, sv
}

func TestCovHintSuggestsATolerancefromANoisyHistory(t *testing.T) {
	r, sv := covRig(t)
	out := r.Srv.OK("n4_read_history", Args{"history_id": "/FakeStation/Noisy", "limit": 1, "cov_hint": true})
	hint, _ := out["cov_hint"].(map[string]any)
	if hint == nil {
		t.Fatalf("cov_hint = %v (note %v)", out["cov_hint"], out["cov_hint_note"])
	}
	if hint["suggested_tolerance"] != 0.5 || hint["records_considered"] != 6.0 || hint["reduction_estimate"] != 0.8333333333 {
		t.Errorf("cov_hint = %v", hint)
	}
	deltas, _ := hint["deltas"].(map[string]any)
	if deltas["median"] != 0.2 || deltas["p90"] != 0.3 || deltas["p99"] != 0.3 || deltas["noise_floor"] != 0.1 {
		t.Errorf("deltas = %v", deltas)
	}
	if rule, _ := hint["rule"].(string); !strings.Contains(rule, "1-2-5") || hint["complete"] != true {
		t.Errorf("rule %q complete %v", rule, hint["complete"])
	}
	if _, has := out["summary"]; has {
		t.Errorf("summary present without summary=true: %v", out["summary"])
	}
	if n := len(ordRequests(sv)); n != 3 {
		t.Errorf("station requests = %d, want 3 (records, count, one summary read)", n)
	}
}

func TestCovHintAndSummaryShareOneRead(t *testing.T) {
	r, sv := covRig(t)
	out := r.Srv.OK("n4_read_history", Args{"history_id": "/FakeStation/Noisy", "cov_hint": true, "summary": true})
	if out["summary"] == nil || out["cov_hint"] == nil {
		t.Fatalf("summary %v cov_hint %v", out["summary"], out["cov_hint"])
	}
	if n := len(ordRequests(sv)); n != 3 {
		t.Errorf("station requests = %d, want 3", n)
	}
}

func TestCovHintOfANonNumericHistoryIsNullWithANote(t *testing.T) {
	r, _ := covRig(t)
	out := r.Srv.OK("n4_read_history", Args{"history_id": "/FakeStation/Door", "cov_hint": true})
	note, _ := out["cov_hint_note"].(string)
	if v, has := out["cov_hint"]; !has || v != nil || !strings.Contains(note, "not numeric") {
		t.Errorf("cov_hint %v (has %v) note %q", v, has, note)
	}
}

func TestReadHistoryReplyHasNoCovHintUnlessAsked(t *testing.T) {
	r, _ := covRig(t)
	out := r.Srv.OK("n4_read_history", Args{"history_id": "/FakeStation/Noisy"})
	var keys []string
	for k := range out {
		keys = append(keys, k)
	}
	want := map[string]bool{"history_id": true, "columns": true, "rows": true, "row_count": true, "order": true,
		"truncated": true, "total_records": true, "elapsed_ms": true}
	got := map[string]bool{}
	for _, k := range keys {
		got[k] = true
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("reply keys = %v", keys)
	}
}

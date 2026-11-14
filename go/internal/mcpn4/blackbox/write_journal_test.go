package blackbox

// The write journal and audit log (port of test_tools_write TestJournalAndAudit).

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest/boxstation"
)

func TestJournalHoldsAWriteAheadIntentAndAResult(t *testing.T) {
	r := verifiedRig(t)
	out := r.Srv.RunWrite("n4_create_component", createPump)
	lines := r.Lines("journal.jsonl")
	if len(lines) != 2 {
		t.Fatalf("journal = %v", lines)
	}
	intent, result := lines[0], lines[1]
	inverse := []any{map[string]any{"nm": "v", "h": "3", "n": "Pump"}}
	id, _ := intent["batch_id"].(string)
	if got := keys(intent); !reflect.DeepEqual(got, []string{"batch_id", "inverse_plan", "ops", "phase",
		"station_name", "targets", "tool", "ts"}) {
		t.Errorf("intent keys = %v", got)
	}
	if !reflect.DeepEqual(intent["targets"], map[string]any{Folder: "3"}) || intent["phase"] != "intent" ||
		intent["batch_id"] != out["batch_id"] || !batchIDPattern.MatchString(id) ||
		intent["tool"] != "n4_create_component" || intent["station_name"] != StationName ||
		!reflect.DeepEqual(intent["inverse_plan"], inverse) || Dig(objects(intent["ops"])[0], "nm") != "a" {
		t.Errorf("intent = %v", intent)
	}
	if got := keys(result); !reflect.DeepEqual(got, []string{"accepted", "batch_id", "inverse", "phase", "ts", "verdict"}) {
		t.Errorf("result keys = %v", got)
	}
	if result["phase"] != "result" || result["batch_id"] != out["batch_id"] || result["verdict"] != "verified" ||
		!reflect.DeepEqual(result["inverse"], inverse) ||
		!reflect.DeepEqual(result["accepted"], map[string]any{"id": "a", "nn": "Pump"}) {
		t.Errorf("result = %v", result)
	}
}

func TestTheIntentIsOnDiskBeforeTheStationReceivesTheOp(t *testing.T) {
	r := verifiedRig(t)
	var seen [][]any
	r.Station.SetFaults(boxstation.Faults{OnSync: func(*boxstation.Station, map[string]any) {
		seen = append(seen, journalPhasesOnDisk(r.StateDir))
	}})
	r.Srv.RunWrite("n4_create_component", createPump)
	var got [][]any
	r.Station.Do(func(*boxstation.Station) { got = seen })
	if !reflect.DeepEqual(got, [][]any{{"intent"}}) {
		t.Errorf("journal phases at each sync = %v", got)
	}
}

func TestAReadBackThatRaisesAnythingElseIsFailedNotACrash(t *testing.T) {
	r := verifiedRig(t)
	c := r.AddWithOut("Calc", 1.0)
	r.Station.SetFaults(boxstation.Faults{OnSync: func(st *boxstation.Station, _ map[string]any) {
		st.ByHandle(c.Handle).Child("out").Child("value").Value = "not-a-number"
	}})
	out := r.Srv.RunWrite("n4_set_slot", Args{"ord": Folder + "/Calc", "slot": "out", "value": 2.0,
		"value_type": "baja:StatusNumeric"})
	readback, _ := out["readback_error"].(string)
	id, _ := out["batch_id"].(string)
	if out["verdict"] != "failed" || !strings.Contains(readback, "ValueError") || !batchIDPattern.MatchString(id) {
		t.Errorf("out = %v", out)
	}
}

func TestAuditRecordsPlannedExecutedAndRefusedCalls(t *testing.T) {
	r := verifiedRig(t)
	r.Srv.RunWrite("n4_create_component", createPump)
	r.Srv.Err("n4_create_component", with(createPump, Args{"parent_ord": "station:|slot:/", "name": "X"}))
	lines := r.Lines("audit.jsonl")
	if len(lines) != 3 {
		t.Fatalf("audit = %v", lines)
	}
	planned, executed, refused := lines[0], lines[1], lines[2]
	id, _ := executed["batch_id"].(string)
	reason, _ := refused["reason"].(string)
	if planned["outcome"] != "planned" || planned["dry_run"] != true || planned["batch_id"] != nil ||
		executed["outcome"] != "executed" || executed["dry_run"] != false || !batchIDPattern.MatchString(id) ||
		refused["outcome"] != "refused" || refused["batch_id"] != nil || !strings.Contains(reason, "outside the write scope") {
		t.Errorf("audit = %v", lines)
	}
	for _, line := range lines {
		if got := keys(line); !reflect.DeepEqual(got, []string{"args_redacted", "batch_id", "dry_run", "outcome",
			"reason", "tool", "ts"}) {
			t.Errorf("audit keys = %v", got)
		}
	}
}

func TestStateFilesArePrivate(t *testing.T) {
	r := verifiedRig(t)
	r.Srv.RunWrite("n4_create_component", createPump)
	if info, err := os.Stat(r.StateDir); err != nil || info.Mode().Perm() != 0o700 {
		t.Errorf("state dir: %v %v", info, err)
	}
	for _, name := range []string{"journal.jsonl", "audit.jsonl"} {
		if info, err := os.Stat(filepath.Join(r.StateDir, name)); err != nil || info.Mode().Perm() != 0o600 {
			t.Errorf("%s: %v %v", name, info, err)
		}
	}
}

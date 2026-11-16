package blackbox

// n4_alarm_text_check through the real server (retro 2026-11-16 row P2, kit rule LC9): the live
// twin of bog-nav alarm-text (niagara-go internal/bognav/alarmtext.go), same rules and rows.

import (
	"reflect"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest/boxstation"
)

const alarmTextHeader = "Slot Path,To Offnormal Text,To Fault Text,To Normal Text\n"

// alarmTextRows mirror the bog-nav alarm-text fixture: OverAmps (a %parent...% path and no
// toNormalText: both rules), Good (record fields, a lexicon call and %% pass; a source slot path in
// toFaultText does not), Quiet (every text at its "" default: nothing). Rows come in tree order,
// not path order.
const alarmTextRows = alarmTextHeader +
	"slot:/Comp1/OverAmps,Over amps %parent.parent.parent.Ctl.amps1% A,,\n" +
	"slot:/Comp1/Good,\"High %alarmData.presentValue%, %lexicon(x:hi)% 100%%\",Fault %out.value%," +
	"Back at %alarmData.presentValue% (%sourceState%)\n" +
	"slot:/Comp1/Quiet,,,\n"

func alarmTextRig(t *testing.T, body string) (*Rig, *boxstation.Servlets) {
	t.Helper()
	r, sv := servletRig(t, nil)
	routeOrds(sv, boxstation.OrdRoute{Needle: "alarm:AlarmSourceExt", Body: []byte(body)})
	return r, sv
}

func TestAlarmTextCheckReportsTheLC9FindingsOfBogNav(t *testing.T) {
	r, sv := alarmTextRig(t, alarmTextRows)
	out := r.Srv.OK("n4_alarm_text_check", nil)
	want := []any{
		map[string]any{"level": "WARN", "rule": "parent-path", "path": "/Comp1/Good", "slot": "toFaultText",
			"token": "%out.value%", "text": "Fault %out.value%"},
		map[string]any{"level": "WARN", "rule": "parent-path", "path": "/Comp1/OverAmps", "slot": "toOffnormalText",
			"token": "%parent.parent.parent.Ctl.amps1%", "text": "Over amps %parent.parent.parent.Ctl.amps1% A"},
		map[string]any{"level": "WARN", "rule": "empty-tonormal", "path": "/Comp1/OverAmps", "slot": "toNormalText",
			"token": "", "text": ""},
	}
	if !reflect.DeepEqual(out["findings"], want) {
		t.Errorf("findings =\n%v\nwant\n%v", out["findings"], want)
	}
	if out["alarm_extensions"] != float64(3) || out["parent_path"] != float64(2) || out["empty_tonormal"] != float64(1) {
		t.Errorf("counts %v", out)
	}
	if s := out["summary"]; s != "3 alarm source extensions; 2 parent-path, 1 empty-tonormal [LC9]" {
		t.Errorf("summary = %q", s)
	}
	if out["base"] != "station:|slot:/" || out["truncated"] != false {
		t.Errorf("base/truncated %v", out)
	}
	ords := ordRequests(sv)
	if len(ords) != 1 || ords[0] != "station:|slot:/|bql:select slotPath, toOffnormalText, toFaultText, "+
		"toNormalText from alarm:AlarmSourceExt|view:file:ITableToCsv" {
		t.Errorf("ords %q", ords)
	}
}

func TestAlarmTextCheckCleanSubtreeHasNoFindings(t *testing.T) {
	r, sv := alarmTextRig(t, alarmTextHeader+"slot:/Logic/Ok/Ext,High %alarmData.presentValue%,,Normal\n")
	out := r.Srv.OK("n4_alarm_text_check", Args{"base": "station:|slot:/Logic"})
	if f, _ := out["findings"].([]any); f == nil || len(f) != 0 {
		t.Errorf("findings = %v", out["findings"])
	}
	if s := out["summary"]; s != "1 alarm source extensions; 0 parent-path, 0 empty-tonormal [LC9]" {
		t.Errorf("summary = %q", s)
	}
	if ords := ordRequests(sv); len(ords) != 1 || !strings.HasPrefix(ords[0], "station:|slot:/Logic|bql:") {
		t.Errorf("ords %q", ords)
	}
}

func TestAlarmTextCheckRefusesABaseOutsideTheStationTree(t *testing.T) {
	r, sv := alarmTextRig(t, alarmTextRows)
	if text := r.Srv.Err("n4_alarm_text_check", Args{"base": "alarm:"}); !strings.Contains(text, "base must be") {
		t.Errorf("alarm: base: %q", text)
	}
	if code := r.Srv.CallError("n4_alarm_text_check", Args{"limit": 3}); code != -32602 {
		t.Errorf("unknown argument: code %d", code)
	}
	if len(ordRequests(sv)) != 0 {
		t.Errorf("a refused call reached the station")
	}
}

func TestAlarmTextCheckSaysWhenTheRowCapCutTheExtensions(t *testing.T) {
	r, _ := alarmTextRig(t, alarmTextRows)
	out := r.Srv.OK("n4_alarm_text_check", Args{"max_rows": 1})
	if out["truncated"] != true || out["alarm_extensions"] != float64(1) || !strings.Contains(pyStr(out["note"]), "max_rows") {
		t.Errorf("capped: %v", out)
	}
}

func pyStr(v any) string { s, _ := v.(string); return s }

package blackbox

// n4_station_runtime: how long the station has been up, the modules it has loaded with their
// versions and, with --niagara-home, the module jars on disk and where they differ from what is
// loaded, so "a restart is needed" is a fact the tool states.

import (
	"reflect"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest/boxstation"
)

func TestStationRuntimeReportsUptimeAndLoadedModulesAndThatDiskIsUnknownWithoutAHome(t *testing.T) {
	r, _ := rtRig(t, "")
	out := r.Srv.OK("n4_station_runtime", nil)
	up := out["uptime"].(map[string]any)
	want := map[string]any{"status": "ok", "boot_time": "15 nov 2026 20:50 CST", "current_time": "16 nov 2026 00:44 CST",
		"up_time": "3 hours, 53 minutes, 54,985 seconds", "uptime_s": rtUptime}
	if !reflect.DeepEqual(up, want) {
		t.Errorf("uptime = %v", up)
	}
	loaded := out["loaded_modules"].(map[string]any)
	mods := objects(loaded["modules"])
	if loaded["status"] != "ok" || loaded["count"] != 6.0 || loaded["truncated"] != false || len(mods) != 6 {
		t.Fatalf("loaded_modules = %v", loaded)
	}
	if first := mods[0]; !reflect.DeepEqual(first, map[string]any{"name": "LabDash-rt", "vendor": "LabVendor", "version": rtLabDash}) {
		t.Errorf("first loaded module = %v", first)
	}
	disk := out["jars_on_disk"].(map[string]any)
	if disk["status"] != "unavailable" || !strings.Contains(disk["note"].(string), rtHomeNote) {
		t.Errorf("jars_on_disk = %v", disk)
	}
	if out["restart_needed"] != nil || out["mismatches"] != nil || out["restart_note"] == nil {
		t.Errorf("restart_needed = %v, mismatches = %v, note = %v", out["restart_needed"], out["mismatches"], out["restart_note"])
	}
}

func TestStationRuntimeCapsItsListsAndSaysSo(t *testing.T) {
	jars := rtInSync()
	jars[0].version, jars[1].version, jars[2].version = "9", "9", "9"
	r, _ := rtRig(t, rtHome(t, jars))
	out := r.Srv.OK("n4_station_runtime", Args{"max_rows": 2})
	loaded := out["loaded_modules"].(map[string]any)
	if len(objects(loaded["modules"])) != 2 || loaded["truncated"] != true || loaded["count"] != 6.0 {
		t.Errorf("loaded_modules = %v", loaded)
	}
	if len(objects(out["mismatches"])) != 2 || out["mismatches_truncated"] != true || out["mismatch_count"] != 3.0 {
		t.Errorf("mismatches = %v (count %v)", out["mismatches"], out["mismatch_count"])
	}
}

func TestStationRuntimeCannotSayNoRestartWhenAPartIsUnknown(t *testing.T) {
	unreadable := rtInSync()
	unreadable[4] = rtJar{file: rtBadJar, age: rtBeforeBoot} // control-rt.jar is no zip
	r, _ := rtRig(t, rtHome(t, unreadable))
	out := r.Srv.OK("n4_station_runtime", nil)
	disk := out["jars_on_disk"].(map[string]any)
	if out["restart_needed"] != nil || !reflect.DeepEqual(disk["unreadable"], []any{rtBadJar}) || disk["complete"] != false {
		t.Errorf("unreadable jar: restart_needed = %v, jars_on_disk = %v", out["restart_needed"], disk)
	}
	r2, sv := rtRig(t, rtHome(t, rtInSync()))
	shxServeSpy(sv, rtSysInfo, boxstation.SpyPage{Status: 403})
	out = r2.Srv.OK("n4_station_runtime", nil)
	up := out["uptime"].(map[string]any)
	if up["status"] != "forbidden" || up["hint"] == nil || out["restart_needed"] != nil {
		t.Errorf("forbidden sysInfo: uptime = %v, restart_needed = %v", up, out["restart_needed"])
	}
	if out["loaded_modules"].(map[string]any)["status"] != "ok" {
		t.Errorf("loaded_modules = %v", out["loaded_modules"])
	}
	shxServeSpy(sv, rtSysInfo, boxstation.SpyPage{Body: []byte("<html><body><table class='spy-props'></table></body></html>")})
	up = r2.Srv.OK("n4_station_runtime", nil)["uptime"].(map[string]any)
	if up["status"] != "not_recognized" || up["uptime_s"] != nil {
		t.Errorf("page without ticks: uptime = %v", up)
	}
	shxServeSpy(sv, rtModules, boxstation.SpyPage{Body: shxSpyFixture(t, "sysinfo.html")}) // tables, none of modules
	out = r2.Srv.OK("n4_station_runtime", nil)
	if loaded := out["loaded_modules"].(map[string]any); loaded["status"] != "not_recognized" || loaded["hint"] == nil || out["restart_needed"] != nil {
		t.Errorf("page without the module table: loaded_modules = %v, restart_needed = %v", loaded, out["restart_needed"])
	}
}

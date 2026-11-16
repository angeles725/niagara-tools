package blackbox

// n4_station_runtime with --niagara-home: each loaded module beside its jar on disk, and
// restart_needed from the differences.

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestStationRuntimeSaysNoRestartIsNeededWhenTheDiskMatchesWhatIsLoaded(t *testing.T) {
	r, _ := rtRig(t, rtHome(t, rtInSync()))
	out := r.Srv.OK("n4_station_runtime", nil)
	disk := out["jars_on_disk"].(map[string]any)
	if disk["status"] != "ok" || disk["count"] != 7.0 || disk["complete"] != true {
		t.Errorf("jars_on_disk = %v", disk)
	}
	if out["restart_needed"] != false || len(objects(out["mismatches"])) != 0 || out["mismatch_count"] != 0.0 {
		t.Errorf("restart_needed = %v, mismatches = %v", out["restart_needed"], out["mismatches"])
	}
}

func TestStationRuntimeNamesEachMismatchAndSaysARestartIsNeeded(t *testing.T) {
	jars := rtInSync()
	jars[0].version = "1.0.1" // LabDash-rt: another version on disk
	jars[1].age = rtAfterBoot // LabDash-ux: same version, replaced after the boot
	jars = jars[:5]           // web-rt: loaded, its jar gone
	jars = append(jars, rtJar{rtNewJar, "newmod-rt", "newmod", "rt", "2.0.0", rtAfterBoot})
	r, _ := rtRig(t, rtHome(t, jars))
	out := r.Srv.OK("n4_station_runtime", nil)
	want := map[string]string{"LabDash-rt": "version_differs", "LabDash-ux": "modified_after_boot",
		"web-rt": "loaded_not_on_disk", "newmod-rt": "not_loaded_modified_after_boot"}
	if got := rtReasons(t, out); !reflect.DeepEqual(got, want) {
		t.Errorf("mismatches = %v", got)
	}
	if out["restart_needed"] != true || out["mismatch_count"] != 4.0 {
		t.Errorf("restart_needed = %v, mismatch_count = %v", out["restart_needed"], out["mismatch_count"])
	}
	for _, m := range objects(out["mismatches"]) {
		if m["module"] == "LabDash-rt" && (m["loaded_version"] != rtLabDash || m["disk_version"] != "1.0.1" || m["jar"] != "LabDash-rt.jar") {
			t.Errorf("LabDash-rt mismatch = %v", m)
		}
	}
}

func TestStationRuntimeNeverFollowsALinkedJar(t *testing.T) {
	home := rtHome(t, rtInSync())
	elsewhere := filepath.Join(t.TempDir(), "outside.jar")
	rtWriteJar(t, elsewhere, rtJar{"outside.jar", "outside-rt", "outside", "rt", "1", 0})
	if err := os.Symlink(elsewhere, filepath.Join(home, "modules", "outside-rt.jar")); err != nil {
		t.Skip("no symlinks here:", err)
	}
	r, _ := rtRig(t, home)
	out := r.Srv.OK("n4_station_runtime", nil)
	disk := out["jars_on_disk"].(map[string]any)
	if disk["count"] != 7.0 || !reflect.DeepEqual(disk["skipped"], []any{"outside-rt.jar"}) {
		t.Errorf("jars_on_disk = %v", disk)
	}
}

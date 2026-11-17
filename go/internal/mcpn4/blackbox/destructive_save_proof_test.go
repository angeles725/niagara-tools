package blackbox

// n4_save_station's persistence proofs from overamps retro rows M5 (no --station-home: the
// station's own StationSaveJob under Services/JobService) and M9 (a config.bog caught mid-rewrite
// is read again once).

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest/boxstation"
)

// withJobService gives the fake station Services/JobService holding one old successful save job,
// and makes each save add StationSaveJob1 ending in state.
func (d *dRig) withJobService(state string) {
	d.Station.Do(func(st *boxstation.Station) {
		services := st.Component(st.Root(), "Services", "baja:ServiceContainer", "")
		js := st.Component(services, "JobService", "baja:JobService", "")
		addJob := func(name, state, start string) {
			job := st.Component(js, name, "baja:StationSaveJob", "")
			job.Children = append(job.Children, &boxstation.Node{Name: "jobState", Type: "baja:JobState", Value: state},
				&boxstation.Node{Name: "startTime", Type: "baja:AbsTime", Value: start},
				&boxstation.Node{Name: "endTime", Type: "baja:AbsTime", Value: start})
		}
		addJob("StationSaveJob", "success", "2026-11-16T10:00:00.000-06:00")
		st.AfterInvoke = func(st *boxstation.Station, action string, _ *boxstation.Node) {
			if action == "save" {
				addJob("StationSaveJob1", state, "2026-11-17T09:00:00.000-06:00")
			}
		}
	})
}

func TestARemoteSaveIsProvedByTheStationsNewSaveJob(t *testing.T) {
	d := destructiveRig(t, nil)
	d.withJobService("success")
	out := d.Srv.RunWrite("n4_save_station", nil)
	if out["persisted"] != true || out["verdict"] != "verified" || Dig(out, "evidence", "route") != "station save job" {
		t.Fatalf("out = %v", out)
	}
	if Dig(out, "evidence", "job", "ord") != "station:|slot:/Services/JobService/StationSaveJob1" ||
		Dig(out, "evidence", "job", "jobState") != "success" || d.Station.Saves() != 1 {
		t.Errorf("evidence = %v", out["evidence"])
	}
}

func TestARemoteSaveWhoseJobFailedIsNotPersisted(t *testing.T) {
	d := destructiveRig(t, nil)
	d.withJobService("failed")
	out := d.Srv.RunWrite("n4_save_station", nil)
	hint, _ := Dig(out, "evidence", "hint").(string)
	if out["persisted"] != false || out["verdict"] != "mismatch" || !strings.Contains(hint, "ended failed") {
		t.Fatalf("out = %v", out)
	}
}

// Without a JobService to read, nothing changed that the server can see: unknown, with why.
func TestARemoteSaveWithNoReadableSaveJobsIsUnknownWithWhy(t *testing.T) {
	d := destructiveRig(t, nil)
	out := d.Srv.RunWrite("n4_save_station", nil)
	hint, _ := Dig(out, "evidence", "hint").(string)
	if out["persisted"] != "unknown" || out["verdict"] != "unverified" ||
		!strings.Contains(hint, "save jobs could not be read before the save") ||
		!strings.Contains(hint, "--station-home FakeStation=") {
		t.Fatalf("out = %v", out)
	}
}

// M9: the save leaves config.bog absent for a moment (moved to its backup before the working
// file is renamed over it); the read-back waits and reads once more, and proves the save.
func TestAConfigBogReadMidRewriteIsRetriedOnceAndProvesTheSave(t *testing.T) {
	d := destructiveRig(t, nil)
	path := d.homeServer("bog", false)
	d.Station.Do(func(st *boxstation.Station) {
		st.AfterInvoke = func(_ *boxstation.Station, action string, _ *boxstation.Node) {
			if action == "save" {
				_ = os.Remove(path)
				go func() {
					time.Sleep(200 * time.Millisecond)
					_ = os.WriteFile(path, []byte("bog-saved"), 0o600)
				}()
			}
		}
	})
	out := d.Srv.RunWrite("n4_save_station", nil)
	if out["persisted"] != true || out["verdict"] != "verified" || Dig(out, "evidence", "retry", "after_s") != 1.0 {
		t.Fatalf("out = %v", out)
	}
}

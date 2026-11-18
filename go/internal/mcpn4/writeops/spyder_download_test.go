package writeops_test

import (
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/writeops"
)

const dlJob = "station:|slot:/Services/JobService/SpyderDownloadJob4"

// dlStation is a Spyder 2012 and its download job in state; readLog answers log.
func dlStation(state, log string) *asks {
	spyder := bacnetNet + "/$32012"
	return &asks{loads: loads{nodes: obj(
		spyder, obj("", obj("h", "3e50d", "t", "honeywellBacnetSpyder:BacnetSpyder")),
		dlJob, obj("", obj("h", "j4", "t", "honeywellSpyderTool:SpyderDownloadJob"), "jobState", obj("t", "baja:JobState", "v", state)),
	)}, online: obj("t", "baja:String", "v", log)}
}

// Live 2026-11-18 (2012, a PUB1012S): spyderDownload with SpyderDownloadParameters
// {fullDownload: true} returned the job ORD; the job validated, compiled and downloaded in about
// two minutes and its readLog ended "Success|...|Job Success|".
func TestSpyderDownloadPlansTheFullDownload(t *testing.T) {
	c := dlStation("success", "")
	args := obj("ord", bacnetNet+"/$32012", "action", "spyderDownload", "arg", "full")
	p, err := writeops.InvokeAction.PlanWith(c, args, clock(), nil)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"ssc": "invokeAction", "arg": {"h": "3e50d", "a": "spyderDownload", "b": {"nm": "p", "t": "honeywellSpyderTool:SpyderDownloadParameters", "s": [{"nm": "p", "n": "fullDownload", "t": "baja:Boolean", "v": "true"}]}}}`
	if got := js(t, p.Ops[0]); got != want {
		t.Errorf("op = %s", got)
	}
	if notes := js(t, p.Notes); !strings.Contains(notes, "restarts the controller") || !strings.Contains(notes, "no inverse") {
		t.Errorf("notes = %s", notes)
	}
	quick, err := writeops.InvokeAction.PlanWith(c, obj("ord", bacnetNet+"/$32012", "action", "spyderDownload", "arg", "quick"), clock(), nil)
	if err != nil || !strings.Contains(js(t, quick.Ops[0]), `"n": "fullDownload", "t": "baja:Boolean", "v": "false"`) {
		t.Errorf("quick: %v %v", err, quick)
	}
}

// spyderDownload needs a Spyder and arg full or quick.
func TestSpyderDownloadRefusals(t *testing.T) {
	c := dlStation("success", "")
	for _, args := range []*pyjson.Object{
		obj("ord", bacnetNet+"/$32012", "action", "spyderDownload"),
		obj("ord", bacnetNet+"/$32012", "action", "spyderDownload", "arg", "all"),
	} {
		if _, err := writeops.InvokeAction.PlanWith(c, args, clock(), nil); err == nil || !strings.Contains(err.Error(), "full or quick") {
			t.Errorf("%s: err = %v", js(t, args), err)
		}
	}
	other := &loads{nodes: obj(bacnetNet+"/Dev", obj("", obj("h", "d", "t", "bacnet:BacnetDevice")))}
	if _, err := writeops.InvokeAction.PlanWith(other, obj("ord", bacnetNet+"/Dev", "action", "spyderDownload", "arg", "full"), clock(), nil); err == nil ||
		!strings.Contains(err.Error(), "honeywellBacnetSpyder:BacnetSpyder") {
		t.Errorf("non-Spyder: err = %v", err)
	}
}

// The read-back waits for the job and returns its state and log lines: success verifies, a failed
// job is a mismatch and one still running at the deadline is unverified.
func TestSpyderDownloadReadsTheJobBack(t *testing.T) {
	log := "Message|2026-11-18T14:44:44|Download started on device - $32012|\nMessage|2026-11-18T14:44:47|Validation success|\nSuccess|2026-11-18T14:46:54|Job Success|\n"
	args := obj("ord", bacnetNet+"/$32012", "action", "spyderDownload", "arg", "full")
	reply := obj("t", "baja:Ord", "v", "slot:/Services/JobService/SpyderDownloadJob4")
	for state, verdict := range map[string]string{"success": "verified", "failed": "mismatch", "running": "unverified"} {
		c := dlStation(state, log)
		p, err := writeops.InvokeAction.PlanWith(c, args, clock(), nil)
		if err != nil {
			t.Fatal(err)
		}
		_, _, observed, v, err := writeops.InvokeAction.Readback(c, args, p, []any{reply}, nil)
		if err != nil || v != verdict {
			t.Errorf("%s: verdict %q err %v", state, v, err)
			continue
		}
		got := js(t, observed)
		for _, want := range []string{`"job": "` + dlJob + `"`, `"job_state": "` + state + `"`, `"Validation success"`, `"Success: Job Success"`} {
			if !strings.Contains(got, want) {
				t.Errorf("%s: missing %s in %s", state, want, got)
			}
		}
	}
}

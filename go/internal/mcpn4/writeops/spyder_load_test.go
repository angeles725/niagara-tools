package writeops_test

import (
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/writeops"
)

const loadJob = "station:|slot:/Services/JobService/LoadJob7"

// loadStation is a Spyder 2012 whose logicLoaded slot is loaded (and whose ControlProgram child
// is there when withProgram) and a LoadJob in state; readLog answers a one-line log.
func loadStation(loaded string, withProgram bool, state string) *asks {
	spyder := bacnetNet + "/$32012"
	slots := obj("", obj("h", "3e50d", "t", "honeywellBacnetSpyder:BacnetSpyder"), "logicLoaded", obj("t", "baja:Boolean", "v", loaded))
	if withProgram {
		slots.Set("ControlProgram", obj("t", "honeywellSpyderTool:ApplicationLogic"))
	}
	nodes := obj(
		spyder, slots,
		loadJob, obj("", obj("h", "j7", "t", "honeywellSpyderTool:LoadJob"), "jobState", obj("t", "baja:JobState", "v", state)),
	)
	return &asks{loads: loads{nodes: nodes}, online: obj("t", "baja:String", "v", "Success|t|Job Success|\n")}
}

var (
	loadArgs  = obj("ord", bacnetNet+"/$32012", "action", "loadControlProgram")
	loadReply = obj("t", "baja:Ord", "v", "slot:/Services/JobService/LoadJob7")
)

// loadControlProgram (BBacnetSpyder.java:518, :1180) takes no argument and returns the LoadJob
// ORD; it has no inverse, can refuse with HeapLimitReached (BDeviceModeActions.java:245) and
// returns null when the program is already loaded (:241).
func TestLoadControlProgramPlansTheNoArgAction(t *testing.T) {
	p, err := writeops.InvokeAction.PlanWith(loadStation("false", false, "success"), loadArgs, clock(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if got := js(t, p.Ops[0]); got != `{"ssc": "invokeAction", "arg": {"h": "3e50d", "a": "loadControlProgram"}}` {
		t.Errorf("op = %s", got)
	}
	if got := js(t, p.Inverse); got != "[]" {
		t.Errorf("inverse = %s", got)
	}
	notes := js(t, p.Notes)
	for _, want := range []string{"no inverse", "HeapLimitReached", "already loaded", "no job"} {
		if !strings.Contains(notes, want) {
			t.Errorf("notes lack %q: %s", want, notes)
		}
	}
}

func TestLoadControlProgramRefusals(t *testing.T) {
	c := loadStation("false", false, "success")
	withArg := obj("ord", bacnetNet+"/$32012", "action", "loadControlProgram", "arg", "x")
	if _, err := writeops.InvokeAction.PlanWith(c, withArg, clock(), nil); err == nil || err.Error() != "action 'loadControlProgram' takes no arg" {
		t.Errorf("arg: err = %v", err)
	}
	other := &loads{nodes: obj(bacnetNet+"/Dev", obj("", obj("h", "d", "t", "bacnet:BacnetDevice")))}
	_, err := writeops.InvokeAction.PlanWith(other, obj("ord", bacnetNet+"/Dev", "action", "loadControlProgram"), clock(), nil)
	if err == nil || !strings.Contains(err.Error(), "needs a honeywellBacnetSpyder:BacnetSpyder") {
		t.Errorf("non-Spyder: err = %v", err)
	}
}

// The read-back waits for the LoadJob, then reports whether the program is loaded (logicLoaded,
// set by BLoadJob.java:126-192 when it ends, and the ControlProgram child): a job that ends
// without the program loaded, or fails, is a mismatch; one still running is unverified.
func TestLoadControlProgramReadsTheJobAndTheLoadedStateBack(t *testing.T) {
	for _, tc := range []struct {
		name, loaded, state, verdict string
		program                      bool
		want                         []string
	}{
		{"loaded", "true", "success", "verified", true, []string{`"job": "` + loadJob + `"`, `"job_state": "success"`, `"logic_loaded": true`, `"control_program": true`}},
		{"job ended but not loaded", "false", "success", "mismatch", false, []string{`"logic_loaded": false`, `"control_program": false`, `"hint"`}},
		{"failed", "false", "failed", "mismatch", false, []string{`"job_state": "failed"`, `"logic_loaded": false`}},
		{"running", "false", "running", "unverified", false, []string{`"job_state": "running"`}},
	} {
		c := loadStation(tc.loaded, tc.program, tc.state)
		p, err := writeops.InvokeAction.PlanWith(c, loadArgs, clock(), nil)
		if err != nil {
			t.Fatal(err)
		}
		_, _, observed, v, err := writeops.InvokeAction.Readback(c, loadArgs, p, []any{loadReply}, nil)
		if err != nil || v != tc.verdict {
			t.Errorf("%s: verdict %q err %v observed %s", tc.name, v, err, js(t, observed))
			continue
		}
		for _, want := range tc.want {
			if !strings.Contains(js(t, observed), want) {
				t.Errorf("%s: missing %s in %s", tc.name, want, js(t, observed))
			}
		}
	}
}

// An already loaded program returns no job (null): the read-back says there was no job to wait
// for and reports the loaded state; a null reply with nothing loaded cannot be told from a
// refusal and stays unverified, with the reason.
func TestLoadControlProgramWithNoJobReportsTheLoadedState(t *testing.T) {
	for _, reply := range []any{nil, obj("t", "baja:Ord", "v", "null")} {
		for _, tc := range []struct{ loaded, verdict, want string }{
			{"true", "verified", `"logic_loaded": true`},
			{"false", "unverified", `"hint"`},
		} {
			c := loadStation(tc.loaded, tc.loaded == "true", "success")
			p, _ := writeops.InvokeAction.PlanWith(c, loadArgs, clock(), nil)
			_, _, observed, v, err := writeops.InvokeAction.Readback(c, loadArgs, p, []any{reply}, nil)
			got := js(t, observed)
			if err != nil || v != tc.verdict || !strings.Contains(got, `"job": null`) || !strings.Contains(got, tc.want) {
				t.Errorf("loaded %s: verdict %q err %v observed %s", tc.loaded, v, err, got)
			}
		}
	}
}

// compile keeps its read-back after the load step joined the dispatch.
func TestCompileReadbackIsUnchangedByTheLoadStep(t *testing.T) {
	c := dlStation("success", "Message|t|Compilation started|\nSuccess|t|Job Success|\n")
	args := obj("ord", bacnetNet+"/$32012", "action", "compile")
	p, _ := writeops.InvokeAction.PlanWith(c, args, clock(), nil)
	_, _, observed, v, err := writeops.InvokeAction.Readback(c, args, p, []any{obj("v", "slot:/Services/JobService/SpyderDownloadJob4")}, nil)
	if err != nil || v != "verified" || strings.Contains(js(t, observed), "logic_loaded") {
		t.Errorf("verdict %q err %v observed %s", v, err, js(t, observed))
	}
}

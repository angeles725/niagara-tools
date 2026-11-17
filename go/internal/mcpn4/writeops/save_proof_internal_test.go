package writeops

// The persistence proofs of overamps retro rows M5 (the station's own save job, no
// --station-home) and M9 (one retry of a config.bog caught mid-rewrite).

import (
	"os"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// saveJob adds a StationSaveJob named name under the JobService with its state and times.
func (r *srig) saveJob(js *tnode, name, state, start string) *tnode {
	job := r.st.add(js, name, "baja:StationSaveJob")
	job.kids = append(job.kids, &tnode{name: "jobState", t: "baja:JobState", v: state},
		&tnode{name: "startTime", t: "baja:AbsTime", v: start},
		&tnode{name: "endTime", t: "baja:AbsTime", v: start})
	return job
}

// jobService gives the station Services/JobService holding one old, successful save job.
func (r *srig) jobService() *tnode {
	services := r.st.add(r.st.root, "Services", "baja:ServiceContainer")
	js := r.st.add(services, "JobService", "baja:JobService")
	r.saveJob(js, "StationSaveJob", "success", "2026-11-16T10:00:00.000-06:00")
	return js
}

func evidenceOf(t *testing.T, out *pyjson.Object) *pyjson.Object {
	t.Helper()
	ev, ok := get(out, "evidence").(*pyjson.Object)
	if !ok {
		t.Fatal(pyjson.Repr(out))
	}
	return ev
}

func TestANewSuccessfulSaveJobProvesPersistenceWithoutAStationHome(t *testing.T) {
	r := newSRig(t, "")
	js := r.jobService()
	r.st.onSave = func() { r.saveJob(js, "StationSaveJob1", "success", "2026-11-17T09:00:00.000-06:00") }
	out := r.save(t)
	ev := evidenceOf(t, out)
	if get(out, "persisted") != true || get(out, "verdict") != "verified" || r.sleeps != 0 {
		t.Fatalf("%s after %d sleeps", pyjson.Repr(out), r.sleeps)
	}
	job := get(ev, "job").(*pyjson.Object)
	if get(ev, "route") != "station save job" || get(ev, "jobs_before") != 1 ||
		get(job, "ord") != "station:|slot:/Services/JobService/StationSaveJob1" || get(job, "jobState") != "success" ||
		get(job, "startTime") != "2026-11-17T09:00:00.000-06:00" {
		t.Errorf("evidence %s", pyjson.Repr(ev))
	}
}

func TestNoNewSaveJobIsUnknownWithWhyAfterThePoll(t *testing.T) {
	r := newSRig(t, "")
	r.jobService()
	out := r.save(t)
	ev := evidenceOf(t, out)
	hint, _ := get(ev, "hint").(string)
	if get(out, "persisted") != "unknown" || get(out, "verdict") != "unverified" || r.sleeps != 4 || r.now != 1.0 ||
		!strings.HasPrefix(hint, "no new StationSaveJob appeared under Services/JobService within 1 s") ||
		!strings.Contains(hint, "--station-home FakeStation=") || get(ev, "route") != "station save job" {
		t.Fatalf("%s after %d sleeps", pyjson.Repr(out), r.sleeps)
	}
}

func TestAFailedSaveJobIsNotPersisted(t *testing.T) {
	r := newSRig(t, "")
	js := r.jobService()
	r.st.onSave = func() { r.saveJob(js, "StationSaveJob1", "failed", "2026-11-17T09:00:00.000-06:00") }
	out := r.save(t)
	if hint := get(evidenceOf(t, out), "hint"); get(out, "persisted") != false || get(out, "verdict") != "mismatch" ||
		hint != "the station's save job ended failed: config.bog was not rewritten" {
		t.Fatal(pyjson.Repr(out))
	}
}

// A running job is polled until it completes; a reused name with a new start time is a new job.
func TestARunningSaveJobIsPolledUntilItSucceeds(t *testing.T) {
	r := newSRig(t, "")
	js := r.jobService()
	var job *tnode
	r.st.onSave = func() {
		js.kids = nil // the old job was housekept; the new one reuses its name
		job = r.saveJob(js, "StationSaveJob", "running", "2026-11-17T09:00:00.000-06:00")
	}
	r.state.Sleep = func(s float64) {
		r.sleeps++
		r.now += s
		if r.sleeps == 2 {
			job.kids[0].v = "success"
		}
	}
	out := r.save(t)
	if get(out, "persisted") != true || r.sleeps != 2 {
		t.Fatalf("%s after %d sleeps", pyjson.Repr(out), r.sleeps)
	}
}

func TestASaveJobStillRunningAtTheTimeoutIsUnknown(t *testing.T) {
	r := newSRig(t, "")
	js := r.jobService()
	r.st.onSave = func() { r.saveJob(js, "StationSaveJob1", "running", "2026-11-17T09:00:00.000-06:00") }
	out := r.save(t)
	hint, _ := get(evidenceOf(t, out), "hint").(string)
	if get(out, "persisted") != "unknown" || !strings.HasPrefix(hint, "the station's save job is still running after 1 s") {
		t.Fatal(pyjson.Repr(out))
	}
}

// M9: config.bog is absent right after the save (moved to its backup, the working file not yet
// renamed over it); one read after bogSettle finds the new file and proves the save.
func TestAConfigBogReadMidRewriteIsRetriedOnceAndProvesTheSave(t *testing.T) {
	r := newSRig(t, "bog")
	r.st.rewrites = false
	r.st.onSave = func() { os.Remove(r.st.bog) }
	r.state.Sleep = func(s float64) {
		r.sleeps++
		r.now += s
		if r.sleeps == 1 {
			os.WriteFile(r.st.bog, []byte("bog-saved"), 0o600)
		}
	}
	out := r.save(t)
	ev := evidenceOf(t, out)
	retry, _ := get(ev, "retry").(*pyjson.Object)
	if get(out, "persisted") != true || get(out, "verdict") != "verified" || r.sleeps != 1 || r.now != bogSettle ||
		retry == nil || get(retry, "after_s") != bogSettle {
		t.Fatalf("%s after %d sleeps", pyjson.Repr(out), r.sleeps)
	}
	if bogSettle > 3 {
		t.Errorf("the settle wait %v s is over the 3 s bound", bogSettle)
	}
}

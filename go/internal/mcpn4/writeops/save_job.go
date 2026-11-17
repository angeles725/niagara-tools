package writeops

import (
	"strconv"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// The remote persistence route (overamps retro row M5): with no --station-home, config.bog cannot
// be read (the file servlet refuses every .bog, web-rt FileServlet.java:451), but the station
// reports its own save. BStation.save runs Station.saveAsync, which submits a BStationSaveJob:
// a TRANSIENT child of Services/JobService named StationSaveJob, StationSaveJob1, ... whose
// jobState ends success, failed or canceled once config.bog.working is renamed over config.bog
// (baja Station.java:409-519, BJobService.java:178). A new job that ends success is the
// station's own evidence of the save; the job list is read before the save to tell new from old.

const (
	jobServiceOrd = "station:|slot:/Services/JobService"
	saveJobType   = "baja:StationSaveJob"
	saveJobRoute  = "station save job"
)

// saveJobs is every StationSaveJob under the JobService as {name|startTime: job}, in load order.
func saveJobs(c plan.Client) (*pyjson.Object, error) {
	if c == nil {
		return nil, plan.ToolError("no station client")
	}
	nodes, err := c.LoadTree(jobServiceOrd, 2, box.LoadOptions{})
	if err != nil {
		return nil, err
	}
	out := pyjson.NewObject()
	kids := box.Children(nodes, "")
	for _, name := range kids.Keys() {
		if get(box.Node(kids, name), "t") != saveJobType {
			continue
		}
		props := box.Children(nodes, name)
		job := obj("ord", jobServiceOrd+"/"+name, "jobState", box.ChildValue(props, "jobState", "unknown"),
			"startTime", box.ChildValue(props, "startTime", nil), "endTime", box.ChildValue(props, "endTime", nil))
		out.Set(name+"|"+pyjson.Str(get(job, "startTime")), job)
	}
	return out, nil
}

// jobComplete is BJob.isComplete on a job state tag.
func jobComplete(state any) bool {
	return state == "success" || state == "failed" || state == "canceled"
}

// newSaveJob is the first job of after that before does not hold, preferring a complete one.
func newSaveJob(before, after *pyjson.Object) *pyjson.Object {
	var found *pyjson.Object
	for _, k := range after.Keys() {
		if _, old := before.Get(k); old {
			continue
		}
		job := box.Node(after, k)
		if jobComplete(get(job, "jobState")) {
			return job
		}
		if found == nil {
			found = job
		}
	}
	return found
}

// jobPersisted is the remote read-back: poll the JobService until a new StationSaveJob is
// complete or the timeout passes. A new job that ended success proves the save; failed or
// canceled disproves it; anything else stays unknown, with why.
func jobPersisted(c plan.Client, d saveData) (any, *pyjson.Object, string) {
	configure := "; configure --station-home " + d.station + "=<directory containing config.bog> to check config.bog directly"
	evidence := obj("route", saveJobRoute, "ord", jobServiceOrd)
	if d.jobsErr != "" {
		evidence.Set("hint", "the station's save jobs could not be read before the save ("+d.jobsErr+")"+configure)
		return "unknown", evidence, "unverified"
	}
	deadline := d.w.Monotonic() + d.w.SaveTimeout
	var job *pyjson.Object
	for {
		after, err := saveJobs(c)
		if err != nil {
			evidence.Set("hint", "the station's save jobs could not be read after the save ("+err.Error()+")"+configure)
			return "unknown", evidence, "unverified"
		}
		job = newSaveJob(d.jobsBefore, after)
		if job != nil && jobComplete(get(job, "jobState")) || d.w.Monotonic() >= deadline {
			break
		}
		d.w.Sleep(d.w.SaveInterval)
	}
	evidence.Set("jobs_before", d.jobsBefore.Len())
	timeout := strconv.FormatFloat(d.w.SaveTimeout, 'g', -1, 64) + " s"
	switch state := get(job, "jobState"); {
	case job == nil:
		evidence.Set("hint", "no new StationSaveJob appeared under Services/JobService within "+timeout+
			" (a station already saving skips this save)"+configure)
	case state == "success":
		evidence.Set("job", job)
		return true, evidence, "verified"
	case jobComplete(state):
		evidence.Set("job", job)
		evidence.Set("hint", "the station's save job ended "+pyjson.Str(state)+": config.bog was not rewritten")
		return false, evidence, "mismatch"
	default:
		evidence.Set("job", job)
		evidence.Set("hint", "the station's save job is still "+pyjson.Str(state)+" after "+timeout+configure)
	}
	return "unknown", evidence, "unverified"
}

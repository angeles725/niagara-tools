package writeops

import (
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/session"
)

// SaveStation is n4_save_station: the save action on the station root, whole-station (any
// write scope will do). The station's null reply proves nothing: with a --station-home the
// read-back polls config.bog until it changes (save_bog.go), else it polls the station's own
// save job (save_job.go); without either proof persistence is unknown.
var SaveStation = plan.Impl{ScopeOrds: func(*pyjson.Object) []any { return nil }, PlanWith: savePlan,
	Readback: saveReadback}

// saveData is the plan's private data: the station home ("" when none is configured), the
// config.bog state before the save (nil when unreadable or no home), the station's save jobs
// before the save when there is no home (or why they could not be read), the station name and
// the write state whose timing the read-back polls with.
type saveData struct {
	home, station string
	backup        bool // zip the station directory after the save (save_backup.go)
	before        *pyjson.Object
	jobsBefore    *pyjson.Object
	jobsErr       string
	w             *plan.State
}

// savePlan is _save_plan with the write state _process adds to its data.
func savePlan(c plan.Client, args *pyjson.Object, w *plan.State, sess *session.Session) (*plan.Planned, error) {
	d := saveData{home: w.StationHomes[sess.StationName], station: sess.StationName, w: w}
	notes, err := backupPlan(&d, args)
	if err != nil {
		return nil, err
	}
	note := "the BOX reply to save is null and proves nothing; "
	if d.home != "" {
		note += "persistence is checked on " + bogPath(d.home)
		d.before = bogState(d.home)
	} else {
		note += "no --station-home for this station, so persistence is checked on the station's own " +
			"save job (a new StationSaveJob under Services/JobService that ends success)"
		if d.jobsBefore, err = saveJobs(c); err != nil {
			d.jobsErr = err.Error()
		}
	}
	op := obj("ssc", "invokeAction", "arg", obj("h", sess.RootHandle, "a", "save"))
	notes = append([]any{note, "no inverse: a save cannot be undone"}, notes...)
	return &plan.Planned{Ops: []any{op}, Notes: notes, Data: d}, nil
}

// saveReadback is _save_readback: the persistence proof, then the backup zip when one was asked.
func saveReadback(c plan.Client, args *pyjson.Object, p *plan.Planned, replies, inverse []any) (any, any, any, string, error) {
	requested, accepted, observed, verdict, err := savePersisted(c, args, p, replies, inverse)
	if d := p.Data.(saveData); err == nil && d.backup {
		requested.(*pyjson.Object).Set("backup", true)
		b := backupAfterSave(d)
		observed.(*pyjson.Object).Set("backup", b)
		if _, failed := b.Get("error"); failed { // a backup was asked for and none exists: never a verified call
			verdict = "mismatch"
		}
	}
	return requested, accepted, observed, verdict, err
}

// savePersisted is the persistence half of the read-back: the station's save job without a home,
// unknown with a home whose config.bog was unreadable before the save, else config.bog before and
// after.
func savePersisted(c plan.Client, _ *pyjson.Object, p *plan.Planned, replies, _ []any) (any, any, any, string, error) {
	d := p.Data.(saveData)
	requested := obj("action", "save")
	var persisted any
	var evidence *pyjson.Object
	var verdict string
	switch {
	case d.home == "":
		persisted, evidence, verdict = jobPersisted(c, d)
	case d.before == nil:
		persisted, verdict = "unknown", "unverified"
		evidence = obj("hint", "config.bog was not readable at "+bogPath(d.home)+" before the save; check that "+
			"--station-home "+d.station+" points at the station directory")
	default:
		persisted, evidence, verdict = bogPersisted(d)
	}
	return requested, replies[0], obj("persisted", persisted, "evidence", evidence), verdict, nil
}

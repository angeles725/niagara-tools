package writeops

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/session"
)

// SaveStation is n4_save_station: the save action on the station root, whole-station (any
// write scope will do). The station's null reply proves nothing: with a --station-home the
// read-back polls config.bog until it changes, else persistence is unknown.
var SaveStation = plan.Impl{ScopeOrds: func(*pyjson.Object) []any { return nil }, PlanWith: savePlan,
	Readback: saveReadback}

// saveData is the plan's private data: the station home ("" when none is configured), the
// config.bog state before the save (nil when unreadable or no home), the station name and the
// write state whose timing the read-back polls with.
type saveData struct {
	home, station string
	before        *pyjson.Object
	w             *plan.State
}

// bogPath is os.path.join(home, "config.bog") for the non-empty home a plan checks.
func bogPath(home string) string {
	if strings.HasSuffix(home, "/") {
		return home + "config.bog"
	}
	return home + "/config.bog"
}

// bogState is _bog_state: config.bog's mtime (st_mtime), size and sha256, nil when it cannot be
// read (an OSError). The file is stat'ed once open, not by path first: the same outcome bar a
// race.
func bogState(home string) *pyjson.Object {
	f, err := os.Open(bogPath(home))
	if err != nil {
		return nil
	}
	defer f.Close()
	st, err := f.Stat()
	digest := sha256.New()
	if err == nil {
		_, err = io.Copy(digest, f)
	}
	if err != nil {
		return nil
	}
	t := st.ModTime() // st_mtime is sec + nsec * 1e-9 as a double
	return obj("mtime", float64(t.Unix())+float64(t.Nanosecond())*1e-9, "size", st.Size(),
		"sha256", hex.EncodeToString(digest.Sum(nil)))
}

// savePlan is _save_plan with the write state _process adds to its data.
func savePlan(_ plan.Client, _ *pyjson.Object, w *plan.State, sess *session.Session) (*plan.Planned, error) {
	d := saveData{home: w.StationHomes[sess.StationName], station: sess.StationName, w: w}
	note := "the BOX reply to save is null and proves nothing; "
	if d.home != "" {
		note += "persistence is checked on " + bogPath(d.home)
		d.before = bogState(d.home)
	} else {
		note += "no --station-home for this station, so persistence cannot be checked"
	}
	op := obj("ssc", "invokeAction", "arg", obj("h", sess.RootHandle, "a", "save"))
	return &plan.Planned{Ops: []any{op}, Notes: []any{note, "no inverse: a save cannot be undone"},
		Data: d}, nil
}

// saveReadback is _save_readback: unknown without a home or a readable config.bog before the
// save; else config.bog is polled until it is unreadable, changed (sha256 or a later mtime) or
// the timeout passes: persisted, unknown or not.
func saveReadback(_ plan.Client, _ *pyjson.Object, p *plan.Planned, replies, _ []any) (any, any, any, string, error) {
	d := p.Data.(saveData)
	requested := obj("action", "save")
	unknown := func(hint string) (any, any, any, string, error) {
		return requested, replies[0], obj("persisted", "unknown", "evidence", obj("hint", hint)), "unverified", nil
	}
	if d.home == "" {
		return unknown("configure --station-home " + d.station + "=<directory containing config.bog> to check persistence")
	}
	path := bogPath(d.home)
	if d.before == nil {
		return unknown("config.bog was not readable at " + path + " before the save; check that --station-home " +
			d.station + " points at the station directory")
	}
	deadline := d.w.Monotonic() + d.w.SaveTimeout
	var after *pyjson.Object
	for {
		after = bogState(d.home)
		if after == nil || changed(d.before, after) || d.w.Monotonic() >= deadline {
			break
		}
		d.w.Sleep(d.w.SaveInterval)
	}
	evidence := obj("path", path, "before", d.before, "after", nil)
	var persisted any = false
	verdict := "mismatch"
	switch {
	case after == nil:
		evidence.Set("hint", "config.bog became unreadable after the save")
		persisted, verdict = "unknown", "unverified"
	case changed(d.before, after):
		persisted, verdict = true, "verified"
	}
	if after != nil { // a nil *Object would not be None
		evidence.Set("after", after)
	}
	return requested, replies[0], obj("persisted", persisted, "evidence", evidence), verdict, nil
}

// changed is a new sha256 or a later mtime.
func changed(before, after *pyjson.Object) bool {
	return get(after, "sha256") != get(before, "sha256") || get(after, "mtime").(float64) > get(before, "mtime").(float64)
}

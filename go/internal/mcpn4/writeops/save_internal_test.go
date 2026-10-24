package writeops

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// saveStation is a tstation that answers the save action on its root as the Python
// FakeStation does: null, and (when rewrites is set) config.bog gets bytes appended and its
// mtime moved 5 s ahead.
type saveStation struct {
	*tstation
	bog      string
	rewrites bool
	saves    int
	onSave   func()
}

func (s *saveStation) SSC(key string, arg any) (any, error) {
	a := arg.(*pyjson.Object)
	if key != "invokeAction" || get(a, "a") != "save" || get(a, "h") != "2" {
		return s.tstation.SSC(key, arg)
	}
	s.sent = append(s.sent, arg)
	s.saves++
	if s.bog != "" && s.rewrites {
		f, err := os.OpenFile(s.bog, os.O_APPEND|os.O_WRONLY, 0o600)
		if err != nil {
			return nil, err
		}
		fmt.Fprintf(f, "saved%d", s.saves)
		f.Close()
		st, _ := os.Stat(s.bog)
		os.Chtimes(s.bog, st.ModTime(), st.ModTime().Add(5*time.Second))
	}
	if s.onSave != nil {
		s.onSave()
	}
	return nil, nil
}

// srig is a write server whose station saves; home (when set) is the station home holding a
// config.bog, polled on a fake clock: sleeps counts the polls.
type srig struct {
	*trig
	st     *saveStation
	home   string
	now    float64
	sleeps int
}

func newSRig(t *testing.T, content string) *srig {
	t.Helper()
	tr := newTRig(t, 200)
	r := &srig{trig: tr, st: &saveStation{tstation: tr.st, rewrites: true}}
	tr.ctx.Session.Client = r.st
	tr.state.SaveTimeout, tr.state.SaveInterval = 1.0, 0.25
	tr.state.Monotonic = func() float64 { return r.now }
	tr.state.Sleep = func(s float64) { r.sleeps++; r.now += s }
	if content != "" {
		r.home = t.TempDir()
		r.st.bog = filepath.Join(r.home, "config.bog")
		if err := os.WriteFile(r.st.bog, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
		tr.state.StationHomes["FakeStation"] = r.home
	}
	return r
}

func (r *srig) save(t *testing.T) *pyjson.Object {
	t.Helper()
	text, isErr, out := r.write(t, "n4_save_station", obj())
	if isErr {
		t.Fatal(text)
	}
	return out
}

func TestPlanIsASaveOnTheRootHandleWithNoInverse(t *testing.T) {
	r := newSRig(t, "")
	text, isErr, out := r.call(t, "n4_save_station", obj())
	p, _ := get(out, "plan").(*pyjson.Object)
	if isErr || p == nil {
		t.Fatal(text)
	}
	if got := pyjson.Repr(get(p, "ops")); got != "[{'ssc': 'invokeAction', 'arg': {'h': '2', 'a': 'save'}}]" {
		t.Errorf("ops %s", got)
	}
	if got := pyjson.Repr(get(p, "inverse")); got != "[]" || r.st.saves != 0 {
		t.Errorf("inverse %s, %d saves", got, r.st.saves)
	}
	if got := pyjson.Repr(get(p, "notes")); got != "['the BOX reply to save is null and proves nothing; no --station-home "+
		"for this station, so persistence cannot be checked', 'no inverse: a save cannot be undone']" {
		t.Errorf("notes %s", got)
	}
}

func TestWithoutAStationHomePersistenceIsUnknownAndSaysHowToConfigure(t *testing.T) {
	r := newSRig(t, "")
	out := r.save(t)
	if get(out, "persisted") != "unknown" || get(out, "verdict") != "unverified" || r.st.saves != 1 {
		t.Fatal(pyjson.Repr(out))
	}
	if hint := get(get(out, "evidence").(*pyjson.Object), "hint"); hint != "configure --station-home FakeStation="+
		"<directory containing config.bog> to check persistence" {
		t.Errorf("hint %v", hint)
	}
}

func TestAChangedConfigBogProvesPersistenceWithEvidence(t *testing.T) {
	r := newSRig(t, "bog-before")
	sum := sha256.Sum256([]byte("bog-before"))
	out := r.save(t)
	if get(out, "persisted") != true || get(out, "verdict") != "verified" || r.sleeps != 0 {
		t.Fatalf("%s after %d sleeps", pyjson.Repr(out), r.sleeps)
	}
	ev := get(out, "evidence").(*pyjson.Object)
	before, after := get(ev, "before").(*pyjson.Object), get(ev, "after").(*pyjson.Object)
	if get(before, "sha256") != hex.EncodeToString(sum[:]) || get(after, "sha256") == get(before, "sha256") ||
		get(before, "size") != int64(10) || get(after, "size") != int64(16) {
		t.Errorf("evidence %s", pyjson.Repr(ev))
	}
	if !(get(after, "mtime").(float64) > get(before, "mtime").(float64)) || get(ev, "path") != r.st.bog {
		t.Errorf("evidence %s", pyjson.Repr(ev))
	}
	if got := pyjson.Repr(get(out, "accepted")); got != "None" || !strings.Contains(pyjson.Repr(out), "'inverse': []") {
		t.Errorf("out %s", pyjson.Repr(out))
	}
}

// TestAnUnchangedConfigBogIsPersistedFalseAfterThePoll: polled every SaveInterval until
// SaveTimeout has passed on the write state's clock (reads at 0, .25, .5, .75 and 1 s).
func TestAnUnchangedConfigBogIsPersistedFalseAfterThePoll(t *testing.T) {
	r := newSRig(t, "bog")
	r.st.rewrites = false
	out := r.save(t)
	if get(out, "persisted") != false || get(out, "verdict") != "mismatch" || r.sleeps != 4 || r.now != 1.0 {
		t.Fatalf("%s after %d sleeps at %v", pyjson.Repr(out), r.sleeps, r.now)
	}
	ev := get(out, "evidence").(*pyjson.Object)
	if pyjson.Repr(get(ev, "before")) != pyjson.Repr(get(ev, "after")) || ev.Len() != 3 {
		t.Errorf("evidence %s", pyjson.Repr(ev))
	}
}

func TestAMissingConfigBogIsUnknown(t *testing.T) {
	r := newSRig(t, "bog")
	os.Remove(r.st.bog)
	r.st.bog = ""
	out := r.save(t)
	want := "config.bog was not readable at " + filepath.Join(r.home, "config.bog") +
		" before the save; check that --station-home FakeStation points at the station directory"
	if get(out, "persisted") != "unknown" || get(out, "verdict") != "unverified" || r.sleeps != 0 ||
		pyjson.Repr(get(out, "evidence")) != pyjson.Repr(obj("hint", want)) {
		t.Fatal(pyjson.Repr(out))
	}
}

func TestTheReplyToSaveIsNeverTakenAsProof(t *testing.T) {
	r := newSRig(t, "bog")
	r.st.rewrites = false
	out := r.save(t)
	if get(out, "accepted") != nil || get(out, "persisted") == true || r.st.saves != 1 {
		t.Fatal(pyjson.Repr(out))
	}
}

func TestAConfigBogThatBecomesUnreadableAfterTheSaveIsUnverified(t *testing.T) {
	r := newSRig(t, "bog")
	r.st.onSave = func() { os.Remove(r.st.bog) }
	out := r.save(t)
	ev := get(out, "evidence").(*pyjson.Object)
	if get(out, "persisted") != "unknown" || get(out, "verdict") != "unverified" || r.sleeps != 0 ||
		get(ev, "hint") != "config.bog became unreadable after the save" || ev.Len() != 4 {
		t.Fatal(pyjson.Repr(out))
	}
	if after, ok := ev.Get("after"); !ok || after != nil || strings.Join(ev.Keys(), ",") != "path,before,after,hint" {
		t.Errorf("evidence %s", pyjson.Repr(ev))
	}
}

// A config.bog that changes only after a few polls is persisted; a mtime that moves without
// new bytes is enough, and the poll stops at the first change.
func TestAConfigBogChangedOnLaterPollsIsPersisted(t *testing.T) {
	r := newSRig(t, "bog")
	r.st.rewrites = false
	r.state.Sleep = func(s float64) {
		r.sleeps++
		r.now += s
		if r.sleeps == 2 {
			st, _ := os.Stat(r.st.bog)
			os.Chtimes(r.st.bog, st.ModTime(), st.ModTime().Add(time.Millisecond))
		}
	}
	out := r.save(t)
	ev := get(out, "evidence").(*pyjson.Object)
	if get(out, "persisted") != true || r.sleeps != 2 || get(get(ev, "after").(*pyjson.Object), "sha256") !=
		get(get(ev, "before").(*pyjson.Object), "sha256") {
		t.Fatalf("%s after %d sleeps", pyjson.Repr(out), r.sleeps)
	}
}

// The execution plans again: config.bog's state before the save is read then, not at the dry
// run, and it is not part of what the token authorizes. A home with a trailing slash joins as
// os.path.join does.
func TestTheStateBeforeTheSaveIsReadWhenItExecutes(t *testing.T) {
	r := newSRig(t, "bog")
	r.state.StationHomes["FakeStation"] = r.home + "/"
	_, _, dry := r.call(t, "n4_save_station", obj())
	if err := os.WriteFile(r.st.bog, []byte("edited"), 0o600); err != nil {
		t.Fatal(err)
	}
	text, isErr, out := r.call(t, "n4_save_station", obj("dry_run", false, "confirmation_token", get(dry, "confirmation_token")))
	if isErr {
		t.Fatal(text)
	}
	sum := sha256.Sum256([]byte("edited"))
	ev := get(out, "evidence").(*pyjson.Object)
	if get(get(ev, "before").(*pyjson.Object), "sha256") != hex.EncodeToString(sum[:]) || get(ev, "path") != r.home+"/config.bog" {
		t.Errorf("evidence %s", pyjson.Repr(ev))
	}
	notes := pyjson.Repr(get(get(dry, "plan").(*pyjson.Object), "notes"))
	if !strings.Contains(notes, "persistence is checked on "+r.home+"/config.bog'") {
		t.Errorf("notes %s", notes)
	}
}

// The intent and result of a save: no targets, no inverse, the reply and verdict.
func TestASaveIsJournaledWithoutTargetsOrInverse(t *testing.T) {
	r := newSRig(t, "bog")
	r.save(t)
	data, _ := os.ReadFile(filepath.Join(r.dir, "journal.jsonl"))
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	for i, want := range []string{`"inverse_plan": [], "station_name": "FakeStation", "phase": "intent", "targets": {}}`,
		`"phase": "result", "accepted": null, "inverse": [], "verdict": "verified"}`} {
		if len(lines) != 2 || !strings.HasSuffix(lines[i], want) {
			t.Errorf("journal %q, want line %d ending %s", lines, i, want)
		}
	}
}

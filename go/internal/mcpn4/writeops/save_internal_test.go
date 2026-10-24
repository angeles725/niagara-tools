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

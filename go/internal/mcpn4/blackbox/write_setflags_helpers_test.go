package blackbox

// The rig of the n4_set_flags tests: a component with one dynamic slot and a record of every
// sync op the station applied.

import (
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest/boxstation"
)

// flagsRig is a verified write rig whose Folder holds Calc with the slot note.
type flagsRig struct {
	*Rig
	calc Component
	ord  string
	ops  []map[string]any // guarded by the station lock
	// afterSync, when set, runs after each applied sync op with the station locked.
	afterSync func(st *boxstation.Station, op map[string]any)
}

// newFlagsRig starts the rig; note is a String "x" holding flags.
func newFlagsRig(t *testing.T, flags string) *flagsRig {
	f := &flagsRig{Rig: verifiedRig(t)}
	f.calc = f.AddWithOut("Calc", 1.0)
	f.ord = Folder + "/" + f.calc.Name
	f.Set(f.calc.Handle, "note", map[string]any{"nm": "p", "t": "baja:String", "v": "x"})
	f.setFlags(flags)
	f.Station.SetFaults(boxstation.Faults{OnSync: func(st *boxstation.Station, op map[string]any) {
		f.ops = append(f.ops, op)
		if f.afterSync != nil {
			f.afterSync(st, op)
		}
	}})
	return f
}

// args are n4_set_flags arguments for note.
func (f *flagsRig) args(flags string) Args { return Args{"ord": f.ord, "slot": "note", "flags": flags} }

// op is the set-flags op the server must send for note.
func (f *flagsRig) op(flags string) map[string]any {
	return map[string]any{"nm": "f", "h": f.calc.Handle, "n": "note", "f": flags}
}

// flags are the flags note holds on the station.
func (f *flagsRig) flags() (out string) {
	f.Station.Do(func(st *boxstation.Station) { out = st.ByHandle(f.calc.Handle).Child("note").Flags })
	return out
}

// setFlags changes note's flags on the station directly, as another operator would.
func (f *flagsRig) setFlags(flags string) {
	f.Station.Do(func(st *boxstation.Station) { st.ByHandle(f.calc.Handle).Child("note").Flags = flags })
}

// sent are the sync ops the station applied so far.
func (f *flagsRig) sent() (out []any) {
	f.Station.Do(func(*boxstation.Station) {
		for _, op := range f.ops {
			out = append(out, op)
		}
	})
	return out
}

// untouched fails the test unless the station applied no op, note holds flags and the journal
// has no line.
func (f *flagsRig) untouched(t *testing.T, flags string) {
	t.Helper()
	if got := f.sent(); len(got) != 0 {
		t.Errorf("the station applied %v", got)
	}
	if got := f.flags(); got != flags {
		t.Errorf("flags = %q, want %q", got, flags)
	}
	if lines := f.Lines("journal.jsonl"); len(lines) != 0 {
		t.Errorf("journal = %v", lines)
	}
}

// noTypes makes the station's registry channel describe no type: every other frame is answered
// as usual.
func (f *flagsRig) noTypes() {
	f.Station.SetHook(func(frame map[string]any) (int, []byte) {
		msgs, _ := frame["m"].([]any)
		for _, m := range objects(msgs) {
			if m["c"] == "reg" {
				return 200, []byte(`{"v": "2.3", "p": "box", "m": [{"c": "reg", "k": "loadTypes", "r": 0, "t": "rp", "b": {}}]}`)
			}
		}
		return 0, nil
	})
}

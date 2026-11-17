package blackbox

// n4_inventory_snapshot and n4_inventory_diff (station-audits S8) on the fake station.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest/boxstation"
)

// inventoryRig is a connected rig whose server keeps its exports under the rig's state dir.
func inventoryRig(t *testing.T) *Rig {
	r := NewRig(t, func(r *Rig, o *Options) { o.StateDir = r.StateDir })
	r.Connect(nil)
	return r
}

// addProp adds a dynamic simple property to the component with handle h.
func (r *Rig) addProp(h, name, typ, value, flags string) {
	r.T.Helper()
	if _, err := r.Station.AddSlot(h, name, map[string]any{"nm": "p", "t": typ, "v": value}, flags); err != nil {
		r.T.Fatal(err)
	}
}

// snapshotName takes an inventory snapshot of the Folder and returns its name.
func (r *Rig) snapshotName() string {
	r.T.Helper()
	out := r.Srv.OK("n4_inventory_snapshot", Args{"base": Folder})
	name, _ := out["name"].(string)
	if !strings.HasPrefix(name, "inventory-FakeStation-") || out["truncated"] != false {
		r.T.Fatalf("snapshot = %v", out)
	}
	return name
}

func asJSONText(t *testing.T, v any) string {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestInventoryDiffShowsExactlyTheSlotTheAddedAndTheRemovedComponent(t *testing.T) {
	r := inventoryRig(t)
	pump := r.Add("3", "Pump", "")
	r.addProp(pump.Handle, "setpoint", "baja:Double", "20.0", "")
	r.Add("3", "Old", "baja:Folder")
	a := r.snapshotName()
	r.Set(pump.Handle, "setpoint", map[string]any{"nm": "p", "t": "baja:Double", "v": "21.5"})
	r.Add("3", "New", "baja:Folder")
	r.Station.Do(func(st *boxstation.Station) {
		f := st.ByHandle("3")
		kids := f.Children[:0]
		for _, c := range f.Children {
			if c.Name != "Old" {
				kids = append(kids, c)
			}
		}
		f.Children = kids
	})
	b := r.snapshotName()
	out := r.Srv.OK("n4_inventory_diff", Args{"a": a, "b": b})
	for key, want := range map[string]string{
		"added":          `[{"path":"slot:/Folder/New","type":"baja:Folder"}]`,
		"removed":        `[{"path":"slot:/Folder/Old","type":"baja:Folder"}]`,
		"changed":        `[{"after":"21.5","before":"20.0","path":"slot:/Folder/Pump","slot":"setpoint"}]`,
		"type_changed":   `[]`,
		"handle_changed": `[]`,
		"counts":         `{"added":1,"changed":1,"handle_changed":0,"removed":1,"type_changed":0,"unchanged":1}`,
		"truncated":      `false`,
	} {
		if got := asJSONText(t, out[key]); got != want {
			t.Errorf("%s = %s, want %s", key, got, want)
		}
	}
}

func TestInventorySnapshotsOfAnUnchangedStationDiffToEmpty(t *testing.T) {
	r := inventoryRig(t)
	r.Station.SetRegistry(map[string]any{"kitBench:Ticker": map[string]any{"c": []any{
		map[string]any{"st": "p", "n": "ticks", "f": "t"}}}})
	calc := r.AddWithOut("Calc", 1)
	tick := r.Add("3", "Tick", "kitBench:Ticker")
	r.addProp(tick.Handle, "ticks", "baja:Long", "1", "")          // declared transient
	r.addProp(calc.Handle, "elapsed", "baja:RelTime", "5", "t")    // transient on the instance
	r.addProp(calc.Handle, "facets", "baja:Facets", "units=C", "") // configuration, unchanged
	r.addProp(calc.Handle, "health", "baja:Status", "0", "")       // status bits: runtime
	a := r.snapshotName()
	r.Set(calc.Handle, "out", StatusNumeric(7, "2"))
	r.Set(tick.Handle, "ticks", map[string]any{"nm": "p", "t": "baja:Long", "v": "2"})
	r.Set(calc.Handle, "health", map[string]any{"nm": "p", "t": "baja:Status", "v": "2"})
	r.Set(calc.Handle, "elapsed", map[string]any{"nm": "p", "t": "baja:RelTime", "v": "6"})
	b := r.snapshotName()
	out := r.Srv.OK("n4_inventory_diff", Args{"a": a, "b": b})
	if got := asJSONText(t, out["counts"]); got != `{"added":0,"changed":0,"handle_changed":0,"removed":0,"type_changed":0,"unchanged":3}` {
		t.Errorf("counts = %s (changed %v)", got, out["changed"])
	}
	data, err := os.ReadFile(filepath.Join(r.StateDir, "exports", a))
	if err != nil || !strings.Contains(string(data), `"facets": "units=C"`) || strings.Contains(string(data), `"elapsed"`) {
		t.Errorf("snapshot file %s: %v", data, err)
	}
	if info, _ := os.Stat(filepath.Join(r.StateDir, "exports", a)); info == nil || info.Mode().Perm() != 0o600 {
		t.Errorf("snapshot file mode %v", info)
	}
}

func TestInventoryDiffRefusesAPathOrAMissingName(t *testing.T) {
	r := inventoryRig(t)
	a := r.snapshotName()
	for _, bad := range []string{"../" + a, "exports/" + a, "..json", "a..b.json"} {
		if text := r.Srv.Err("n4_inventory_diff", Args{"a": a, "b": bad}); !strings.Contains(text, "no '..' and no path separator") {
			t.Errorf("b=%q: %s", bad, text)
		}
	}
	if text := r.Srv.Err("n4_inventory_diff", Args{"a": "inventory-none.json", "b": a}); !strings.Contains(text, "no inventory snapshot") {
		t.Errorf("missing: %s", text)
	}
}

func TestInventorySlotsArgumentRecordsTheChosenSlotsOnly(t *testing.T) {
	r := inventoryRig(t)
	calc := r.AddWithOut("Calc", 1)
	r.addProp(calc.Handle, "facets", "baja:Facets", "units=C", "")
	out := r.Srv.OK("n4_inventory_snapshot", Args{"base": Folder, "slots": []any{"out"}})
	data, err := os.ReadFile(out["path"].(string))
	if err != nil || !strings.Contains(string(data), `"out": "1.0 {0}"`) || strings.Contains(string(data), "facets") {
		t.Errorf("snapshot file %s: %v", data, err)
	}
}

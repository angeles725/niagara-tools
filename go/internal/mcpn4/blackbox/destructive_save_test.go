package blackbox

// n4_save_station and its proof on disk (port of test_tools_destructive.TestSaveStation and
// TestWriteBranchDebt.test_a_config_bog_that_becomes_unreadable_after_the_save_is_unverified).

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest/boxstation"
)

// homeServer writes config.bog holding content into a fresh station home, makes the station
// rewrite it on save when rewrite is set, and restarts the server with --station-home; it
// returns the file's path.
func (d *dRig) homeServer(content string, rewrite bool) string {
	d.T.Helper()
	dir := d.T.TempDir()
	path := filepath.Join(dir, "config.bog")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		d.T.Fatal(err)
	}
	if rewrite {
		d.Station.Do(func(st *boxstation.Station) { st.ConfigPath = path })
	}
	d.restart(func(o *Options) { o.StationHomes = map[string]string{StationName: dir} })
	return path
}

func TestPlanIsASaveOnTheRootHandleWithNoInverse(t *testing.T) {
	d := destructiveRig(t, nil)
	plan := planOf(d.Srv.Dry("n4_save_station", nil))
	if !same(plan["ops"], []any{map[string]any{"ssc": "invokeAction", "arg": map[string]any{"h": "2", "a": "save"}}}) {
		t.Errorf("ops = %v", plan["ops"])
	}
	if !same(plan["inverse"], []any{}) || d.Station.Saves() != 0 {
		t.Errorf("inverse %v, saves %d", plan["inverse"], d.Station.Saves())
	}
}

func TestWithoutAStationHomePersistenceIsUnknownAndSaysHowToConfigure(t *testing.T) {
	d := destructiveRig(t, nil)
	out := d.Srv.RunWrite("n4_save_station", nil)
	hint, _ := Dig(out, "evidence", "hint").(string)
	if out["persisted"] != "unknown" || out["verdict"] != "unverified" || !strings.Contains(hint, "--station-home FakeStation=") {
		t.Errorf("out = %v", out)
	}
	if d.Station.Saves() != 1 {
		t.Errorf("saves = %d", d.Station.Saves())
	}
}

func TestAChangedConfigBogProvesPersistenceWithEvidence(t *testing.T) {
	d := destructiveRig(t, nil)
	path := d.homeServer("bog-before", true)
	sum := sha256.Sum256([]byte("bog-before"))
	before := hex.EncodeToString(sum[:])
	out := d.Srv.RunWrite("n4_save_station", nil)
	if out["persisted"] != true || out["verdict"] != "verified" {
		t.Errorf("out = %v", out)
	}
	ev, _ := out["evidence"].(map[string]any)
	if Dig(ev, "before", "sha256") != before || Dig(ev, "after", "sha256") == before || ev["path"] != path {
		t.Errorf("evidence = %v", ev)
	}
	if b, a := Dig(ev, "before", "mtime"), Dig(ev, "after", "mtime"); !(isNumber(a) && isNumber(b) && a.(float64) > b.(float64)) {
		t.Errorf("mtimes %v -> %v", b, a)
	}
}

func isNumber(v any) bool { _, ok := v.(float64); return ok }

func TestAMissingConfigBogIsUnknown(t *testing.T) {
	d := destructiveRig(t, nil)
	path := d.homeServer("bog", true)
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	d.f.set(func(f *faults) {
		f.before = func(sck string) { // the kit's fake opened config.bog for append, creating it
			if sck == "invokeAction" {
				if fh, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0o600); err == nil {
					fh.Close()
				}
			}
		}
	})
	out := d.Srv.RunWrite("n4_save_station", nil)
	if hint, _ := Dig(out, "evidence", "hint").(string); out["persisted"] != "unknown" || !strings.Contains(hint, "config.bog") {
		t.Errorf("out = %v", out)
	}
}

func TestAWriteScopeIsStillRequired(t *testing.T) {
	d := destructiveRig(t, nil)
	d.restart(func(o *Options) { o.WriteScopes = nil })
	if text := d.Srv.Err("n4_save_station", nil); !strings.Contains(text, "no --write-scope") {
		t.Errorf("text = %s", text)
	}
	if d.Station.Saves() != 0 {
		t.Errorf("saves = %d", d.Station.Saves())
	}
}

func TestAConfigBogThatBecomesUnreadableAfterTheSaveIsUnverified(t *testing.T) {
	d := destructiveRig(t, nil)
	path := d.homeServer("bog", false)
	d.Station.Do(func(st *boxstation.Station) {
		st.AfterInvoke = func(_ *boxstation.Station, action string, _ *boxstation.Node) {
			if action == "save" {
				_ = os.Remove(path)
			}
		}
	})
	out := d.Srv.RunWrite("n4_save_station", nil)
	hint, _ := Dig(out, "evidence", "hint").(string)
	if out["persisted"] != "unknown" || out["verdict"] != "unverified" || !strings.Contains(hint, "unreadable after the save") {
		t.Errorf("out = %v", out)
	}
}

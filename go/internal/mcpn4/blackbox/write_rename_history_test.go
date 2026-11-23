package blackbox

// n4_rename_component lists the history extensions whose id resolves through a renamed component
// and, with freeze_history_names, sets each historyName to its current text in the same batch, so
// the trends do not split (retro R12 ΔM5).

import (
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest/boxstation"
)

// renameRig is a history rig whose Folder holds NW with the extension hist (historyName
// %parent.name%) and Plain with none; the station lists the extension and one of another folder.
func renameRig(t *testing.T) *historyRig {
	h := newHistoryRig(t, "Slot Path,Id\nslot:/Folder/NW/hist,/FakeStation/NW\nslot:/Folder/Elsewhere/hist,/FakeStation/Elsewhere\n")
	nw := h.Add("3", "NW", "baja:Folder")
	ext := h.Add(nw.Handle, "hist", "history:NumericIntervalHistoryExt")
	h.Set(ext.Handle, "historyName", map[string]any{"nm": "p", "t": "baja:Format", "v": "%parent.name%"})
	h.Add("3", "Plain", "baja:Folder")
	return h
}

func renameArgs(name string, extra ...any) Args {
	a := Args{"items": []any{map[string]any{"parent_ord": Folder, "name": name, "new_name": "North"}}}
	for i := 0; i < len(extra); i += 2 {
		a[extra[i].(string)] = extra[i+1]
	}
	return a
}

func TestRenameListsTheHistoryIdsItChanges(t *testing.T) {
	h := renameRig(t)
	notes, _ := planOf(h.Srv.Dry("n4_rename_component", renameArgs("NW")))["notes"].([]any)
	want := "History ids that change with this rename (1): /FakeStation/NW -> /FakeStation/North " +
		"(/Folder/NW/hist, historyName %parent.name%). Trends split at the rename; freeze_history_names: true " +
		"sets each historyName to its current text in the same batch."
	if len(notes) != 3 || notes[2] != want {
		t.Errorf("notes %q, want the last to be %q", notes, want)
	}
}

func TestRenameWithFreezeHistoryNamesSetsEachNameInTheSameBatch(t *testing.T) {
	h := renameRig(t)
	args := renameArgs("NW", "freeze_history_names", true)
	dry := h.Srv.Dry("n4_rename_component", args)
	ops := objects(planOf(dry)["ops"])
	if len(ops) != 2 || ops[0]["nm"] != "s" || ops[0]["n"] != "historyName" || ops[1]["nm"] != "r" ||
		jsonText(ops[0]["b"]) != `{"nm":"p","t":"baja:Format","v":"NW"}` {
		t.Fatalf("ops %v", planOf(dry)["ops"])
	}
	if notes, _ := planOf(dry)["notes"].([]any); len(notes) != 3 ||
		notes[2] != "History names frozen to their current text in this batch (1): /Folder/NW/hist = NW." {
		t.Errorf("notes %q", notes)
	}
	done := h.Srv.OK("n4_rename_component", Execute(args, dry["confirmation_token"]))
	if done["verdict"] != "verified" || len(objects(done["inverse"])) != 2 {
		t.Errorf("done %v", done)
	}
	var got any
	h.Station.Do(func(st *boxstation.Station) {
		got = st.ByHandle("3").Child("North").Child("hist").Child("historyName").Value
	})
	if got != "NW" {
		t.Errorf("historyName after the rename = %v, want NW", got)
	}
}

func TestRenameWithoutHistoryExtensionsKeepsItsNotes(t *testing.T) {
	h := renameRig(t)
	got, _ := planOf(h.Srv.Dry("n4_rename_component", renameArgs("Plain")))["notes"].([]any)
	if len(got) != 2 || !strings.HasPrefix(got[1].(string), "Only dynamic slots can be renamed") {
		t.Errorf("notes %v", got)
	}
}

func TestRenameSaysWhenTheHistoryExtensionsCouldNotBeRead(t *testing.T) {
	h := renameRig(t)
	routeOrds(h.sv) // the station answers no history table
	notes, _ := planOf(h.Srv.Dry("n4_rename_component", renameArgs("NW")))["notes"].([]any)
	if len(notes) != 3 || !strings.HasPrefix(notes[2].(string), "The history extensions below the renamed "+
		"components could not be read, so the history ids this rename changes are not listed.") {
		t.Errorf("notes %q", notes)
	}
	refusedDry(t, h.Rig, "n4_rename_component", renameArgs("NW", "freeze_history_names", true),
		"freeze_history_names needs the station's history extensions")
}

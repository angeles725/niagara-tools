package blackbox

// The dry runs of n4_create_component and n4_create_link refuse what they can check before a
// token is issued (bench retro 2026-11-15 delta 4): nothing is journaled.

import (
	"strings"
	"testing"
)

// numericConst is a registry entry declaring the in10 property and the out property.
var numericConst = map[string]any{"c": []any{map[string]any{"st": "p", "n": "out"},
	map[string]any{"st": "p", "n": "in10"}}}

// refusedDry is a dry run that must answer a tool error naming every want, with nothing
// journaled and no token issued.
func refusedDry(t *testing.T, r *Rig, tool string, args Args, want ...string) {
	t.Helper()
	text := r.Srv.Err(tool, args)
	for _, w := range want {
		if !strings.Contains(text, w) {
			t.Errorf("reply %q lacks %q", text, w)
		}
	}
	if lines := r.Lines("journal.jsonl"); len(lines) != 0 {
		t.Errorf("journal = %v", lines)
	}
}

func TestCreateOfAnUnknownTypeIsRefusedByItsName(t *testing.T) {
	r := verifiedRig(t)
	r.Station.SetRegistry(map[string]any{"kitControl:Nope": nil, "modbusAsync:NoExt": nil})
	refusedDry(t, r, "n4_create_component", withArgs(createPump, Args{"type": "kitControl:Nope"}),
		"kitControl:Nope", "not known to the station")
	refusedDry(t, r, "n4_create_component", withArgs(createPump, Args{"proxy_ext": "modbusAsync:NoExt"}),
		"modbusAsync:NoExt", "not known to the station")
	if len(r.Children()) != 0 {
		t.Errorf("children %v", r.Children())
	}
}

// A 4.15.6.20 station answers loadTypes for a type it does not know with an error entry
// ({"isErr": true, "m": spec, "c": "", "s": ""}, seen live on the lab station), not by leaving it out.
func TestCreateOfATypeTheStationAnswersAsAnErrorIsRefusedByItsName(t *testing.T) {
	r := verifiedRig(t)
	r.Station.SetRegistry(map[string]any{"kitControl:NoSuchType": map[string]any{"c": "", "isErr": true, "m": "kitControl:NoSuchType", "s": ""}})
	refusedDry(t, r, "n4_create_component", withArgs(createPump, Args{"type": "kitControl:NoSuchType"}),
		"kitControl:NoSuchType", "not known to the station")
	if len(r.Children()) != 0 || len(r.Lines("journal.jsonl")) != 0 {
		t.Errorf("children %v journal %d", r.Children(), len(r.Lines("journal.jsonl")))
	}
}

// A type the station's registry does not describe may still be loaded: a lab station whose module
// jar was removed and re-added under the same name kept its registry "Up-to-date" without the
// module's types (client A retro 2026-11-16 rows M12, M14). The refusal names that route.
func TestAnUnknownTypeRefusalNamesTheStaleRegistryRoute(t *testing.T) {
	r := verifiedRig(t)
	r.Station.SetRegistry(map[string]any{"vendorX:Pump": map[string]any{"c": "", "isErr": true, "m": "vendorX:Pump", "s": ""}})
	refusedDry(t, r, "n4_create_component", withArgs(createPump, Args{"type": "vendorX:Pump"}),
		"vendorX:Pump", "not known to the station", "registry may be stale", "touch the module jar",
		"Rebuilding registry", "reconnect with n4_connect")
}

func TestLinkToAMissingSlotIsRefused(t *testing.T) {
	r, _, _, args := linkRig(t)
	r.Station.SetRegistry(map[string]any{"kitControl:NumericConst": numericConst})
	refusedDry(t, r, "n4_create_link", withArgs(args, Args{"source_slot": "nope"}), "source slot", "nope", "does not exist")
	refusedDry(t, r, "n4_create_link", withArgs(args, Args{"target_slot": "in99"}), "target slot", "in99", "does not exist")
	r.Srv.Dry("n4_create_link", args) // in10 is declared by the type, out is on the component
}

func TestAnIdenticalLinkIsRefusedButAnotherSlotIsNot(t *testing.T) {
	r, src, tgt, args := linkRig(t)
	name := r.Link(src, "out", tgt, "in10")
	refusedDry(t, r, "n4_create_link", args, "identical link already exists", name)
	r.Srv.Dry("n4_create_link", withArgs(args, Args{"target_slot": "in11"}))
}

// A name already taken under the parent is not refused (the station assigns another one and the
// read-back and inverse follow it): the dry run says so in its first note.
func TestCreateOfATakenNameWarnsFirstInTheDryRun(t *testing.T) {
	r := verifiedRig(t)
	r.Add("3", "Pump", "")
	notes, _ := planOf(r.Srv.Dry("n4_create_component", createPump))["notes"].([]any)
	want := "name 'Pump' is already taken under " + Folder + ": the station will assign another name; the read-back and the inverse follow it"
	if len(notes) == 0 || notes[0] != want {
		t.Errorf("notes = %v", notes)
	}
	free := verifiedRig(t)
	if notes, _ := planOf(free.Srv.Dry("n4_create_component", createPump))["notes"].([]any); len(notes) != 1 {
		t.Errorf("free name notes = %v", notes)
	}
}

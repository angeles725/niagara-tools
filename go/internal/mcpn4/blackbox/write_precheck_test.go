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

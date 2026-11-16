package blackbox

// Write-scope rules added after the 2026-11-13 client A retro: the scope may create its own root
// (M5) and a link is scoped on its target only (M10).

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"
)

const scopeRoot = Folder + "/Root"

func rootScopeRig(t *testing.T) *Rig {
	r := writeRig(t, func(o *Options) { o.WriteScopes = []string{scopeRoot} })
	r.ConnectVerified()
	return r
}

func TestScopeRootCanBeCreatedUnderItsParent(t *testing.T) {
	r := rootScopeRig(t)
	r.Srv.RunWrite("n4_create_component", Args{"parent_ord": Folder, "name": "Root", "type": "baja:Folder"})
	if !slices.Contains(r.Children(), "Root") {
		t.Errorf("children = %v, want Root created", r.Children())
	}
}

func TestScopeRootThatExistsIsNotCreatedAgain(t *testing.T) {
	r := rootScopeRig(t)
	r.Add("3", "Root", "baja:Folder")
	text := r.Srv.Err("n4_create_component", Args{"parent_ord": Folder, "name": "Root", "type": "baja:Folder"})
	if !strings.Contains(text, "already exists") || !strings.Contains(text, scopeRoot) {
		t.Errorf("refusal = %s", text)
	}
	if got := r.Children(); len(got) != 1 {
		t.Errorf("children = %v, want only the existing Root (no renamed copy outside the scope)", got)
	}
}

func TestOnlyTheScopeRootItselfIsCreatableUnderItsParent(t *testing.T) {
	r := rootScopeRig(t)
	for _, name := range []string{"Other", "Roo", "RootX"} {
		text := r.Srv.Err("n4_create_component", Args{"parent_ord": Folder, "name": name, "type": "baja:Folder"})
		if !strings.Contains(text, "outside the write scope") {
			t.Errorf("%s: refusal = %s", name, text)
		}
	}
	if len(r.Children()) != 0 {
		t.Errorf("children = %v, want none", r.Children())
	}
}

func TestLinkFromASourceOutsideTheScopeIsPlanned(t *testing.T) {
	r := writeRig(t, func(o *Options) { o.WriteScopes = []string{Folder + "/B"} })
	r.ConnectVerified()
	r.AddWithOut("A", 1)
	r.Add("3", "B", "")
	r.Srv.Dry("n4_create_link", Args{"source_ord": Folder + "/A", "source_slot": "out",
		"target_ord": Folder + "/B", "target_slot": "in10"})
}

func TestLinkRefusalNamesTheTargetEnd(t *testing.T) {
	r := writeRig(t, func(o *Options) { o.WriteScopes = []string{Folder + "/A"} })
	r.ConnectVerified()
	r.AddWithOut("A", 1)
	r.Add("3", "B", "")
	text := r.Srv.Err("n4_create_link", Args{"source_ord": Folder + "/A", "source_slot": "out",
		"target_ord": Folder + "/B", "target_slot": "in10"})
	if !strings.Contains(text, "target_ord") || !strings.Contains(text, "outside the write scope") {
		t.Errorf("refusal = %s", text)
	}
}

func TestLinkSourceMustStillBeAPlainOrd(t *testing.T) {
	r := verifiedRig(t)
	r.AddWithOut("A", 1)
	text := r.Srv.Err("n4_create_link", Args{"source_ord": Folder + "/A|slot:../X", "source_slot": "out",
		"target_ord": Folder + "/A", "target_slot": "in10"})
	if !strings.Contains(text, "source_ord") || !strings.Contains(text, "is not a plain station ORD") {
		t.Errorf("refusal = %s", text)
	}
}

// The dry run of a scope-root create tells the operator how the root goes away again.
func TestScopeRootCreatePlanSaysHowToRemoveIt(t *testing.T) {
	r := rootScopeRig(t)
	out := r.Srv.Dry("n4_create_component", Args{"parent_ord": Folder, "name": "Root", "type": "baja:Folder"})
	notes := pyjsonText(out["plan"])
	if !strings.Contains(notes, "creates the write-scope root") || !strings.Contains(notes, "wider scope or by hand") {
		t.Errorf("plan = %s", notes)
	}
	plain := verifiedRig(t).Srv.Dry("n4_create_component", createPump)
	if strings.Contains(pyjsonText(plain["plan"]), "write-scope root") {
		t.Errorf("an ordinary create carries the root note: %v", plain["plan"])
	}
}

func pyjsonText(v any) string {
	raw, _ := json.Marshal(v)
	return string(raw)
}

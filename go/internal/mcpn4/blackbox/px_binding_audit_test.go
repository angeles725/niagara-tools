package blackbox

// n4_px_binding_audit (station-audits S15): the Px bindings whose ORD does not resolve, read-only,
// on a fake station whose Folder holds Good (with out) and nothing named Gone.

import (
	"reflect"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest/boxstation"
)

const pbaPx = `<px version="1.0" media="workbench:WbPxMedia"><content><CanvasPane name="c">
<BoundLabel name="good"><ValueBinding ord="station:|slot:/Folder/Good"/></BoundLabel>
<BoundLabel name="gone"><ValueBinding ord="station:|slot:/Folder/Gone"/></BoundLabel>
<WebChart name="trend"><ValueBinding ord="history:/demo/T"/></WebChart>
<BoundLabel name="rel"><ValueBinding ord="slot:Good/out"/></BoundLabel>
<ActionButton name="relGone"><ActionBinding ord="slot:Gone2"/></ActionButton>
<BoundLabel name="again"><ValueBinding ord="station:|slot:/Folder/Good"/></BoundLabel>
<BoundLabel name="deep"><ValueBinding ord="station:|slot:/Nowhere/Deep/X"/></BoundLabel>
</CanvasPane></content></px>`

// pbaRig serves px/Audit.px as the view on /Folder and px/Lost.px (absent) as a view on the root.
func pbaRig(t *testing.T) (*Rig, *boxstation.Servlets) {
	t.Helper()
	r, sv := servletRig(t, nil)
	r.AddWithOut("Good", 1)
	serveFile(sv, "px/Audit.px", []byte(pbaPx), "text/xml")
	routeOrds(sv, boxstation.OrdRoute{Needle: "baja:PxView", Body: []byte("Slot Path,Display Name,Px File,Media\n" +
		"slot:/Folder/graphic,Audit,file:^px/Audit.px,workbench:WbPxMedia\n" +
		"NULL,Lost,file:^px/Lost.px,workbench:WbPxMedia\n")})
	return r, sv
}

func pbaOrds(list any) []string {
	out := []string{}
	for _, e := range objects(list) {
		out = append(out, e["ord"].(string))
	}
	return out
}

func TestTheAuditReportsOnlyDefiniteMissesAsBroken(t *testing.T) {
	r, _ := pbaRig(t)
	out := r.Srv.OK("n4_px_binding_audit", nil)
	wantStrings(t, "broken", pbaOrds(out["broken"]),
		[]string{"station:|slot:/Folder/Gone", "slot:Gone2", "station:|slot:/Nowhere/Deep/X"})
	wantStrings(t, "unknown", pbaOrds(out["unknown"]), []string{"history:/demo/T"})
	gone := objects(out["broken"])[0]
	want := map[string]any{"px_file": "file:^px/Audit.px", "widget": "gone", "element": "ValueBinding",
		"ord": "station:|slot:/Folder/Gone", "target": "station:|slot:/Folder/Gone",
		"why": "station:|slot:/Folder holds no slot Gone"}
	if !reflect.DeepEqual(gone, want) {
		t.Errorf("broken[0] = %v", gone)
	}
	if rel := objects(out["broken"])[1]; rel["target"] != "station:|slot:/Folder/Gone2" {
		t.Errorf("a relative ORD was not read against the view's base: %v", rel)
	}
	if why := objects(out["broken"])[2]["why"]; why != "station:|slot:/ holds no slot Nowhere" {
		t.Errorf("deep why = %v", why)
	}
	counts := map[string]any{"views": 2.0, "bindings": 7.0, "unique_targets": 5.0, "resolves": 3.0, "missing": 3.0,
		"unknown": 1.0}
	if !reflect.DeepEqual(out["counts"], counts) {
		t.Errorf("counts = %v", out["counts"])
	}
	views := objects(out["views"])
	if views[0]["status"] != "ok" || views[0]["base"] != "station:|slot:/Folder" || views[0]["bindings"] != 7.0 ||
		views[1]["status"] != "not_found" {
		t.Errorf("views = %v", views)
	}
	if out["truncated"] != false {
		t.Errorf("truncated = %v", out["truncated"])
	}
}

func TestOneFileWithoutABaseLeavesRelativeOrdsUnknown(t *testing.T) {
	r, _ := pbaRig(t)
	out := r.Srv.OK("n4_px_binding_audit", Args{"file_ord": "px/Audit.px"})
	wantStrings(t, "broken", pbaOrds(out["broken"]), []string{"station:|slot:/Folder/Gone", "station:|slot:/Nowhere/Deep/X"})
	wantStrings(t, "unknown", pbaOrds(out["unknown"]), []string{"history:/demo/T", "slot:Good/out", "slot:Gone2"})
	if why, _ := objects(out["unknown"])[1]["why"].(string); !strings.Contains(why, "base") {
		t.Errorf("relative why = %q", why)
	}
	based := r.Srv.OK("n4_px_binding_audit", Args{"file_ord": "px/Audit.px", "base": "station:|slot:/Folder"})
	wantStrings(t, "broken with base", pbaOrds(based["broken"]),
		[]string{"station:|slot:/Folder/Gone", "slot:Gone2", "station:|slot:/Nowhere/Deep/X"})
}

func TestTheBindingCapTruncatesAndABadBaseIsRefused(t *testing.T) {
	r, _ := pbaRig(t)
	out := r.Srv.OK("n4_px_binding_audit", Args{"file_ord": "px/Audit.px", "max_bindings": 2})
	if out["truncated"] != true || Dig(out, "counts", "bindings") != 2.0 {
		t.Errorf("truncated = %v, counts = %v", out["truncated"], out["counts"])
	}
	if text := r.Srv.Err("n4_px_binding_audit", Args{"file_ord": "px/Audit.px", "base": "station:|slot:/A|bql:x"}); !strings.Contains(text, "base") {
		t.Errorf("refusal = %q", text)
	}
}

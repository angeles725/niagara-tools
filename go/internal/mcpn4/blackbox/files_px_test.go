package blackbox

// n4_list_px_views and n4_px_summary (port of mcp-n4-kit/tests/test_files.py,
// TestListPxViewsTool and TestPxSummaryTool).

import (
	"reflect"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest/boxstation"
)

var pxViewRows = []byte("Slot Path,Display Name,Px File,Media\n" +
	"NULL,General,file:^px/General.px,workbench:WbPxMedia\n" +
	"slot:/Plant/AHU1/graphic,AHU Graphic,file:^px/AHU.px,workbench:WbPxMedia\n")

func pxViewRig(t *testing.T) (*Rig, *boxstation.Servlets) {
	r, sv := fileRig(t, nil)
	routeOrds(sv, boxstation.OrdRoute{Needle: "baja:PxView", Body: pxViewRows})
	return r, sv
}

func TestPxViewsAreListedWithTheirFileAndMedia(t *testing.T) {
	r, sv := pxViewRig(t)
	out := r.Srv.OK("n4_list_px_views", nil)
	views := objects(out["views"])
	if out["row_count"] != 2.0 || len(views) != 2 {
		t.Fatalf("row_count = %v, views = %v", out["row_count"], out["views"])
	}
	want := map[string]any{"display_name": "General", "px_file": "file:^px/General.px",
		"media": "workbench:WbPxMedia", "slot_path": nil, "owner_path": nil}
	if !reflect.DeepEqual(views[0], want) {
		t.Errorf("first view = %v", views[0])
	}
	if views[1]["owner_path"] != "station:|slot:/Plant/AHU1" {
		t.Errorf("second owner_path = %v", views[1]["owner_path"])
	}
	if note, _ := out["owner_note"].(string); !strings.Contains(note, "NULL") {
		t.Errorf("owner_note = %v", out["owner_note"])
	}
	ords := ordRequests(sv)
	if len(ords) == 0 {
		t.Fatal("no ORD was requested")
	}
	if !strings.HasPrefix(ords[0], "station:|slot:/|bql:select slotPath, displayName, pxFile, media from baja:PxView") ||
		!strings.HasSuffix(ords[0], "|view:file:ITableToCsv") {
		t.Errorf("ORD = %q", ords[0])
	}
}

func TestTheRootScopesTheQueryAndAPipeIsRefused(t *testing.T) {
	r, sv := pxViewRig(t)
	r.Srv.OK("n4_list_px_views", Args{"root": "station:|slot:/Plant"})
	if ords := ordRequests(sv); len(ords) == 0 || !strings.HasPrefix(ords[0], "station:|slot:/Plant|bql:") {
		t.Errorf("ORDs = %q", ords)
	}
	r.Srv.Err("n4_list_px_views", Args{"root": "/Plant|bql:select x from y"})
	if n := len(ordRequests(sv)); n != 1 {
		t.Errorf("%d ORD requests, want 1", n)
	}
}

func TestTheSummaryParsesTheFetchedPx(t *testing.T) {
	r, _ := fileRig(t, nil)
	out := r.Srv.OK("n4_px_summary", Args{"file_ord": "file:^px/General.px"})
	if out["status"] != "ok" || Dig(out, "summary", "media") != "workbench:WbPxMedia" {
		t.Errorf("status/media = %v %v", out["status"], Dig(out, "summary", "media"))
	}
	if got := Dig(out, "summary", "included_px"); !reflect.DeepEqual(got, []any{"file:^px/includes/Header.px"}) {
		t.Errorf("included_px = %v", got)
	}
	if _, has := out["text"]; has {
		t.Error("the summary carries the text")
	}
}

func TestAMissingPxIsATypedResult(t *testing.T) {
	r, _ := fileRig(t, nil)
	out := r.Srv.OK("n4_px_summary", Args{"file_ord": "px/Nope.px"})
	if out["status"] != "not_found" {
		t.Errorf("status = %v", out["status"])
	}
	if _, has := out["summary"]; has {
		t.Error("a missing Px has a summary")
	}
}

func TestATruncatedPxIsRefusedWithTheCapNamed(t *testing.T) {
	r, _ := fileRig(t, nil)
	if text := r.Srv.Err("n4_px_summary", Args{"file_ord": "px/General.px", "max_bytes": 1024}); !strings.Contains(text, "max_bytes") {
		t.Errorf("refusal = %q", text)
	}
}

func TestAFileThatIsNotPxXMLIsRefused(t *testing.T) {
	r, sv := fileRig(t, nil)
	serveFile(sv, "px/Bad.px", []byte("<html/>"), "text/xml")
	if text := r.Srv.Err("n4_px_summary", Args{"file_ord": "px/Bad.px"}); !strings.Contains(text, "px") {
		t.Errorf("refusal = %q", text)
	}
}

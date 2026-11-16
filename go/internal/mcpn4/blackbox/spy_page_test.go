package blackbox

// n4_spy_page: one diagnostics page of the station's spy tree as data. The pages are recorded
// answers of a laboratory station (Niagara 4.15), anonymized.

import (
	"reflect"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest/boxstation"
)

func TestSpyPageReturnsTheTablesOfAPageAsData(t *testing.T) {
	r, sv := shxSpyRig(t)
	out := r.Srv.OK("n4_spy_page", Args{"path": "metrics"})
	if out["status"] != "ok" || out["http_status"] != 200.0 || out["path"] != "metrics" || out["title"] != "metrics" ||
		out["truncated"] != false {
		t.Errorf("out = %v", out)
	}
	tables := objects(out["tables"])
	if len(tables) != 2 || out["row_count"] != 9.0 {
		t.Fatalf("tables = %v", out["tables"])
	}
	want := map[string]any{"title": "Global Capacity", "header": []any{"Type", "Limit", "Used"}, "rows": []any{
		[]any{"Networks", "none", "10"}, []any{"Devices", "none", "11"}, []any{"Points", "none", "36"},
		[]any{"Links", "none", "94"}, []any{"Histories", "none", "14"}, []any{"Schedules", "none", "5"}}}
	if !reflect.DeepEqual(tables[1], want) {
		t.Errorf("capacity table = %v", tables[1])
	}
	wantStrings(t, "spy requests", shxSpyRequests(sv), []string{"/ord?spy:/metrics"})
}

func TestSpyPageListsChildPagesAsRelativePathsThatCanBeReadNext(t *testing.T) {
	r, sv := shxSpyRig(t)
	out := r.Srv.OK("n4_spy_page", nil)
	links := objects(out["links"])
	if out["title"] != "Index" || len(links) != 19 || !reflect.DeepEqual(links[5], map[string]any{"name": "util", "path": "util"}) {
		t.Fatalf("title %v, links %v", out["title"], out["links"])
	}
	child := r.Srv.OK("n4_spy_page", Args{"path": links[5]["path"]})
	if got := objects(child["links"]); len(got) != 1 || got[0]["path"] != "util/JMX Info" {
		t.Errorf("util links = %v", child["links"])
	}
	r.Srv.OK("n4_spy_page", Args{"path": "util/JMX Info"})
	wantStrings(t, "spy requests", shxSpyRequests(sv), []string{"/ord?spy:/", "/ord?spy:/util", "/ord?spy:/util/JMX%20Info"})
}

func TestSpyPageRefusesUnsafePathsBeforeAnyRequest(t *testing.T) {
	r, sv := shxSpyRig(t)
	before := getCount(sv)
	for _, bad := range []string{"spy:/metrics", "https://other.invalid/x", "//other.invalid/x", "../metrics",
		"util/../../file/config.bog", "metrics|bql:select name from baja:Component", "logSetup?level=FINE",
		"a&b=c", "util/%2e%2e", "a;b", `a\b`, "a\nb", "util/gc", "util/forceThreadDump", "/Util/GC"} {
		if text := r.Srv.Err("n4_spy_page", Args{"path": bad}); !strings.Contains(text, "refusing spy path") {
			t.Errorf("%q: %q", bad, text)
		}
	}
	if got := shxSpyRequests(sv); len(got) != 0 || getCount(sv) != before {
		t.Errorf("a refused path reached the station: %q (%d GETs, were %d)", got, getCount(sv), before)
	}
}

func TestSpyPageNamesTheActionsOfAPageAndNeverTheirFormFields(t *testing.T) {
	r, _ := shxSpyRig(t)
	res := r.Srv.Call("n4_spy_page", Args{"path": "util"})
	if got := res.Structured["actions"]; !reflect.DeepEqual(got, []any{"util/gc", "util/forceThreadDump"}) {
		t.Errorf("actions = %v", got)
	}
	if strings.Contains(res.Text, "FIXTURE-CSRF") || strings.Contains(res.Text, "csrf") {
		t.Errorf("a form field leaked: %s", res.Text)
	}
}

func TestSpyPageAnswersATypedStatusForAPageItCannotRead(t *testing.T) {
	r, sv := shxSpyRig(t)
	shxServeSpy(sv, "securityInfo", boxstation.SpyPage{Status: 403})
	shxServeSpy(sv, "gone", boxstation.SpyPage{Status: 404})
	shxServeSpy(sv, "csv", boxstation.SpyPage{Body: []byte("a,b\n1,2\n"), ContentType: "text/csv"})
	for path, want := range map[string][2]any{"securityInfo": {"forbidden", 403.0}, "gone": {"not_found", 404.0},
		"nope": {"not_found", 200.0}, "csv": {"not_html", 200.0}} {
		out := r.Srv.OK("n4_spy_page", Args{"path": path})
		if out["status"] != want[0] || out["http_status"] != want[1] {
			t.Errorf("%s: status %v, http_status %v", path, out["status"], out["http_status"])
		}
		if _, has := out["tables"]; has {
			t.Errorf("%s: an unreadable page carries tables: %v", path, out)
		}
	}
	if hint, _ := r.Srv.OK("n4_spy_page", Args{"path": "securityInfo"})["hint"].(string); !strings.Contains(hint, "spy") {
		t.Errorf("hint = %q", hint)
	}
}

func TestSpyPageCapsRowsAndBytesAndSaysWhichCapCut(t *testing.T) {
	r, sv := shxSpyRig(t)
	out := r.Srv.OK("n4_spy_page", Args{"path": "metrics", "max_rows": 4})
	if out["truncated"] != true || out["row_count"] != 4.0 || !reflect.DeepEqual(out["truncated_by"], []any{"max_rows"}) {
		t.Errorf("row cap: %v", out)
	}
	big := "<table><tr><td>" + strings.Repeat("x", 1500) + "</td></tr><tr><td>tail</td></tr></table>"
	shxServeSpy(sv, "big", boxstation.SpyPage{Body: []byte(big)})
	out = r.Srv.OK("n4_spy_page", Args{"path": "big", "max_bytes": 1024})
	if out["truncated"] != true || out["size_bytes"] != 1024.0 || !reflect.DeepEqual(out["truncated_by"], []any{"max_bytes"}) {
		t.Errorf("byte cap: %v", out)
	}
}

func TestSpyPageWithARefusedLoginIsAnAuthErrorNotAStatus(t *testing.T) {
	r, sv := shxSpyRig(t)
	sv.SetPassword("changed")
	if text := r.Srv.Err("n4_spy_page", Args{"path": "metrics"}); !strings.Contains(text, "401") {
		t.Errorf("error = %q", text)
	}
}

func TestSpyPageNeedsASessionAndNoWriteMode(t *testing.T) {
	r := NewRig(t, nil)
	tool := toolsByName(t, r.Srv)["n4_spy_page"]
	if Dig(tool, "annotations", "readOnlyHint") != true {
		t.Errorf("tools/list entry = %v", tool)
	}
	if text := r.Srv.Err("n4_spy_page", Args{"path": "metrics"}); !strings.Contains(text, "n4_connect") {
		t.Errorf("without a session: %q", text)
	}
}

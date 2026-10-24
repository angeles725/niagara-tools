package px

// Go twins of test_files.TestPxParser (go_server.go_twin("px", "unit")).

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/files"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

func syntheticPx(t testing.TB) string {
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "mcp-n4-kit", "tests", "fixtures", "synthetic_view.px"))
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func summary(t *testing.T) *pyjson.Object {
	t.Helper()
	s, err := Parse(syntheticPx(t), files.DefaultMaxItems)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// field is repr(s[key]) (repr of a nested key path).
func field(s *pyjson.Object, keys ...string) string {
	var v any = s
	for _, k := range keys {
		v, _ = v.(*pyjson.Object).Get(k)
	}
	return pyjson.Repr(v)
}

func want(t *testing.T, s *pyjson.Object, w string, keys ...string) {
	t.Helper()
	if got := field(s, keys...); got != w {
		t.Errorf("%v = %s\nwant %s", keys, got, w)
	}
}

func TestPxSummaryReportsMediaImportsCanvasAndLayers(t *testing.T) {
	s := summary(t)
	want(t, s, "'workbench:WbPxMedia'", "media")
	want(t, s, "['baja', 'bajaui', 'converters', 'gx', 'kitPx']", "imports")
	want(t, s, "[{'type': 'CanvasPane', 'name': 'content', 'view_size': '1024.0,768.0'}]", "canvases")
	want(t, s, "'1024.0,768.0'", "view_size")
	want(t, s, "['background', 'labels', 'values']", "layers")
}

func TestPxSummaryCountsWidgetsAndBindingsApart(t *testing.T) {
	s := summary(t)
	want(t, s, "3", "widget_counts", "BoundLabel")
	want(t, s, "2", "widget_counts", "Image")
	want(t, s, "1", "widget_counts", "Bargraph")
	want(t, s, "None", "widget_counts", "ValueBinding")
	want(t, s, "{'ValueBinding': 5, 'ActionBinding': 1}", "binding_counts")
}

func TestBoundOrdsAreGroupedByComponentWithTheTrailingSlotStripped(t *testing.T) {
	s := summary(t)
	want(t, s, "['station:|slot:/Plant/AHU1/SupplyTemp', 'station:|slot:/Plant/AHU1/SupplyTemp/out']",
		"bound_components", "station:|slot:/Plant/AHU1/SupplyTemp")
	want(t, s, "['station:|slot:/Plant/AHU1/ValvePos/out']", "bound_components", "station:|slot:/Plant/AHU1/ValvePos")
	b, _ := s.Get("bound_components")
	for _, k := range []string{"station:|slot:/Plant/AHU1/FanCmd", "station:|slot:/Plant/Schedule", "station:|slot:/Plant/AHU1"} {
		if _, ok := b.(*pyjson.Object).Get(k); !ok {
			t.Errorf("no bound component %s", k)
		}
	}
	if _, ok := b.(*pyjson.Object).Get("history:/demo/SupplyTemp"); ok {
		t.Error("a history ORD is bound")
	}
}

func TestImagesIncludedPxAndChartsAreListed(t *testing.T) {
	s := summary(t)
	want(t, s, "['module://demoModule/rc/logo.png', 'file:^images/floor%20plan.png']", "images")
	want(t, s, "['file:^px/includes/Header.px']", "included_px")
	want(t, s, "[{'type': 'WebChart', 'name': 'trend', 'ords': ['history:/demo/SupplyTemp']}]", "charts")
	if !strings.Contains(field(s, "other_ords"), "'history:/demo/SupplyTemp'") {
		t.Errorf("other_ords %s", field(s, "other_ords"))
	}
}

func TestLabelTextsAreListedAndBounded(t *testing.T) {
	if got := field(summary(t), "labels"); !strings.HasPrefix(got, "['Plant Overview', 'Supply Temp', '%out.value%', ") {
		t.Errorf("labels %s", got)
	}
	var many strings.Builder
	for i := range 50 {
		fmt.Fprintf(&many, `<Label text="t%d"/>`, i)
	}
	out, err := Parse(`<px media="m"><content>`+many.String()+`</content></px>`, 10)
	if err != nil {
		t.Fatal(err)
	}
	want(t, out, "['t0', 't1', 't2', 't3', 't4', 't5', 't6', 't7', 't8', 't9']", "labels")
	want(t, out, "True", "labels_truncated")
}

func TestADoctypeOrEntityDeclarationIsRefused(t *testing.T) {
	for _, text := range []string{`<!DOCTYPE px [<!ENTITY a "x">]><px>&a;</px>`,
		"<?xml version=\"1.0\"?>\n<!DOCTYPE px SYSTEM \"http://x/y.dtd\"><px/>", "<px><!-- <!entİty --></px>"} {
		if _, err := Parse(text, files.DefaultMaxItems); err == nil ||
			err.Error() != "refusing a Px document with a DOCTYPE or ENTITY declaration" {
			t.Errorf("%q: %v", text, err)
		}
	}
}

func TestANonPxDocumentOrBrokenXmlIsRefused(t *testing.T) {
	for text, w := range map[string]string{
		"<html><body/></html>": "not a Px document: the root element is <html>, expected <px>",
		"<px><content>":        "not well-formed Px XML: no element found: line 1, column 13",
		"not xml":              "not well-formed Px XML: syntax error: line 1, column 0",
	} {
		if _, err := Parse(text, files.DefaultMaxItems); err == nil || err.Error() != w {
			t.Errorf("%q: %v\nwant %s", text, err, w)
		}
	}
}

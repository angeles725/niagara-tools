package px

// Binding extraction and absolutization for n4_px_binding_audit (station-audits S15).

import (
	"reflect"
	"strings"
	"testing"
)

func TestBindingsListsEveryBindingOrdWithItsWidget(t *testing.T) {
	got, err := Bindings(syntheticPx(t))
	if err != nil {
		t.Fatal(err)
	}
	want := []Binding{
		{Element: "ValueBinding", Widget: "tempLabel", Ord: "station:|slot:/Plant/AHU1/SupplyTemp"},
		{Element: "ValueBinding", Widget: "tempOut", Ord: "station:|slot:/Plant/AHU1/SupplyTemp/out"},
		{Element: "ValueBinding", Widget: "fanLabel", Ord: "station:|slot:/Plant/AHU1/FanCmd"},
		{Element: "ValueBinding", Widget: "valve", Ord: "station:|slot:/Plant/AHU1/ValvePos/out"},
		{Element: "ActionBinding", Widget: "occ", Ord: "station:|slot:/Plant/Schedule"},
		{Element: "ValueBinding", Widget: "trend", Ord: "history:/demo/SupplyTemp"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Bindings =\n%+v\nwant\n%+v", got, want)
	}
}

func TestBindingsSkipsAnEmptyOrdAndRefusesWhatParseRefuses(t *testing.T) {
	got, err := Bindings(`<px><content><Label name="a"><ValueBinding ord=" "/><ValueBinding/></Label></content></px>`)
	if err != nil || len(got) != 0 {
		t.Errorf("Bindings = %v, %v; want none", got, err)
	}
	for _, text := range []string{"<html/>", "<px><content>", `<!DOCTYPE px><px/>`} {
		if _, err := Bindings(text); err == nil {
			t.Errorf("Bindings(%q) was not refused", text)
		}
	}
}

func TestAbsolutizeReadsEachFormAgainstTheBase(t *testing.T) {
	cases := []struct {
		ord, base string
		want      Target
	}{
		{"station:|slot:/A/B", "", Target{Path: "/A/B"}},
		{"slot:/A/B/out", "/X", Target{Path: "/A/B/out"}},
		{"local:|slot:/A/B|slot:out", "", Target{Path: "/A/B"}},
		{"slot:B/out", "/A", Target{Path: "/A/B/out"}},
		{"slot:../C", "/A/B", Target{Path: "/A/C"}},
		{"../C/./D", "/A/B", Target{Path: "/A/C/D"}},
		{"slot:B", "/", Target{Path: "/B"}},
		{"station:|h:2f3", "/A", Target{Handle: "2f3"}},
	}
	for _, c := range cases {
		if got := Absolutize(c.ord, c.base); got != c.want {
			t.Errorf("Absolutize(%q, %q) = %+v, want %+v", c.ord, c.base, got, c.want)
		}
	}
}

func TestAbsolutizeNamesWhyAnOrdCannotBeChecked(t *testing.T) {
	for _, c := range []struct{ ord, base, why string }{
		{"history:/demo/T", "/A", "scheme history:"},
		{"file:^px/x.px", "", "scheme file:"},
		{"slot:B", "", "needs the view's base"},
		{"slot:../../../X", "/A", "above the station root"},
		{"station:|slot:/A|bql:select x", "", "chained"},
		{"slot:/A//B", "", "plain slot path"},
		{"h:", "", "plain slot path"},
	} {
		got := Absolutize(c.ord, c.base)
		if got.Path != "" || got.Handle != "" || !strings.Contains(got.Why, c.why) {
			t.Errorf("Absolutize(%q, %q) = %+v, want why containing %q", c.ord, c.base, got, c.why)
		}
	}
}

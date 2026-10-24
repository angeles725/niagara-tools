package obixlinkaudit

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/xmlsrc"
)

const tableMap = "## Table 1 — OPERATOR config slots (facade → control)\n" +
	"| Facade slot | Target RT slot | notes |\n|---|---|---|\n" +
	"| `setpoint` | `Rack/suctionSetpoint` | |\n| `diffDown` | _(fill)_ | |\n" +
	"## Table 22 — not table 2\n| `zz` | `q` | |\n" +
	"## Table 2 — SUMMARY display slots (control → facade)\n" +
	"| `comp1State` | `slot:/Plant//Rack.comp1Running` | |\n| `freeze` | | |\n| `short` | `x` |\n" +
	"| ----- | y | |\n| `_(todo)` | z | |\n| `bare` | `` `dripActive` `` | |\n"

func TestDeclaredRows(t *testing.T) {
	t.Parallel()
	if got, want := DeclaredRows(tableMap, "2"), []Declared{
		{Slot: "comp1State", Want: "comp1Running", Label: "comp1State", Comp: "Plant//Rack"}, // an inner // is kept, as bash did
		{Slot: "freeze", Label: "freeze"},
		{Slot: "short", Want: "x", Label: "short"}, // `| a | b |` is 4 cells
		{Slot: "bare", Want: "dripActive", Label: "bare"},
	}; !reflect.DeepEqual(got, want) {
		t.Errorf("Table 2 rows\n got %+v\nwant %+v", got, want)
	}
	if got, want := DeclaredRows(tableMap, "1"), []Declared{
		{Slot: "suctionSetpoint", Want: "setpoint", Label: "setpoint"},
		{Want: "diffDown", Label: "diffDown"},
	}; !reflect.DeepEqual(got, want) {
		t.Errorf("Table 1 rows\n got %+v\nwant %+v", got, want)
	}
}

func TestLinks(t *testing.T) {
	t.Parallel()
	doc, err := xmlsrc.Parse("t.xml", []byte(`<obj>
<obj display="Indirect: slot:/P/IO.relayOut1 &#x2192; slot:/P/F.comp1State"/>
<obj display="Direct:  bare  ->  F.s2 "/><obj display="Knob: a.b -> F.s3"/><obj display="Direct: a.b to F.s4"/>
<obj display='Indirect: slot:/P/R.x &#8594; slot:/P/F.s5 -> G.s6'/></obj>`))
	if err != nil {
		t.Fatal(err)
	}
	want := []Link{{"comp1State", "relayOut1", "slot:/P/IO"}, {"s2", "bare", "bare"}, {"s6", "x", "slot:/P/R"}}
	if got := Links(doc); !reflect.DeepEqual(got, want) {
		t.Errorf("Links\n got %+v\nwant %+v", got, want)
	}
}

func TestReport(t *testing.T) {
	t.Parallel()
	rows := []Declared{
		{Slot: "a", Want: "src", Label: "a", Comp: "Rack2"},
		{Slot: "b", Want: "src", Label: "b", Comp: "Rack2"},
		{Slot: "c", Want: "src", Label: "c"},
		{Slot: "d", Want: "src", Label: "d"},
		{Want: "e", Label: "e"},
	}
	links := []Link{{"a", "src", "slot:/Plant/Rack2"}, {"b", "src", "slot:/Plant/Rack12"}, {"b", "other", "Rack2"}, {"c", "src", "x"}}
	var out bytes.Buffer
	code := Report(&out, rows, links)
	want := "MATCH  obix-link-audit  a  linked from declared src in Rack2\n" +
		"MISMATCH  obix-link-audit  b  declared src in Rack2, linked from src (slot:/Plant/Rack12), other (Rack2)\n" +
		"MATCH  obix-link-audit  c  linked from declared src\n" +
		"MISSING  obix-link-audit  d  declared src, no link into d\n" +
		"SKIP  obix-link-audit  e  source not declared in the wiring map (fill it before commissioning)\n" +
		"obix-link-audit: 2 MATCH · 1 MISMATCH · 1 MISSING · 1 SKIP  ->  ISSUES\n"
	if code != 1 || out.String() != want {
		t.Errorf("Report = %d\n%s\nwant 1\n%s", code, out.String(), want)
	}
	out.Reset()
	if code := Report(&out, rows[:1], links); code != 0 || out.String() != "MATCH  obix-link-audit  a  linked from declared src in Rack2\n"+
		"obix-link-audit: 1 MATCH · 0 MISMATCH · 0 MISSING · 0 SKIP  ->  CLEAN\n" {
		t.Errorf("clean Report = %d\n%s", code, out.String())
	}
}

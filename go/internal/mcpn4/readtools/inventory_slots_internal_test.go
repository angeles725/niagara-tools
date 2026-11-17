package readtools

import (
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// typesStub answers every registry call with types.
type typesStub struct{ types *pyjson.Object }

func (s typesStub) LoadTypes([]string) (any, error) { return s.types, nil }

func parsed(t *testing.T, text string) *pyjson.Object {
	t.Helper()
	v, err := pyjson.Unmarshal([]byte(text))
	if err != nil {
		t.Fatal(err)
	}
	return v.(*pyjson.Object)
}

// writableNodes is a NumericWritable as a 4.15 load sends it: no instance flags on out, fallback,
// in1 (linked) or in10, as the live check of #803 read them.
const writableNodes = `{
 "": {"t": "control:NumericWritable", "h": "10"},
 "out": {"n": "out", "t": "baja:StatusNumeric"}, "out/value": {"n": "value", "t": "baja:Double", "v": "33.5"},
 "fallback": {"n": "fallback", "t": "baja:StatusNumeric"}, "fallback/value": {"n": "value", "t": "baja:Double", "v": "21.0"},
 "in10": {"n": "in10", "t": "baja:StatusNumeric"}, "in10/value": {"n": "value", "t": "baja:Double", "v": "5.0"},
 "in1": {"n": "in1", "t": "baja:StatusNumeric"}, "in1/value": {"n": "value", "t": "baja:Double", "v": "7.0"},
 "facets": {"n": "facets", "t": "baja:Facets", "v": "units=C"},
 "Link": {"n": "Link", "t": "baja:Link", "s": [{"n": "targetSlotName", "t": "baja:String", "v": "in1"}]}}`

// writableContract is BNumericWritable's (BNumericPoint.java:31, BNumericWritable.java:138,147,154).
const writableContract = `{"control:NumericWritable": {"c": [{"st": "p", "n": "out", "f": "rtso"},
 {"st": "p", "n": "in1", "f": "r"}, {"st": "p", "n": "in10", "f": "ts"}, {"st": "p", "n": "fallback"}]}}`

func TestInventorySlotsKeepAStatusConfigSlotAndDropTheRuntimeOnes(t *testing.T) {
	ct := plan.NewContracts(typesStub{parsed(t, writableContract)})
	if err := ct.Load([]string{"control:NumericWritable"}); err != nil {
		t.Fatal(err)
	}
	for name, r := range map[string]*invReader{
		"contracts":    {nodes: parsed(t, writableNodes), ct: ct},
		"no contracts": {nodes: parsed(t, writableNodes)}, // in10 stays: only the contract flags it
	} {
		want := `{"fallback": "21.0 {0}", "in10": "5.0 {0}", "facets": "units=C"}`
		if name == "contracts" {
			want = `{"fallback": "21.0 {0}", "facets": "units=C"}`
		}
		if got := asJSON(t, r.slots("")); got != want {
			t.Errorf("%s: slots = %s, want %s", name, got, want)
		}
	}
}

func TestInventorySlotsKeepAConfiguredOut(t *testing.T) {
	ct := plan.NewContracts(typesStub{parsed(t, `{"kitControl:NumericConst": {"c": [{"st": "p", "n": "out", "f": "s"}]}}`)})
	r := &invReader{ct: ct, nodes: parsed(t, `{"": {"t": "kitControl:NumericConst"}, "out": {"n": "out", "t": "baja:StatusNumeric"},
 "out/value": {"n": "value", "t": "baja:Double", "v": "4.0"}}`)}
	if got := asJSON(t, r.slots("")); got != `{"out": "4.0 {0}"}` {
		t.Errorf("slots = %s", got)
	}
}

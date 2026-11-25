package writeops_test

// Retro 2026-11-02 row 9: the enum-writable slots a lab check needs (facets, minActiveTime) are
// frozen properties the station omits at their default. A set of them is planned from the type
// contract and read back, as any other declared default.

import (
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/writeops"
)

const bw = "station:|slot:/Folder/Pump"

func bwProp(name string, v *pyjson.Object) *pyjson.Object {
	return obj("st", "p", "n", name, "v", v)
}

// bwTypes is the registry of control:BooleanWritable: in1 (a StatusBoolean), facets (baja:Facets)
// and minActiveTime (baja:RelTime) declared, each at its default, the way a station sends them.
func bwTypes(t *testing.T) *typed {
	t.Helper()
	c := newTyped(t, extNodes())
	types := c.types.(*pyjson.Object)
	types.Set("control:BooleanWritable", obj("p", "baja:Component", "c", []any{
		bwProp("in1", obj("nm", "p", "t", "baja:StatusBoolean", "d", "false {null}", "s", []any{
			obj("nm", "p", "n", "value", "t", "baja:Boolean", "v", "false"),
			obj("nm", "p", "n", "status", "t", "baja:Status", "v", "20")})),
		bwProp("facets", obj("nm", "p", "t", "baja:Facets", "v", "", "d", "")),
		bwProp("minActiveTime", obj("nm", "p", "t", "baja:RelTime", "v", "0", "d", "0 s"))}))
	types.Set("baja:Component", obj())
	c.nodes = obj(bw, obj("", obj("h", "5", "t", "control:BooleanWritable")))
	return c
}

func bwSet(slot string, value any, valueType string) *pyjson.Object {
	return obj("ord", bw, "slot", slot, "value", value, "value_type", valueType)
}

func bwReadback(t *testing.T, c *typed, args *pyjson.Object, after ...any) (string, string, string) {
	t.Helper()
	p, err := writeops.SetSlot.Plan(c, args)
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	n := obj("", obj("h", "5", "t", "control:BooleanWritable"))
	for i := 0; i < len(after); i += 2 {
		n.Set(after[i].(string), after[i+1])
	}
	c.nodes = obj(bw, n)
	_, _, _, verdict, err := writeops.SetSlot.Readback(c, args, p, []any{"reply"}, p.Inverse)
	if err != nil {
		t.Fatalf("readback: %v", err)
	}
	return js(t, p.Ops[0]), js(t, p.Inverse[0]), verdict
}

func TestMinActiveTimeAtItsDefaultIsSetAsARelTimeAndRolledBackToZero(t *testing.T) {
	op, inv, verdict := bwReadback(t, bwTypes(t), bwSet("minActiveTime", pyjson.IntLiteral("5000"), "baja:RelTime"),
		"minActiveTime", obj("t", "baja:RelTime", "v", "5000"))
	if want := `{"nm": "s", "h": "5", "n": "minActiveTime", "b": {"nm": "p", "t": "baja:RelTime", "v": "5000"}}`; op != want {
		t.Errorf("op\n got  %s\n want %s", op, want)
	}
	if want := `{"nm": "s", "h": "5", "n": "minActiveTime", "b": {"nm": "p", "t": "baja:RelTime", "v": "0"}}`; inv != want {
		t.Errorf("inverse\n got  %s\n want %s", inv, want)
	}
	if verdict != "verified" {
		t.Errorf("verdict = %q", verdict)
	}
}

func TestFacetsAtTheirDefaultAreSetAsASimpleOfTypeFacetsAndRolledBackToEmpty(t *testing.T) {
	const f = "range=E:{off=0,on=1}"
	op, inv, verdict := bwReadback(t, bwTypes(t), bwSet("facets", f, "baja:Simple"),
		"facets", obj("t", "baja:Facets", "v", f))
	if want := `{"nm": "s", "h": "5", "n": "facets", "b": {"nm": "p", "t": "baja:Facets", "v": "` + f + `"}}`; op != want {
		t.Errorf("op\n got  %s\n want %s", op, want)
	}
	if want := `{"nm": "s", "h": "5", "n": "facets", "b": {"nm": "p", "t": "baja:Facets", "v": ""}}`; inv != want {
		t.Errorf("inverse\n got  %s\n want %s", inv, want)
	}
	if verdict != "verified" {
		t.Errorf("verdict = %q", verdict)
	}
}

// Setting the facets back to the empty default: the station omits the slot again, which reads back
// as the empty string.
func TestFacetsSetBackToEmptyReadBackVerifiedWhenTheStationOmitsThem(t *testing.T) {
	c := bwTypes(t)
	c.nodes = obj(bw, obj("", obj("h", "5", "t", "control:BooleanWritable"), "facets", obj("t", "baja:Facets", "v", "precision=i:1")))
	_, inv, verdict := bwReadback(t, c, bwSet("facets", "", "baja:Simple"))
	if !strings.Contains(inv, `"v": "precision=i:1"`) || verdict != "verified" {
		t.Errorf("inverse %s, verdict %q", inv, verdict)
	}
}

func TestAFacetsValueIsNotAcceptedUnderAnotherTypeThanTheSlots(t *testing.T) {
	_, err := writeops.SetSlot.Plan(bwTypes(t), bwSet("facets", "x", "baja:String"))
	if err == nil || err.Error() != "slot 'facets' is baja:Facets, not baja:String" {
		t.Errorf("err = %v", err)
	}
}

// A station may send the empty facets default without a "v": the inverse is still the empty string.
func TestAFacetsDefaultSentWithoutAValueIsTheEmptyString(t *testing.T) {
	c := bwTypes(t)
	types := c.types.(*pyjson.Object)
	types.Set("control:BooleanWritable", obj("p", "baja:Component", "c", []any{
		bwProp("facets", obj("nm", "p", "t", "baja:Facets"))}))
	_, inv, verdict := bwReadback(t, c, bwSet("facets", "precision=i:1", "baja:Simple"),
		"facets", obj("t", "baja:Facets", "v", "precision=i:1"))
	if want := `{"nm": "s", "h": "5", "n": "facets", "b": {"nm": "p", "t": "baja:Facets", "v": ""}}`; inv != want || verdict != "verified" {
		t.Errorf("inverse %s, verdict %q", inv, verdict)
	}
}

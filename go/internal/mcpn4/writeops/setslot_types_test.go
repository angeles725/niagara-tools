package writeops_test

import (
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/writeops"
)

func TestAnOmittedDefaultSlotTakesItsTypeFromTheTypeDefinition(t *testing.T) {
	for _, tc := range []struct {
		name, slot, valueType string
		value                 any
		after                 *pyjson.Object
		op, inverse, readback string
	}{
		{"enabled", "enabled", "baja:Boolean", true, obj("t", "baja:Boolean", "v", "true"),
			`{"nm": "s", "h": "42", "n": "enabled", "b": {"nm": "p", "t": "baja:Boolean", "v": "true"}}`,
			`{"nm": "s", "h": "42", "n": "enabled", "b": {"nm": "p", "t": "baja:Boolean", "v": "false"}}`,
			`[true, "reply", true, "verified"]`},
		{"historyName", "historyName", "baja:Format", "S%parent.parent.parent.name%_%parent.name%",
			obj("t", "baja:Format", "v", "S%parent.parent.parent.name%_%parent.name%"),
			`{"nm": "s", "h": "42", "n": "historyName", "b": {"nm": "p", "t": "baja:Format", "v": "S%parent.parent.parent.name%_%parent.name%"}}`,
			`{"nm": "s", "h": "42", "n": "historyName", "b": {"nm": "p", "t": "baja:Format", "v": "%parent.name%"}}`,
			`["S%parent.parent.parent.name%_%parent.name%", "reply", "S%parent.parent.parent.name%_%parent.name%", "verified"]`},
		{"interval 5 min", "interval", "baja:RelTime", pyjson.IntLiteral("300000"), obj("t", "baja:RelTime", "v", "300000"),
			`{"nm": "s", "h": "42", "n": "interval", "b": {"nm": "p", "t": "baja:RelTime", "v": "300000"}}`,
			`{"nm": "s", "h": "42", "n": "interval", "b": {"nm": "p", "t": "baja:RelTime", "v": "900000"}}`,
			`["300000", "reply", "300000", "verified"]`},
		// a nested slot whose owner is listed: the owner's own type definition answers
		{"nested historyConfig/historyName", "historyConfig/historyName", "baja:String", "x",
			obj("t", "baja:String", "v", "x"),
			`{"nm": "s", "h": "42", "n": "historyConfig/historyName", "b": {"nm": "p", "t": "baja:String", "v": "x"}}`,
			`{"nm": "s", "h": "42", "n": "historyConfig/historyName", "b": {"nm": "p", "t": "baja:String", "v": ""}}`,
			`["x", "reply", "x", "verified"]`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := newTyped(t, extNodes())
			p, rb := planAndReadback(t, c, extSet(tc.slot, tc.value, tc.valueType), extNodes(tc.slot, tc.after))
			if got := js(t, p.Ops[0]); got != tc.op {
				t.Errorf("op\n got  %s\n want %s", got, tc.op)
			}
			if got := js(t, p.Inverse[0]); got != tc.inverse {
				t.Errorf("inverse\n got  %s\n want %s", got, tc.inverse)
			}
			if got := js(t, rb); got != tc.readback {
				t.Errorf("readback\n got  %s\n want %s", got, tc.readback)
			}
		})
	}
}

// A slot set back to its default is omitted again: the read-back compares with the definition's
// default, not with an empty value.
func TestAReadBackOfAValueEqualToTheDefaultIsVerified(t *testing.T) {
	c := newTyped(t, extNodes("interval", obj("t", "baja:RelTime", "v", "300000")))
	p, rb := planAndReadback(t, c, extSet("interval", pyjson.IntLiteral("900000"), "baja:RelTime"), extNodes())
	if got := js(t, p.Inverse[0]); !strings.Contains(got, `"v": "300000"`) {
		t.Errorf("inverse = %s, want the listed 300000", got)
	}
	if got := js(t, rb); got != `["900000", "reply", "900000", "verified"]` {
		t.Errorf("readback = %s", got)
	}
}

// An omitted owner is typed from its parent's definition, then the slot from the owner's.
func TestAnOmittedOwnerOfANestedSlotIsTypedThroughItsParent(t *testing.T) {
	c := newTyped(t, obj("", obj("h", "42", "t", "history:NumericIntervalHistoryExt")))
	p, err := writeops.SetSlot.Plan(c, extSet("historyConfig/historyName", "x", "baja:String"))
	if err != nil {
		t.Fatal(err)
	}
	if got := js(t, p.Inverse[0]); !strings.Contains(got, `"t": "baja:String", "v": ""`) {
		t.Errorf("inverse = %s", got)
	}
}

func TestTheTypeDefinitionStillRefusesAWrongTypeOrAnUnknownSlot(t *testing.T) {
	for _, tc := range []struct{ slot, valueType, want string }{
		{"enabled", "baja:String", "slot 'enabled' is baja:Boolean, not baja:String"},
		{"historyName", "baja:String", "slot 'historyName' is baja:Format, not baja:String"},
		{"ghost", "baja:Boolean", "slot 'ghost' not found on " + ext + ". The station leaves out a frozen property whose value equals its declared default"},
		{"enable", "baja:Boolean", "slot 'enable' not found on "}, // an action, not a property
	} {
		var value any = "x"
		if tc.valueType == "baja:Boolean" {
			value = true
		}
		_, err := writeops.SetSlot.Plan(newTyped(t, extNodes()), extSet(tc.slot, value, tc.valueType))
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s as %s: err = %v, want %q", tc.slot, tc.valueType, err, tc.want)
		}
	}
}

func TestFormatAndRelTimeValuesAreCheckedBeforeAnythingIsLoaded(t *testing.T) {
	for _, tc := range []struct {
		valueType string
		value     any
		want      string
	}{
		{"baja:Format", pyjson.IntLiteral("5"), "value must be a string for baja:Format"},
		{"baja:RelTime", "5m", "value must be a whole number of milliseconds (0 or more) for baja:RelTime"},
		{"baja:RelTime", pyjson.FloatNumber(1.5), "value must be a whole number of milliseconds (0 or more) for baja:RelTime"},
		{"baja:RelTime", pyjson.IntLiteral("-1"), "value must be a whole number of milliseconds (0 or more) for baja:RelTime"},
		{"baja:RelTime", pyjson.IntLiteral("9223372036854775808"), "value must be a whole number of milliseconds (0 or more) for baja:RelTime"},
		{"baja:RelTime", true, "value must be a whole number of milliseconds (0 or more) for baja:RelTime"},
	} {
		c := newTyped(t, extNodes())
		_, err := writeops.SetSlot.Plan(c, extSet("interval", tc.value, tc.valueType))
		if err == nil || err.Error() != tc.want {
			t.Errorf("%s %v: err = %v, want %q", tc.valueType, tc.value, err, tc.want)
		}
		if len(c.calls) != 0 {
			t.Errorf("%s %v: asked the registry %v", tc.valueType, tc.value, c.calls)
		}
	}
}

// Every value type already supported keeps its plan and read-back when the station lists the
// slot, and the registry is never asked: the listed value's type stays authoritative.
func TestAListedSlotOfEachExistingTypeIsUnchangedAndNeverAsksTheRegistry(t *testing.T) {
	for _, tc := range []struct {
		valueType string
		value     any
		listed    []any
	}{
		{"baja:Double", pyjson.FloatNumber(2.5), []any{"sp", obj("t", "baja:Double", "v", "1.0")}},
		{"baja:Boolean", false, []any{"sp", obj("t", "baja:Boolean", "v", "true")}},
		{"baja:String", "b", []any{"sp", obj("t", "baja:String", "v", "a")}},
		{"baja:StatusNumeric", pyjson.FloatNumber(2.5), []any{"sp", obj("t", "baja:StatusNumeric"),
			"sp/value", obj("t", "baja:Double", "v", "1.0"), "sp/status", obj("t", "baja:Status", "v", "0")}},
		{"baja:StatusBoolean", true, []any{"sp", obj("t", "baja:StatusBoolean"),
			"sp/value", obj("t", "baja:Boolean", "v", "false"), "sp/status", obj("t", "baja:Status", "v", "0")}},
	} {
		args := extSet("sp", tc.value, tc.valueType)
		plain := &loads{nodes: obj(ext, extNodes(tc.listed...))}
		want, err := writeops.SetSlot.Plan(plain, args)
		if err != nil {
			t.Fatalf("%s: %v", tc.valueType, err)
		}
		c := newTyped(t, extNodes(tc.listed...))
		got, err := writeops.SetSlot.Plan(c, args)
		if err != nil {
			t.Fatalf("%s: %v", tc.valueType, err)
		}
		if js(t, got.Ops) != js(t, want.Ops) || js(t, got.Inverse) != js(t, want.Inverse) {
			t.Errorf("%s: plan %s / %s, want %s / %s", tc.valueType, js(t, got.Ops), js(t, got.Inverse),
				js(t, want.Ops), js(t, want.Inverse))
		}
		if _, _, _, verdict, err := writeops.SetSlot.Readback(c, args, got, []any{"reply"}, got.Inverse); err != nil || verdict != "mismatch" {
			t.Errorf("%s: read-back of the unchanged station = %q, %v; want mismatch", tc.valueType, verdict, err)
		}
		if len(c.calls) != 0 {
			t.Errorf("%s: asked the registry %v", tc.valueType, c.calls)
		}
	}
}

// An omitted Status slot is typed from the definition too, and its default children make the
// previous value (the default StatusNumeric: 0.0, status 0).
func TestAnOmittedStatusSlotIsTypedFromTheDefinitionWithItsDefaultChildren(t *testing.T) {
	c := newTyped(t, obj("", obj("h", "7", "t", "control:NumericPoint")))
	p, err := writeops.SetSlot.Plan(c, extSet("out", pyjson.FloatNumber(1), "baja:StatusNumeric"))
	if err != nil {
		t.Fatal(err)
	}
	if got := js(t, p.Inverse[0]); !strings.Contains(got, `"t": "baja:Double", "v": "0.0", "n": "value"`) {
		t.Errorf("inverse = %s", got)
	}
}

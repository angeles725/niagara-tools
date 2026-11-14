package writeops_test

import (
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/writeops"
)

// mb is a Modbus client point proxy extension, the shape a field gateway point has on a station.
const mb = "station:|slot:/Drivers/ModbusAsyncNetwork/Gw/points/Setpoint/proxyExt"

func mbNodes(extra ...any) *pyjson.Object {
	n := obj("", obj("h", "9", "t", "modbusCore:ModbusClientNumericProxyExt"),
		"deviceAddress", obj("t", "baja:Integer", "v", "2"),
		"dataType", obj("t", "modbusCore:DataTypeEnum", "v", "integerType"),
		"conversion", obj("t", "driver:DefaultProxyConversion", "v", ""),
		"deviceFacets", obj("t", "baja:Facets", "v", "precision=i:1"),
		"dataAddress", obj("t", "modbusCore:FlexAddress"),
		"dataAddress/addressFormat", obj("t", "modbusCore:AddressFormatEnum", "v", "modbus"),
		"dataAddress/address", obj("t", "baja:String", "v", "40005"))
	for i := 0; i < len(extra); i += 2 {
		n.Set(extra[i].(string), extra[i+1])
	}
	return n
}

func mbSet(slot string, value any, valueType string, kv ...any) *pyjson.Object {
	a := obj("ord", mb, "slot", slot, "value", value, "value_type", valueType)
	for i := 0; i < len(kv); i += 2 {
		a.Set(kv[i].(string), kv[i+1])
	}
	return a
}

func mbPlanAndReadback(t *testing.T, args, after *pyjson.Object) (string, string, string, []any) {
	t.Helper()
	c := &loads{nodes: obj(mb, mbNodes())}
	p, err := writeops.SetSlot.Plan(c, args)
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	c.nodes = obj(mb, after)
	requested, accepted, observed, verdict, err := writeops.SetSlot.Readback(c, args, p, []any{"reply"}, p.Inverse)
	if err != nil {
		t.Fatalf("readback: %v", err)
	}
	return js(t, p.Ops[0]), js(t, p.Inverse[0]), js(t, p.Notes), []any{requested, accepted, observed, verdict}
}

func TestAnIntegerSlotIsWrittenAsAWholeNumber(t *testing.T) {
	op, inv, _, rb := mbPlanAndReadback(t, mbSet("deviceAddress", pyjson.IntLiteral("5"), "baja:Integer"),
		mbNodes("deviceAddress", obj("t", "baja:Integer", "v", "5")))
	if want := `{"nm": "s", "h": "9", "n": "deviceAddress", "b": {"nm": "p", "t": "baja:Integer", "v": "5"}}`; op != want {
		t.Errorf("op\n got  %s\n want %s", op, want)
	}
	if want := `{"nm": "s", "h": "9", "n": "deviceAddress", "b": {"nm": "p", "t": "baja:Integer", "v": "2"}}`; inv != want {
		t.Errorf("inverse\n got  %s\n want %s", inv, want)
	}
	if got := js(t, rb); got != `["5", "reply", "5", "verified"]` {
		t.Errorf("readback = %s", got)
	}
}

func TestAnIntegerValueOutsideTheJavaIntRangeOrNotWholeIsRefused(t *testing.T) {
	want := "value must be a whole number from -2147483648 to 2147483647 for baja:Integer"
	for _, v := range []any{"5", pyjson.FloatNumber(1.5), true, pyjson.IntLiteral("2147483648"), pyjson.IntLiteral("-2147483649")} {
		c := &loads{nodes: obj(mb, mbNodes())}
		if _, err := writeops.SetSlot.Plan(c, mbSet("deviceAddress", v, "baja:Integer")); err == nil || err.Error() != want {
			t.Errorf("%v: err = %v, want %q", v, err, want)
		}
	}
}

// An enum (or any other simple) keeps the slot's own type: the value is its encoded form, the tag.
func TestASimpleValueKeepsTheSlotTypeAndSendsTheEncodedForm(t *testing.T) {
	op, inv, notes, rb := mbPlanAndReadback(t, mbSet("dataType", "signedInteger", "baja:Simple"),
		mbNodes("dataType", obj("t", "modbusCore:DataTypeEnum", "v", "signedInteger")))
	if want := `{"nm": "s", "h": "9", "n": "dataType", "b": {"nm": "p", "t": "modbusCore:DataTypeEnum", "v": "signedInteger"}}`; op != want {
		t.Errorf("op\n got  %s\n want %s", op, want)
	}
	if want := `{"nm": "s", "h": "9", "n": "dataType", "b": {"nm": "p", "t": "modbusCore:DataTypeEnum", "v": "integerType"}}`; inv != want {
		t.Errorf("inverse\n got  %s\n want %s", inv, want)
	}
	if notes != `[]` {
		t.Errorf("notes = %s, want none", notes)
	}
	if got := js(t, rb); got != `["signedInteger", "reply", "signedInteger", "verified"]` {
		t.Errorf("readback = %s", got)
	}
}

// simple_type replaces the value's type (a proxy conversion from default to linear); the inverse
// restores the previous type and value, and a note says the type changes.
func TestSimpleTypeReplacesTheValueTypeAndTheInverseRestoresTheOldOne(t *testing.T) {
	op, inv, notes, rb := mbPlanAndReadback(t, mbSet("conversion", "0.1;0.0", "baja:Simple", "simple_type", "driver:LinearConversion"),
		mbNodes("conversion", obj("t", "driver:LinearConversion", "v", "0.1;0.0")))
	if want := `{"nm": "s", "h": "9", "n": "conversion", "b": {"nm": "p", "t": "driver:LinearConversion", "v": "0.1;0.0"}}`; op != want {
		t.Errorf("op\n got  %s\n want %s", op, want)
	}
	if want := `{"nm": "s", "h": "9", "n": "conversion", "b": {"nm": "p", "t": "driver:DefaultProxyConversion", "v": ""}}`; inv != want {
		t.Errorf("inverse\n got  %s\n want %s", inv, want)
	}
	if !strings.Contains(notes, "driver:DefaultProxyConversion is replaced by driver:LinearConversion") {
		t.Errorf("notes = %s", notes)
	}
	if got := js(t, rb); got != `["0.1;0.0", "reply", "0.1;0.0", "verified"]` {
		t.Errorf("readback = %s", got)
	}
}

// The read-back of a type change also checks the type: the same text under the old type is a mismatch.
func TestATypeChangeTheStationDidNotApplyIsAMismatch(t *testing.T) {
	_, _, _, rb := mbPlanAndReadback(t, mbSet("conversion", "0.1;0.0", "baja:Simple", "simple_type", "driver:LinearConversion"),
		mbNodes("conversion", obj("t", "driver:DefaultProxyConversion", "v", "0.1;0.0")))
	if rb[3] != "mismatch" {
		t.Errorf("verdict = %v, want mismatch", rb[3])
	}
}

func TestASimpleWriteRefusesBadInputBeforeOrAfterLoading(t *testing.T) {
	for _, tc := range []struct {
		args *pyjson.Object
		want string
	}{
		{mbSet("dataType", true, "baja:Simple"), "value must be a string or a whole number (the encoded form, e.g. an enum tag) for baja:Simple"},
		{mbSet("dataType", "x", "baja:Simple", "simple_type", "Linear"), "simple_type must be a type spec module:Type, got 'Linear'"},
		{mbSet("dataType", "x", "baja:String", "simple_type", "driver:LinearConversion"), "simple_type only applies to baja:Simple"},
		{mbSet("dataAddress", "modbus:40005", "baja:Simple"), "slot 'dataAddress' is a struct (modbusCore:FlexAddress); set its child slots, e.g. dataAddress/addressFormat"},
		{mbSet("deviceAddress", "x", "baja:Simple", "simple_type", "baja:StatusNumeric"), "baja:Simple does not write Status types; use baja:StatusNumeric or baja:StatusBoolean"},
	} {
		c := &loads{nodes: obj(mb, mbNodes())}
		if _, err := writeops.SetSlot.Plan(c, tc.args); err == nil || err.Error() != tc.want {
			t.Errorf("%s: err = %v, want %q", js(t, tc.args), err, tc.want)
		}
	}
}

// A nested child of a struct is reachable as before: the address string of a Modbus address.
func TestTheAddressOfAModbusFlexAddressIsSetThroughItsChildSlot(t *testing.T) {
	op, _, _, rb := mbPlanAndReadback(t, mbSet("dataAddress/addressFormat", "modbus", "baja:Simple"),
		mbNodes())
	if want := `{"nm": "s", "h": "9", "n": "dataAddress/addressFormat", "b": {"nm": "p", "t": "modbusCore:AddressFormatEnum", "v": "modbus"}}`; op != want {
		t.Errorf("op\n got  %s\n want %s", op, want)
	}
	if rb[3] != "verified" {
		t.Errorf("readback = %v", rb)
	}
}

// An MCP client may turn a numeric-looking string into a JSON number: baja:Simple takes a whole
// number as its decimal string, so a Modbus register address "40014" is still sent as text.
func TestASimpleWholeNumberIsSentAsItsDecimalString(t *testing.T) {
	op, _, _, rb := mbPlanAndReadback(t, mbSet("dataAddress/address", pyjson.IntLiteral("40014"), "baja:Simple"),
		mbNodes("dataAddress/address", obj("t", "baja:String", "v", "40014")))
	if want := `{"nm": "s", "h": "9", "n": "dataAddress/address", "b": {"nm": "p", "t": "baja:String", "v": "40014"}}`; op != want {
		t.Errorf("op\n got  %s\n want %s", op, want)
	}
	if rb[3] != "verified" {
		t.Errorf("readback = %v", rb)
	}
	c := &loads{nodes: obj(mb, mbNodes())}
	if _, err := writeops.SetSlot.Plan(c, mbSet("dataAddress/address", pyjson.FloatNumber(1.5), "baja:Simple")); err == nil {
		t.Errorf("a fractional number was accepted")
	}
}

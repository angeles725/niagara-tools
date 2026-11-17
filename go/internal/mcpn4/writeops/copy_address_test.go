package writeops_test

import (
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/writeops"
)

// A Modbus client device allows one writable point per register: the station resets a second
// writable proxy at the same address to Hex:-1 (BModbusClientProxyExt.started/changed, log
// "Duplicate Writable point for register address"; proven live on a 4.15 lab station). copies[]
// gives a copy its own proxyExt/dataAddress inside the same batch.

// pointTree is a writable Modbus point as a load returns it (Gw under net in copyClient): the
// address in modbus format and regType carrying the readonly flag the station sets itself.
func pointTree(typ, address string) *pyjson.Object {
	addr := obj("nm", "p", "n", "dataAddress", "t", "modbusCore:FlexAddress", "s", []any{
		obj("nm", "p", "n", "addressFormat", "t", "modbusCore:AddressFormatEnum", "v", "modbus"),
		obj("nm", "p", "n", "address", "t", "baja:String", "v", address)})
	ext := obj("nm", "p", "n", "proxyExt", "h", "31", "t", "modbusCore:ModbusClientNumericProxyExt", "s", []any{
		obj("nm", "p", "n", "status", "t", "baja:Status", "v", "14"), addr, obj("nm", "p", "n", "regType", "f", "r")})
	return obj("h", "30", "t", typ, "s", []any{
		obj("nm", "p", "n", "out", "t", "baja:StatusNumeric", "s", []any{obj("nm", "p", "n", "value", "v", "0.0")}),
		ext})
}

func copyWith(names []any, copies ...any) *pyjson.Object {
	args := obj("parent_ord", net, "name", "Gw", "new_names", names)
	if len(copies) > 0 {
		args.Set("copies", copies)
	}
	return args
}

const dupNote = "'M3': a writable Modbus point copied onto its source's device uses the register address " +
	"Modbus:40002, and the device allows one writable point per register: the station resets the copy's " +
	"proxyExt/dataAddress to Hex:-1 (station log \"Duplicate Writable point for register address\"). " +
	"Give each copy its own address with copies[].data_address."

func TestACopyOverrideSetsEachCopysOwnAddressAndNotesTheDuplicates(t *testing.T) {
	p, err := writeops.CopyComponent.Plan(copyClient(t, pointTree("control:NumericWritable", "40002")),
		copyWith([]any{"M2", "M3"}, obj("name", "M2", "data_address", "40010")))
	if err != nil {
		t.Fatal(err)
	}
	addr := `{"nm": "p", "t": "modbusCore:FlexAddress", "n": "dataAddress", "s": [` +
		`{"nm": "p", "t": "modbusCore:AddressFormatEnum", "n": "addressFormat", "v": "modbus"}, ` +
		`{"nm": "p", "t": "baja:String", "n": "address", "v": "%s"}]}`
	if op := js(t, p.Ops[0]); !strings.Contains(op, strings.Replace(addr, "%s", "40010", 1)) {
		t.Errorf("M2 op lacks its own address: %s", op)
	}
	if op := js(t, p.Ops[1]); !strings.Contains(op, strings.Replace(addr, "%s", "40002", 1)) {
		t.Errorf("M3 op lacks the source address: %s", op)
	}
	if notes := js(t, p.Notes); !strings.Contains(notes, strings.ReplaceAll(dupNote, `"`, `\"`)) {
		t.Errorf("notes %s\n lack %s", notes, dupNote)
	}
}

func TestACopyOverrideCanChangeTheAddressFormat(t *testing.T) {
	p, err := writeops.CopyComponent.Plan(copyClient(t, pointTree("control:NumericWritable", "40002")),
		copyWith([]any{"M2"}, obj("name", "M2", "data_address", "0x0A", "address_format", "hex")))
	if err != nil {
		t.Fatal(err)
	}
	want := `{"nm": "p", "t": "modbusCore:AddressFormatEnum", "n": "addressFormat", "v": "hex"}, ` +
		`{"nm": "p", "t": "baja:String", "n": "address", "v": "0x0A"}`
	if op := js(t, p.Ops[0]); !strings.Contains(op, want) {
		t.Errorf("op %s\n lacks %s", op, want)
	}
	if notes := js(t, p.Notes); strings.Contains(notes, "Duplicate Writable") {
		t.Errorf("notes = %s", notes)
	}
}

// A source point with its proxy at the default address carries no dataAddress: the override adds it.
func TestACopyOverrideAddsTheAddressOfAProxyAtItsDefault(t *testing.T) {
	tree := pointTree("control:NumericWritable", "40002")
	ext := child(tree, "proxyExt")
	ext.Set("s", []any{child(ext, "status")})
	p, err := writeops.CopyComponent.Plan(copyClient(t, tree), copyWith([]any{"M2"}, obj("name", "M2", "data_address", "7")))
	if err != nil {
		t.Fatal(err)
	}
	want := `{"nm": "p", "t": "modbusCore:FlexAddress", "n": "dataAddress", "s": [{"nm": "p", "t": "baja:String", "n": "address", "v": "7"}]}`
	if op := js(t, p.Ops[0]); !strings.Contains(op, want) {
		t.Errorf("op %s\n lacks %s", op, want)
	}
}

// Two copies given the same address collide with each other: the second is noted.
func TestACopyNotesTwoCopiesGivenTheSameAddress(t *testing.T) {
	p, err := writeops.CopyComponent.Plan(copyClient(t, pointTree("control:NumericWritable", "40002")),
		copyWith([]any{"M2", "M3"}, obj("name", "M2", "data_address", "40010"), obj("name", "M3", "data_address", "40010")))
	if err != nil {
		t.Fatal(err)
	}
	if notes := js(t, p.Notes); !strings.Contains(notes, "'M3': a writable Modbus point") || strings.Contains(notes, "'M2'") ||
		!strings.Contains(notes, "Modbus:40010") {
		t.Errorf("notes = %s", notes)
	}
}

// A read-only point is not subject to the duplicate rule: its copy is planned as before, no note.
func TestACopyOfANumericPointIsUnchangedAndNotNoted(t *testing.T) {
	tree := pointTree("control:NumericPoint", "40002")
	p, err := writeops.CopyComponent.Plan(copyClient(t, tree), copyWith([]any{"M3"}))
	if err != nil {
		t.Fatal(err)
	}
	want := `{"nm": "a", "h": "10", "n": "M3", "b": {"nm": "p", "t": "control:NumericPoint", "s": [` +
		`{"nm": "p", "t": "modbusCore:ModbusClientNumericProxyExt", "n": "proxyExt", "s": [` +
		`{"nm": "p", "t": "modbusCore:FlexAddress", "n": "dataAddress", "s": [` +
		`{"nm": "p", "t": "modbusCore:AddressFormatEnum", "n": "addressFormat", "v": "modbus"}, ` +
		`{"nm": "p", "t": "baja:String", "n": "address", "v": "40002"}]}, {"nm": "p", "n": "regType", "f": "r"}]}]}}`
	if op := js(t, p.Ops[0]); op != want {
		t.Errorf("op\n got  %s\n want %s", op, want)
	}
	if notes := js(t, p.Notes); strings.Contains(notes, "Duplicate Writable") {
		t.Errorf("notes = %s", notes)
	}
}

func TestACopyOverrideIsRefusedWhenItCannotApply(t *testing.T) {
	point := pointTree("control:NumericWritable", "40002")
	for _, tc := range []struct {
		tree *pyjson.Object
		args *pyjson.Object
		want string
	}{
		{gwTree(), copyWith([]any{"M2"}, obj("name", "M2", "data_address", "1")),
			"copies[].data_address needs a source with a proxyExt that declares dataAddress; 'Gw' has none"},
		{point, copyWith([]any{"M2"}, obj("name", "M9", "data_address", "1")),
			"copies[].name 'M9' is not one of new_names"},
		{point, copyWith([]any{"M2"}, obj("name", "M2", "data_address", "1"), obj("name", "M2", "data_address", "2")),
			"copies[].name 'M2' is given twice"},
		{point, copyWith([]any{"M2"}, obj("name", "M2", "data_address", " ")),
			"copies[].data_address of 'M2' must be a non-empty address"},
	} {
		if _, err := writeops.CopyComponent.Plan(copyClient(t, tc.tree), tc.args); err == nil || err.Error() != tc.want {
			t.Errorf("err = %v, want %s", err, tc.want)
		}
	}
}

// readBackPoint plans copies of a writable point, lets the station hold held as copy nn and
// returns the read-back's observed list and verdict.
func readBackPoint(t *testing.T, args *pyjson.Object, nn string, held *pyjson.Object) (string, string) {
	t.Helper()
	c := copyClient(t, pointTree("control:NumericWritable", "40002"))
	p, err := writeops.CopyComponent.Plan(c, args)
	if err != nil {
		t.Fatal(err)
	}
	replies := []any{[]any{obj("nn", nn)}}
	inverse, err := writeops.CopyComponent.Inverse(p, replies)
	if err != nil {
		t.Fatal(err)
	}
	c.nodes.Set(net, obj("", obj("h", "10"), "Gw", obj("t", "control:NumericWritable"), nn, obj("t", "control:NumericWritable")))
	c.nodes.Set(net+"/"+nn, obj("", held))
	_, _, observed, verdict, err := writeops.CopyComponent.Readback(c, args, p, replies, inverse)
	if err != nil {
		t.Fatal(err)
	}
	return js(t, observed), verdict
}

func TestACopyReadBackNamesTheDuplicateRegisterCause(t *testing.T) {
	reset := pointTree("control:NumericWritable", "-1")
	under(reset, "proxyExt/dataAddress/addressFormat").Set("v", "hex")
	observed, verdict := readBackPoint(t, copyWith([]any{"M3"}), "M3", reset)
	want := `"cause": "the station reset proxyExt/dataAddress to Hex:-1: another writable point on this device ` +
		`already uses register Modbus:40002 (\"Duplicate Writable point for register address\"); ` +
		`give the copy its own address with copies[].data_address or n4_set_slot"`
	if verdict != "mismatch" || !strings.Contains(observed, want) {
		t.Errorf("%s %s\n lacks %s", verdict, observed, want)
	}
}

func TestACopyReadBackVerifiesAnOverriddenAddress(t *testing.T) {
	observed, verdict := readBackPoint(t, copyWith([]any{"M2"}, obj("name", "M2", "data_address", "40010")),
		"M2", pointTree("control:NumericWritable", "40010"))
	if verdict != "verified" || strings.Contains(observed, "cause") {
		t.Errorf("%s %s", verdict, observed)
	}
}

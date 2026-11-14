package writeops_test

import (
	"os"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/writeops"
)

const net = "station:|slot:/Drivers/ModbusAsyncNetwork"

// gwTree is a Modbus device as a load returns it: runtime slots (status, health, a point's out,
// readValue) next to configuration five levels deep (the point address) and one internal link.
func gwTree() *pyjson.Object {
	addr := obj("nm", "p", "n", "dataAddress", "t", "modbusCore:FlexAddress", "s", []any{
		obj("nm", "p", "n", "addressFormat", "t", "modbusCore:AddressFormatEnum", "v", "modbus"),
		obj("nm", "p", "n", "address", "t", "baja:String", "v", "40005")})
	ext := obj("nm", "p", "n", "proxyExt", "h", "31", "t", "modbusCore:ModbusClientNumericProxyExt", "s", []any{
		obj("nm", "p", "n", "status", "t", "baja:Status", "v", "14"),
		obj("nm", "p", "n", "readValue", "t", "baja:StatusNumeric", "s", []any{obj("nm", "p", "n", "value", "v", "0.0")}),
		obj("nm", "p", "n", "dataType", "t", "modbusCore:DataTypeEnum", "v", "signedInteger"),
		addr})
	point := obj("nm", "p", "n", "Setpoint", "h", "30", "t", "control:NumericWritable", "s", []any{
		obj("nm", "p", "n", "out", "t", "baja:StatusNumeric", "s", []any{obj("nm", "p", "n", "value", "v", "0.0")}),
		obj("nm", "p", "n", "fallback", "t", "baja:StatusNumeric", "s", []any{obj("nm", "p", "n", "value", "v", "200.0")}),
		ext})
	points := obj("nm", "p", "n", "points", "h", "21", "t", "modbusCore:ModbusClientPointDeviceExt", "s", []any{point})
	return obj("h", "20", "t", "modbusAsync:ModbusAsyncDevice", "s", []any{
		obj("nm", "p", "n", "status", "t", "baja:Status", "v", "4"),
		obj("nm", "p", "n", "health", "t", "driver:PingHealth", "v", "Fail pingFail"),
		obj("nm", "p", "n", "deviceAddress", "t", "baja:Integer", "v", "2"),
		points})
}

// copyClient loads tree as the device Gw under net and answers the registry channel with the
// contracts of testdata/copy-contracts.json (the real flags of each slot).
func copyClient(t *testing.T, tree *pyjson.Object, parentKids ...any) *typed {
	t.Helper()
	parent := obj("", obj("h", "10", "t", "modbusAsync:ModbusAsyncNetwork"), "Gw", obj("h", "20", "t", "modbusAsync:ModbusAsyncDevice"))
	for i := 0; i < len(parentKids); i += 2 {
		parent.Set(parentKids[i].(string), parentKids[i+1])
	}
	return &typed{loads: loads{nodes: obj(net, parent, net+"/Gw", obj("", tree))}, types: fixture(t, "copy-contracts.json")}
}

// fixture is testdata/name parsed.
func fixture(t *testing.T, name string) any {
	t.Helper()
	data, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	v, err := pyjson.Unmarshal(data)
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func copyArgs(names ...any) *pyjson.Object {
	return obj("parent_ord", net, "name", "Gw", "new_names", names)
}

func TestACopyAddsTheConfiguredSubtreeOncePerNewNameWithoutRuntimeSlots(t *testing.T) {
	p, err := writeops.CopyComponent.Plan(copyClient(t, gwTree()), copyArgs("Gw3", "Gw4"))
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Ops) != 2 || len(p.Inverse) != 2 {
		t.Fatalf("ops %d inverse %d, want 2 and 2", len(p.Ops), len(p.Inverse))
	}
	want := `{"nm": "a", "h": "10", "n": "Gw3", "b": {"nm": "p", "t": "modbusAsync:ModbusAsyncDevice", "s": [` +
		`{"nm": "p", "t": "baja:Integer", "n": "deviceAddress", "v": "2"}, ` +
		`{"nm": "p", "t": "modbusCore:ModbusClientPointDeviceExt", "n": "points", "s": [` +
		`{"nm": "p", "t": "control:NumericWritable", "n": "Setpoint", "s": [` +
		`{"nm": "p", "t": "baja:StatusNumeric", "n": "fallback", "s": [{"nm": "p", "n": "value", "v": "200.0"}]}, ` +
		`{"nm": "p", "t": "modbusCore:ModbusClientNumericProxyExt", "n": "proxyExt", "s": [` +
		`{"nm": "p", "t": "modbusCore:DataTypeEnum", "n": "dataType", "v": "signedInteger"}, ` +
		`{"nm": "p", "t": "modbusCore:FlexAddress", "n": "dataAddress", "s": [` +
		`{"nm": "p", "t": "modbusCore:AddressFormatEnum", "n": "addressFormat", "v": "modbus"}, ` +
		`{"nm": "p", "t": "baja:String", "n": "address", "v": "40005"}]}]}]}]}]}}`
	if got := js(t, p.Ops[0]); got != want {
		t.Errorf("op\n got  %s\n want %s", got, want)
	}
	if got := js(t, p.Inverse[1]); got != `{"nm": "v", "h": "10", "n": "Gw4"}` {
		t.Errorf("inverse = %s", got)
	}
}

func TestACopyRefusesBadNamesAndAMissingSource(t *testing.T) {
	for _, tc := range []struct {
		args *pyjson.Object
		want string
	}{
		{copyArgs(), "new_names must list 1 to 30 distinct plain slot names"},
		{copyArgs("Gw3", "Gw3"), "new_names must list 1 to 30 distinct plain slot names"},
		{copyArgs("a/b"), "new_name must be a plain slot name (letters, digits, underscore), got 'a/b'"},
		{copyArgs("Gw"), "a child named 'Gw' already exists under " + net},
		{obj("parent_ord", net, "name", "Ghost", "new_names", []any{"X"}), "component 'Ghost' not found under " + net},
	} {
		if _, err := writeops.CopyComponent.Plan(copyClient(t, gwTree()), tc.args); err == nil || err.Error() != tc.want {
			t.Errorf("%s: err = %v, want %q", js(t, tc.args), err, tc.want)
		}
	}
}

// A subtree deeper than the copy depth is refused: a partial copy would look complete.
func TestACopyRefusesASubtreeDeeperThanItCanCapture(t *testing.T) {
	tree := gwTree()
	deep := obj("nm", "p", "n", "L1", "t", "baja:Folder")
	cur := deep
	for i := 2; i <= writeops.CopyDepth+1; i++ {
		next := obj("nm", "p", "n", "L"+string(rune('0'+i)), "t", "baja:Folder")
		cur.Set("s", []any{next})
		cur = next
	}
	kids, _ := tree.Get("s")
	tree.Set("s", append(kids.([]any), deep))
	_, err := writeops.CopyComponent.Plan(copyClient(t, tree), copyArgs("Gw3"))
	if err == nil || !strings.Contains(err.Error(), "deeper than the copy depth") {
		t.Errorf("err = %v", err)
	}
}

// Links inside the source are not copied, and the plan says so.
func TestACopyNotesThatLinksAreNotCopied(t *testing.T) {
	tree := gwTree()
	kids, _ := tree.Get("s")
	tree.Set("s", append(kids.([]any), obj("nm", "p", "n", "Link", "t", "baja:Link", "s", []any{
		obj("nm", "p", "n", "sourceOrd", "v", "h:30"), obj("nm", "p", "n", "sourceSlotName", "v", "out"),
		obj("nm", "p", "n", "targetSlotName", "v", "x")})))
	p, err := writeops.CopyComponent.Plan(copyClient(t, tree), copyArgs("Gw3"))
	if err != nil {
		t.Fatal(err)
	}
	if got := js(t, p.Notes); !strings.Contains(got, "1 link(s) inside 'Gw' are not copied") {
		t.Errorf("notes = %s", got)
	}
	if strings.Contains(js(t, p.Ops[0]), "baja:Link") {
		t.Errorf("a link was copied: %s", js(t, p.Ops[0]))
	}
}

// The read-back finds each copy under its assigned name with the source type; a missing copy is a mismatch.
func TestACopyReadBackChecksEveryCopyUnderItsAssignedName(t *testing.T) {
	c := copyClient(t, gwTree())
	args := copyArgs("Gw3", "Gw4")
	p, err := writeops.CopyComponent.Plan(c, args)
	if err != nil {
		t.Fatal(err)
	}
	replies := []any{[]any{obj("nn", "Gw3")}, []any{obj("nn", "Gw4")}}
	inverse, err := writeops.CopyComponent.Inverse(p, replies)
	if err != nil {
		t.Fatal(err)
	}
	c.nodes.Set(net, obj("", obj("h", "10"), "Gw", obj("t", "modbusAsync:ModbusAsyncDevice"),
		"Gw3", obj("t", "modbusAsync:ModbusAsyncDevice"), "Gw4", obj("t", "modbusAsync:ModbusAsyncDevice")))
	c.nodes.Set(net+"/Gw3", obj("", gwTree()))
	c.nodes.Set(net+"/Gw4", obj("", gwTree()))
	if _, _, _, v, err := writeops.CopyComponent.Readback(c, args, p, replies, inverse); err != nil || v != "verified" {
		t.Errorf("verdict = %q, %v", v, err)
	}
	c.nodes.Set(net, obj("", obj("h", "10"), "Gw3", obj("t", "modbusAsync:ModbusAsyncDevice")))
	if _, _, _, v, _ := writeops.CopyComponent.Readback(c, args, p, replies, inverse); v != "mismatch" {
		t.Errorf("verdict with Gw4 missing = %q, want mismatch", v)
	}
}

// absoluteAddress is derived by the driver and is not copied either.
// A load lists actions (upload, download) and frozen slots at their default (regType) by name
// only, without a type; a copy leaves them out, because an add op without a type is not a value
// and the copy gets those slots from its type anyway.
func TestACopyLeavesOutChildrenTheLoadListsWithoutAType(t *testing.T) {
	tree := gwTree()
	kids, _ := tree.Get("s")
	tree.Set("s", append(kids.([]any), obj("nm", "p", "n", "upload"), obj("nm", "p", "n", "download")))
	// absoluteAddress lives on the proxy extension, whose type declares it readonly and transient.
	addKids(under(tree, "points/Setpoint/proxyExt"), obj("nm", "p", "n", "absoluteAddress", "t", "modbusCore:FlexAddress", "s", []any{
		obj("nm", "p", "n", "address", "t", "baja:String", "v", "40005")}))
	p, err := writeops.CopyComponent.Plan(copyClient(t, tree), copyArgs("Gw3"))
	if err != nil {
		t.Fatal(err)
	}
	if got := js(t, p.Ops[0]); strings.Contains(got, `"upload"`) || strings.Contains(got, `"download"`) ||
		strings.Contains(got, "absoluteAddress") {
		t.Errorf("a typeless child was copied: %s", got)
	}
}

// An alarm extension's state slots are runtime (their contract flags, testdata/copy-contracts.json): timeInCurrentState ticks every few seconds, so
// copying it made every dry run a different plan and the confirmation token never matched
// (client A, 2026-11-13). alarmState, ackedTransitions and the transition timestamps are the
// source's alarm history, not configuration. Config such as offnormalAlgorithm/alarmValue stays.
func TestACopyLeavesOutAlarmExtensionState(t *testing.T) {
	tree := gwTree()
	kids, _ := tree.Get("s")
	tree.Set("s", append(kids.([]any), obj("nm", "p", "n", "AlarmExt", "t", "alarm:AlarmSourceExt", "s", []any{
		obj("nm", "p", "n", "timeInCurrentState", "t", "baja:RelTime", "v", "30018"),
		obj("nm", "p", "n", "alarmState", "t", "alarm:AlarmState", "v", "offnormal"),
		obj("nm", "p", "n", "ackedTransitions", "t", "alarm:AlarmTransitionBits", "v", "0"),
		obj("nm", "p", "n", "toOffnormalTimes", "t", "alarm:AlarmTimestamps", "v", "x"),
		obj("nm", "p", "n", "toFaultTimes", "t", "alarm:AlarmTimestamps", "v", "x"),
		obj("nm", "p", "n", "offnormalAlgorithm", "t", "alarm:BooleanChangeOfStateAlgorithm", "s", []any{
			obj("nm", "p", "n", "alarmValue", "t", "baja:Boolean", "v", "true")})})))
	p, err := writeops.CopyComponent.Plan(copyClient(t, tree), copyArgs("Gw3"))
	if err != nil {
		t.Fatal(err)
	}
	got := js(t, p.Ops[0])
	for _, n := range []string{"timeInCurrentState", "alarmState", "ackedTransitions", "toOffnormalTimes", "toFaultTimes"} {
		if strings.Contains(got, `"`+n+`"`) {
			t.Errorf("runtime slot %s was copied: %s", n, got)
		}
	}
	if !strings.Contains(got, `"alarmValue"`) || !strings.Contains(got, "BooleanChangeOfStateAlgorithm") {
		t.Errorf("alarm configuration was dropped: %s", got)
	}
}

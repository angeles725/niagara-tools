package readtools_test

// Go twins of the test_server.TestReadSlots tests that call tools_read._slot_entry or inject
// the clock (go_server.go_twin("readtools", ...)).

import (
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/session"
)

// serve is a connected server whose station answers a load of ord with the nodes in tree.
func serve(t *testing.T, ord, tree string) *readtoolstest.Server {
	t.Helper()
	nodes, err := pyjson.Unmarshal([]byte(tree))
	if err != nil {
		t.Fatal(err)
	}
	s := readtoolstest.NewServer(readtoolstest.NewStation(), session.Config{})
	s.Connect(t)
	s.Station.Trees = map[string]*pyjson.Object{ord: nodes.(*pyjson.Object)}
	return s
}

// slots is n4_read_slots of ord: each slot entry by name.
func slots(t *testing.T, s *readtoolstest.Server, ord string) map[string]*pyjson.Object {
	t.Helper()
	text, isErr, out := s.Call(t, "n4_read_slots", "ord", ord)
	if isErr {
		t.Fatalf("n4_read_slots: %s", text)
	}
	list, _ := out.Get("slots")
	byName := map[string]*pyjson.Object{}
	for _, e := range list.([]any) {
		name, _ := e.(*pyjson.Object).Get("name")
		byName[name.(string)] = e.(*pyjson.Object)
	}
	return byName
}

func want(t *testing.T, entry *pyjson.Object, kv ...any) {
	t.Helper()
	for i := 0; i < len(kv); i += 2 {
		if got, _ := entry.Get(kv[i].(string)); got != kv[i+1] {
			t.Errorf("%s = %s, want %s", kv[i], pyjson.Repr(got), pyjson.Repr(kv[i+1]))
		}
	}
}

const ord = "station:|slot:/Folder/Ext"

func TestComplexSlotWithoutAValueFallsBackToItsDisplayString(t *testing.T) {
	s := serve(t, ord, `{"": {"t": "modbusCore:ModbusClientNumericProxyExt"},
		"dataAddress": {"n": "dataAddress", "t": "modbusCore:FlexAddress", "d": "Decimal:302"},
		"dataAddress/address": {"n": "address", "t": "baja:String", "v": "302", "d": "302"},
		"address": {"n": "address", "t": "bacnet:BacnetAddress", "d": "1:<device-ip>:47808"}}`)
	got := slots(t, s, ord)
	want(t, got["dataAddress"], "value", "Decimal:302", "value_display", "Decimal:302")
	want(t, got["address"], "value", "1:<device-ip>:47808")
}

func TestADecodedValueIsKeptAndItsDisplayRidesAlong(t *testing.T) {
	s := serve(t, ord, `{"": {}, "conversion": {"t": "driver:LinearConversion", "v": "0.1;0.0", "d": "Linear *0.10"}}`)
	want(t, slots(t, s, ord)["conversion"], "value", "0.1;0.0", "value_display", "Linear *0.10")
}

func TestReadSlotsAndNavigateReportElapsedMs(t *testing.T) {
	s := serve(t, ord, `{"": {}, "out": {"t": "baja:StatusNumeric"}}`)
	ticks := []float64{10.0, 10.25, 20.0, 20.5}
	s.Ctx.Clock = func() float64 { tick := ticks[0]; ticks = ticks[1:]; return tick }
	for _, call := range [][]any{{"n4_read_slots", "ord", ord}, {"n4_navigate"}} {
		_, _, out := s.Call(t, call[0].(string), call[1:]...)
		elapsed, _ := out.Get("elapsed_ms")
		if ms := map[string]int{"n4_read_slots": 250, "n4_navigate": 500}[call[0].(string)]; elapsed != ms {
			t.Errorf("%s elapsed_ms = %v, want %d", call[0], elapsed, ms)
		}
	}
}

// rfxFlags is the status_flags of entry as a Python repr.
func rfxFlags(entry *pyjson.Object) string {
	flags, _ := entry.Get("status_flags")
	return pyjson.Repr(flags)
}

// A slot whose type is baja:Status carries its bits in "v", with no value/status children.
const rfxStatusTree = `{"": {"t": "control:NumericWritable"},
	"faulted": {"n": "faulted", "t": "baja:Status", "v": "2", "d": "{fault}"},
	"alarmed": {"n": "alarmed", "t": "baja:Status", "v": "88", "d": "{alarm,unackedAlarm}"},
	"healthy": {"n": "healthy", "t": "baja:Status", "v": "0", "d": "{ok}"},
	"omitted": {"n": "omitted", "t": "baja:Status"},
	"nulled": {"n": "nulled", "t": "baja:Status", "v": "40;units=x"},
	"out": {"n": "out", "t": "baja:StatusNumeric"},
	"out/value": {"n": "value", "t": "baja:Double", "v": "5.5"},
	"out/status": {"n": "status", "t": "baja:Status", "v": "8"}}`

func TestAStatusValueSlotNamesItsFlagsAndIsNotOkWhenAnyIsSet(t *testing.T) {
	got := slots(t, serve(t, ord, rfxStatusTree), ord)
	want(t, got["faulted"], "value", "2", "status", "2", "status_ok", false, "status_null", false, "value_display", "{fault}")
	want(t, got["alarmed"], "value", "88", "status", "88", "status_ok", false, "status_null", false)
	want(t, got["nulled"], "status", "40;units=x", "status_ok", false, "status_null", true)
	for name, flags := range map[string]string{"faulted": "['fault']", "alarmed": "['alarm', 'unackedAlarm']", "nulled": "['null']"} {
		if f := rfxFlags(got[name]); f != flags {
			t.Errorf("%s status_flags = %s, want %s", name, f, flags)
		}
	}
}

func TestAStatusValueSlotWithNoFlagIsOk(t *testing.T) {
	got := slots(t, serve(t, ord, rfxStatusTree), ord)
	for _, name := range []string{"healthy", "omitted"} { // a station omits the default "0"
		want(t, got[name], "value", "0", "status", "0", "status_ok", true, "status_null", false)
		if f := rfxFlags(got[name]); f != "[]" {
			t.Errorf("%s status_flags = %s", name, f)
		}
	}
}

func TestAStatusComplexKeepsItsValueStatusAndFlagsReply(t *testing.T) {
	out := slots(t, serve(t, ord, rfxStatusTree), ord)["out"]
	if got := pyjson.Repr(out); got != "{'name': 'out', 'type': 'baja:StatusNumeric', 'value': '5.5', 'status': '8', 'status_ok': False, 'status_null': False}" {
		t.Errorf("out = %s", got)
	}
}

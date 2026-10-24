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
	st := readtoolstest.NewStation()
	st.Trees = map[string]*pyjson.Object{ord: nodes.(*pyjson.Object)}
	s := readtoolstest.NewServer(st, session.Config{})
	s.Connect(t)
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

package writeops_test

import (
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/writeops"
)

const pts = "station:|slot:/Drivers/ModbusAsyncNetwork/Gw/points"

// proxy_ext creates a point with its driver extension in the same add: a read-only Modbus point
// is a NumericPoint with a ModbusClientNumericProxyExt, never a writable copy.
func TestProxyExtCreatesThePointWithItsDriverExtension(t *testing.T) {
	c := &loads{nodes: obj(pts, obj("", obj("h", "21")))}
	args := obj("parent_ord", pts, "name", "RoomTemp", "type", "control:NumericPoint",
		"proxy_ext", "modbusCore:ModbusClientNumericProxyExt")
	p, err := writeops.CreateComponent.Plan(c, args)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"nm": "a", "h": "21", "n": "RoomTemp", "b": {"nm": "p", "t": "control:NumericPoint", "s": [` +
		`{"nm": "p", "n": "proxyExt", "t": "modbusCore:ModbusClientNumericProxyExt"}]}}`
	if got := js(t, p.Ops[0]); got != want {
		t.Errorf("op\n got  %s\n want %s", got, want)
	}
	inverse, _ := writeops.CreateComponent.Inverse(p, []any{[]any{obj("nn", "RoomTemp")}})
	c.nodes.Set(pts, obj("", obj("h", "21"), "RoomTemp", obj("t", "control:NumericPoint"),
		"RoomTemp/proxyExt", obj("t", "modbusCore:ModbusClientNumericProxyExt")))
	if _, _, obs, v, err := writeops.CreateComponent.Readback(c, args, p, []any{[]any{"r"}}, inverse); err != nil || v != "verified" {
		t.Errorf("verdict = %q (%s), %v", v, js(t, obs), err)
	}
	c.nodes.Set(pts, obj("", obj("h", "21"), "RoomTemp", obj("t", "control:NumericPoint"),
		"RoomTemp/proxyExt", obj("t", "driver:NullProxyExt")))
	if _, _, _, v, _ := writeops.CreateComponent.Readback(c, args, p, []any{[]any{"r"}}, inverse); v != "mismatch" {
		t.Errorf("verdict with the null ext = %q, want mismatch", v)
	}
}

func TestProxyExtMustBeATypeSpec(t *testing.T) {
	c := &loads{nodes: obj(pts, obj("", obj("h", "21")))}
	_, err := writeops.CreateComponent.Plan(c, obj("parent_ord", pts, "name", "X", "type", "control:NumericPoint", "proxy_ext", "Modbus"))
	if want := "proxy_ext must be a type spec module:Type, got 'Modbus'"; err == nil || err.Error() != want {
		t.Errorf("err = %v, want %q", err, want)
	}
}

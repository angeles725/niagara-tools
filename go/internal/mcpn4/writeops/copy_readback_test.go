package writeops_test

import (
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/writeops"
)

// readBackCopy plans one copy Gw3, executes it as named nn, lets the station hold copy as nn and
// returns the read-back's observed list and verdict.
func readBackCopy(t *testing.T, nn string, copy *pyjson.Object) (string, string) {
	t.Helper()
	c := copyClient(t, gwTree())
	args := copyArgs("Gw3")
	p, err := writeops.CopyComponent.Plan(c, args)
	if err != nil {
		t.Fatal(err)
	}
	replies := []any{[]any{obj("nn", nn)}}
	inverse, err := writeops.CopyComponent.Inverse(p, replies)
	if err != nil {
		t.Fatal(err)
	}
	if got := js(t, inverse); got != `[{"nm": "v", "h": "10", "n": "`+nn+`"}]` {
		t.Errorf("inverse = %s", got)
	}
	c.nodes.Set(net, obj("", obj("h", "10"), "Gw", obj("t", "modbusAsync:ModbusAsyncDevice"),
		nn, obj("t", "modbusAsync:ModbusAsyncDevice")))
	c.nodes.Set(net+"/"+nn, obj("", copy))
	_, _, observed, verdict, err := writeops.CopyComponent.Readback(c, args, p, replies, inverse)
	if err != nil {
		t.Fatal(err)
	}
	return js(t, observed), verdict
}

// (h) The read-back compares the whole configured subtree, ignoring handles, child order and
// runtime slots (a copy shows its own status and health).
func TestACopyReadBackComparesTheWholeConfiguredSubtree(t *testing.T) {
	same := gwTree()
	same.Set("h", "77")
	child(same, "status").Set("v", "0")
	kids, _ := same.Get("s")
	list := kids.([]any)
	same.Set("s", []any{list[3], list[2], list[1], list[0]})
	observed, verdict := readBackCopy(t, "Gw3", same)
	if verdict != "verified" || observed != `[{"name": "Gw3", "type": "modbusAsync:ModbusAsyncDevice", "differences": []}]` {
		t.Errorf("equal copy: %s %s", verdict, observed)
	}

	changed := gwTree()
	under(changed, "points/Setpoint/proxyExt/dataAddress/address").Set("v", "40006")
	changed.Set("s", []any{child(changed, "status"), child(changed, "points")}) // deviceAddress missing
	observed, verdict = readBackCopy(t, "Gw3", changed)
	if verdict != "mismatch" || !strings.Contains(observed,
		`"differences": ["deviceAddress", "points/Setpoint/proxyExt/dataAddress/address"]`) {
		t.Errorf("changed copy: %s %s", verdict, observed)
	}
}

// (i) The station may rename on collision: the inverse and the read-back use its name.
func TestACopyReadBackLooksUpTheStationAssignedName(t *testing.T) {
	observed, verdict := readBackCopy(t, "Gw5", gwTree())
	if verdict != "verified" || !strings.Contains(observed, `"name": "Gw5"`) {
		t.Errorf("renamed copy: %s %s", verdict, observed)
	}
}

// (i) new_names must be a list of 1 to 30 names; the source must be a valid name.
func TestACopyRefusesTooManyOrMalformedNames(t *testing.T) {
	many := []any{}
	for i := 0; i < 31; i++ {
		many = append(many, "Gw"+string(rune('a'+i%26))+string(rune('a'+i/26)))
	}
	refuse := "new_names must list 1 to 30 distinct plain slot names"
	for _, tc := range []struct {
		args *pyjson.Object
		want string
	}{
		{copyArgs(many...), refuse},
		{obj("parent_ord", net, "name", "Gw", "new_names", "Gw3"), refuse},
		{obj("parent_ord", net, "name", "a/b", "new_names", []any{"Gw3"}), "name must be a plain slot name"},
	} {
		_, err := writeops.CopyComponent.Plan(copyClient(t, gwTree()), tc.args)
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: err = %v, want %q", js(t, tc.args), err, tc.want)
		}
	}
}

package tags

import (
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// Every allowed type checks its value before it is encoded; a type outside ValueTypes is refused.
func TestBodyChecksEachTypeAndRefusesNonValueTypes(t *testing.T) {
	ok := map[string][2]any{
		"baja:Float":       {pyjson.IntLiteral("2"), "2.0"},
		"baja:Long":        {pyjson.IntLiteral("9007199254740993"), "9007199254740993"},
		"baja:DynamicEnum": {"3@{a=1,b=3}", "3@{a=1,b=3}"},
		"baja:Integer":     {pyjson.IntLiteral("-4"), "-4"},
		"baja:Boolean":     {true, "true"},
		"baja:RelTime":     {pyjson.IntLiteral("1000"), "1000"},
	}
	for typ, c := range ok {
		b, err := Body(typ, c[0], nil)
		if err != nil || get(b, "v") != c[1] || get(b, "t") != typ {
			t.Errorf("%s %v: %v %v", typ, c[0], b, err)
		}
	}
	if b, err := Body("baja:DynamicEnum", pyjson.IntLiteral("2"), nil); err != nil || get(b, "v") != "2" {
		t.Errorf("enum ordinal: %v %v", b, err)
	}
	for typ, v := range map[string]any{"baja:Float": "abc", "baja:Long": "1", "baja:Long ": pyjson.IntLiteral("18446744073709551616"), "baja:DynamicEnum": "on",
		"baja:Integer": pyjson.IntLiteral("4294967296"), "baja:String": pyjson.IntLiteral("1")} {
		if _, err := Body(strings.TrimSpace(typ), v, nil); err == nil {
			t.Errorf("%s %v accepted", typ, v)
		}
	}
	for _, typ := range []string{"baja:Folder", "baja:Relation", "nonsense", "baja:StatusNumeric"} {
		if _, err := Body(typ, "x", nil); err == nil || !strings.Contains(plan.PyStr(err), "not a tag value type") {
			t.Errorf("%s: %v", typ, err)
		}
	}
}

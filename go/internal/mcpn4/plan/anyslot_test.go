package plan

import (
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// anySlotTypes answers a fixed registry reply.
type anySlotTypes struct{ reply any }

func (a anySlotTypes) LoadTypes([]string) (any, error) { return a.reply, nil }

// AnySlot finds a declared action or topic, which Slot (properties only) does not, walks the
// supertypes like it, and keeps Slot's answer for a property.
func TestAnySlotFindsADeclaredSlotOfAnyKindUpTheSupertypes(t *testing.T) {
	types, err := pyjson.Unmarshal([]byte(`{
		"acme:Pump": {"p": "acme:Base", "c": [{"st": "p", "n": "speed", "f": "o", "v": {"t": "baja:Double", "v": "0.0"}}]},
		"acme:Base": {"c": [{"st": "a", "n": "start", "f": "c"}, {"st": "t", "n": "tripped"}]}}`))
	if err != nil {
		t.Fatal(err)
	}
	ct := NewContracts(anySlotTypes{types})
	for _, tc := range []struct{ name, flags string }{{"speed", "o"}, {"start", "c"}, {"tripped", ""}} {
		slot, err := ct.AnySlot("acme:Pump", tc.name)
		if err != nil || slot == nil {
			t.Fatalf("%s: slot %v, err %v", tc.name, slot, err)
		}
		if f, _ := field(slot, "f").(string); f != tc.flags {
			t.Errorf("%s: flags %q, want %q", tc.name, f, tc.flags)
		}
	}
	if slot, err := ct.AnySlot("acme:Pump", "nope"); slot != nil || err != nil {
		t.Errorf("an undeclared name: slot %v, err %v", slot, err)
	}
	if slot, _ := ct.Slot("acme:Pump", "start"); slot != nil { // unchanged: properties only
		t.Errorf("Slot found the action: %v", slot)
	}
	if slot, _ := ct.Slot("acme:Pump", "speed"); slot == nil {
		t.Error("Slot no longer finds the property")
	}
}

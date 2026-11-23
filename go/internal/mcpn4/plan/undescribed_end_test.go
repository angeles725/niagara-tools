package plan

import (
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// UndescribedEnd flags a type the station answered with an error entry ({"isErr": true, ...}, what a 4.15
// station says of a type it does not know), wherever it sits in the chain and whatever its module: unlike a
// baja type answered with {}, which says nothing and is not flagged.
func TestUndescribedEndFlagsAnErrorEntryEvenOfABajaType(t *testing.T) {
	types, err := pyjson.Unmarshal([]byte(`{
		"acme:Pump":     {"p": "acme:Base", "c": []},
		"acme:Base":     {"p": "baja:Component", "c": []},
		"acme:Ghost":    {"isErr": true, "m": "acme:Ghost"},
		"acme:Lost":     {"p": "acme:Ghost", "c": []},
		"acme:Empty":    {},
		"baja:Gone":     {"isErr": true, "m": "baja:Gone"},
		"baja:Struct":   {},
		"baja:Component": {}}`))
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ name, spec, want string }{
		{"a chain described to a baja type answered with {}", "acme:Pump", ""},
		{"a baja type answered with {} is no end", "baja:Struct", ""},
		{"an error entry of the type itself", "acme:Ghost", "acme:Ghost"},
		{"an error entry up the chain", "acme:Lost", "acme:Ghost"},
		{"an error entry of a baja type is flagged", "baja:Gone", "baja:Gone"},
		{"a non-baja type answered with {}", "acme:Empty", "acme:Empty"},
	} {
		got, err := NewContracts(anySlotTypes{types}).UndescribedEnd(tc.spec)
		if err != nil || got != tc.want {
			t.Errorf("%s: UndescribedEnd(%q) = %q, %v; want %q", tc.name, tc.spec, got, err, tc.want)
		}
	}
}

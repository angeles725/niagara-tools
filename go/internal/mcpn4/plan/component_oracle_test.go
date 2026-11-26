package plan

import (
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

const oracleRegistry = `{
	"acme:Pump": {"p": "acme:Base"}, "acme:Base": {"p": "baja:Component"}, "baja:Component": {},
	"alarm:LimitEnable": {"p": "baja:Struct"}, "baja:Struct": {},
	"acme:Bare": {}, "baja:Double": {}}`

func oracleOver(t *testing.T, reply string) *RegistryOracle {
	t.Helper()
	types, err := pyjson.Unmarshal([]byte(reply))
	if err != nil {
		t.Fatal(err)
	}
	return NewRegistryOracle(NewContracts(anySlotTypes{types}))
}

// B1200-G3: a type is a component when the registry's supertype chain reaches baja:Component and a
// value when it is described completely without reaching it; the naming rule would call both
// components (every type outside baja).
func TestTheRegistryOracleTellsAComponentFromAStruct(t *testing.T) {
	o := oracleOver(t, oracleRegistry)
	for _, tc := range []struct {
		spec          string
		yes, known    bool
		structByChain bool
	}{
		{"acme:Pump", true, true, false},
		{"alarm:LimitEnable", false, true, true},
	} {
		if yes, known := o.IsComponent(tc.spec); yes != tc.yes || known != tc.known {
			t.Errorf("%s: IsComponent = %v, %v; want %v, %v", tc.spec, yes, known, tc.yes, tc.known)
		}
		if got := o.IsStruct(tc.spec); got != tc.structByChain {
			t.Errorf("%s: IsStruct = %v, want %v", tc.spec, got, tc.structByChain)
		}
	}
	if !box.IsComponentType("alarm:LimitEnable") {
		t.Error("the naming rule changed: alarm:LimitEnable is no longer a component by it")
	}
	if got := o.FallbackNote(); got != "" {
		t.Errorf("a note for types the registry described: %q", got)
	}
}

// The static table keeps deciding what it lists, whatever the registry says.
func TestTheRegistryOracleKeepsTheStaticTableFirst(t *testing.T) {
	o := oracleOver(t, `{"control:PriorityLevel": {"p": "baja:Component"}, "control:NullProxyExt": {"p": "baja:Struct"},
		"baja:Struct": {}, "baja:Component": {}}`)
	if yes, known := o.IsComponent("control:PriorityLevel"); yes || !known {
		t.Errorf("PriorityLevel = %v, %v; want a value", yes, known)
	}
	if yes, known := o.IsComponent("control:NullProxyExt"); !yes || !known {
		t.Errorf("NullProxyExt = %v, %v; want a component", yes, known)
	}
	if o.IsStruct("control:PriorityLevel") {
		t.Error("a type the table lists is not decided by the registry")
	}
}

// A type the registry does not answer, answers {} or leaves cut short is unknown: the naming rule
// decides (ClassifyComponent) and the plan note names the types outside baja it decided.
func TestTheRegistryOracleFallsBackToTheNamingRuleAndNotesIt(t *testing.T) {
	o := oracleOver(t, oracleRegistry)
	for _, spec := range []string{"acme:Bare", "acme:Missing", "baja:Double"} {
		if _, known := o.IsComponent(spec); known {
			t.Errorf("%s: known", spec)
		}
	}
	if !ClassifyComponent(o, "acme:Bare") || !ClassifyComponent(o, "acme:Missing") || ClassifyComponent(o, "baja:Double") {
		t.Error("an unknown type is not classified by the naming rule")
	}
	note := o.FallbackNote()
	if !strings.Contains(note, "acme:Bare, acme:Missing") || strings.Contains(note, "baja:Double") ||
		!strings.Contains(note, "naming rule") {
		t.Errorf("note = %q", note)
	}
}

// A nil oracle (a client without a registry channel) is the naming rule alone, and says nothing.
func TestANilRegistryOracleLeavesEveryTypeToTheNamingRule(t *testing.T) {
	var o *RegistryOracle
	for _, spec := range []any{"alarm:LimitEnable", "baja:Double", "control:PriorityLevel", "control:NullProxyExt", 7, "x"} {
		if got, want := ClassifyComponent(o, spec), box.IsComponentType(spec); got != want {
			t.Errorf("%v: %v, want %v", spec, got, want)
		}
		if got, want := ClassifyComponent(nil, spec), box.IsComponentType(spec); got != want {
			t.Errorf("%v with no oracle: %v, want %v", spec, got, want)
		}
	}
	if o.IsStruct("alarm:LimitEnable") || o.FallbackNote() != "" {
		t.Error("a nil oracle answers")
	}
}

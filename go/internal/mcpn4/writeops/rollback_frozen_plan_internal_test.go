package writeops

import (
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// fixedTypes answers a registry call from a JSON table.
type fixedTypes struct{ table *pyjson.Object }

func (f fixedTypes) LoadTypes([]string) (any, error) { return f.table, nil }

func mustJSON(t *testing.T, s string) *pyjson.Object {
	t.Helper()
	v, err := pyjson.Unmarshal([]byte(s))
	if err != nil {
		t.Fatal(err)
	}
	return v.(*pyjson.Object)
}

// The frozen-config plan writes only configuration that is off its declared default: a status
// or transient slot is ignored (not compared), a readonly slot is neither written nor ignored (it
// is still compared), a dynamic slot is not planned, and a slot at its declared default needs no
// write (the fresh child holds it).
func TestTheFrozenConfigPlanKeepsOnlyConfigurationOffItsDefault(t *testing.T) {
	ct := plan.NewContracts(fixedTypes{mustJSON(t, `{"x:Algo": {"c": [
		{"st": "p", "n": "highLimit", "v": {"nm": "p", "t": "baja:Double"}},
		{"st": "p", "n": "lowLimit", "v": {"nm": "p", "t": "baja:Double", "v": "5.0"}},
		{"st": "p", "n": "status", "v": {"nm": "p", "t": "baja:Status"}},
		{"st": "p", "n": "alarmState", "f": "r", "v": {"nm": "p", "t": "baja:String"}},
		{"st": "p", "n": "lastSeen", "f": "t", "v": {"nm": "p", "t": "baja:String"}}]}}`)})
	body := mustJSON(t, `{"nm": "p", "t": "x:Algo", "s": [
		{"nm": "p", "n": "highLimit", "v": "80.0"},
		{"nm": "p", "n": "lowLimit", "v": "5.0"},
		{"nm": "p", "n": "status", "t": "baja:Status", "v": "0"},
		{"nm": "p", "n": "alarmState", "v": "offnormal"},
		{"nm": "p", "n": "lastSeen", "v": "now"},
		{"nm": "p", "n": "note", "t": "baja:String", "v": "dynamic"}]}`)
	cfg := frozenConfig(ct, body, "x:Algo", "x:Algo")
	want := `{'type': 'x:Algo', 'swap': False, 'sets': [{'n': 'highLimit', 'b': {'nm': 'p', 'v': '80.0', 't': 'baja:Double'}}], ` +
		`'ignored': ['status', 'lastSeen']}`
	if got := pyjson.Repr(cfg); got != want {
		t.Errorf("frozen_config =\n %s\nwant\n %s", got, want)
	}
	// another declared type: swap it in, and write every configuration slot, defaults included
	swapped := frozenConfig(ct, body, "x:Algo", "x:Default")
	if get(swapped, "swap") != true || len(get(swapped, "sets").([]any)) != 2 {
		t.Errorf("swapped frozen_config = %s", pyjson.Repr(swapped))
	}
}

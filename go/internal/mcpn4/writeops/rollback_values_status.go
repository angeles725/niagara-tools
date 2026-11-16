package writeops

import (
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// The rollback guard's view of a StatusEnum or StatusString (an enum or string schedule's
// defaultOutput, written by n4_set_schedule): {value: its text, status}. An enum compares by
// ordinal (a DynamicEnum loads as "ordinal@range"); an omitted child takes its type default.

// textDefault is the value an omitted value child holds, per Status type compared as text.
var textDefault = map[string]string{"baja:StatusEnum": "0", "baja:StatusString": ""}

func textStatus(t string) bool {
	_, ok := textDefault[t]
	return ok
}

// textStatusValue is {value, status} of a text Status of type t.
func textStatusValue(t string, value, status any) *pyjson.Object {
	text, ok := value.(string)
	if !ok {
		text = textDefault[t]
	}
	if t == "baja:StatusEnum" {
		text, _, _ = strings.Cut(text, "@")
	}
	if !pyjson.Truthy(status) {
		status = "0"
	}
	return obj("value", text, "status", status)
}

// observeTextStatus is the loaded slot as textStatusValue, nil when it is gone or no longer of
// type t (it changed).
func observeTextStatus(nodes *pyjson.Object, slot, t string) any {
	n := box.Node(nodes, slot)
	if n == nil {
		return nil
	}
	if typ, _ := n.Get("t"); typ != t {
		return nil
	}
	kids := box.Children(nodes, slot)
	return textStatusValue(t, box.ChildValue(kids, "value", nil), box.ChildValue(kids, "status", "0"))
}

// observeValue is plan.Observe, with a text Status observed as textStatusValue (the rollback's
// read-back of a restored slot).
func observeValue(nodes *pyjson.Object, slot, t string) (any, error) {
	if textStatus(t) {
		return observeTextStatus(nodes, slot, t), nil
	}
	return plan.Observe(nodes, slot, t)
}

// sOf is body.get("s", []).
func sOf(b *pyjson.Object) any {
	if v, ok := b.Get("s"); ok {
		return v
	}
	return []any{}
}

func orZero(v any) any {
	if pyjson.Truthy(v) {
		return v
	}
	return "0.0"
}

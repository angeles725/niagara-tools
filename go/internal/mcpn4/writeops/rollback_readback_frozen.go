package writeops

// The frozen-child read-back of a rollback: a captured configuration compared with the live
// slots, omitted defaults included.

import (
	"errors"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// FrozenCheckDepth is FROZEN_CHECK_DEPTH: the frozen-child read-back loads the child (1), its
// slots (2) and a Status slot's value/status (3) below the child's parent.
const FrozenCheckDepth = 3

// frozenCheck is _frozen_check: nil when the live frozen child holds the captured
// configuration, else the gap. A station read error proves nothing either way: the gap carries
// readback_error and no `missing`. The slots the plan ignored (readonly, transient, status) are
// not compared; the sets the station refused are listed in refused, with its reason.
func frozenCheck(c plan.Client, check *pyjson.Object, refused []any) (*pyjson.Object, error) {
	path := get(check, "path").(string)
	head, leaf := rpartition(path)
	body := get(check, "b").(*pyjson.Object)
	gap := obj("path", path, "type", get(body, "t"), "captured", body)
	nodes, err := c.LoadTree(head, FrozenCheckDepth, box.LoadOptions{})
	var be *box.Error
	if errors.As(err, &be) {
		gap.Set("readback_error", be.Msg)
		return gap, nil
	} else if err != nil {
		return nil, err
	}
	live := box.Node(nodes, leaf)
	if live == nil {
		kids, _ := get(body, "s").([]any)
		slots := []any{}
		for _, k := range kids {
			n, err := need(k.(*pyjson.Object), "n")
			if err != nil {
				return nil, err
			}
			slots = append(slots, n)
		}
		set(gap, "missing", true, "live_type", nil, "slots", slots)
		return gap, nil
	}
	slots, err := differs(nodes, leaf, body)
	if err != nil {
		return nil, err
	}
	slots = without(slots, get(check, "ignored"))
	if box.PyEqual(get(live, "t"), get(body, "t")) && len(slots) == 0 && box.PyEqual(get(live, "v"), get(body, "v")) {
		return nil, nil
	}
	set(gap, "missing", false, "live_type", get(live, "t"), "slots", slots)
	if len(refused) > 0 {
		gap.Set("refused", refused)
	}
	return gap, nil
}

// defaultV is _DEFAULT_V: type defaults the station omits from a load.
var defaultV = map[string]string{"baja:Status": "0", "baja:Double": "0.0", "baja:Boolean": "false"}

// statusValueType is _STATUS_VALUE_TYPE: the type of a Status slot's typeless value child.
var statusValueType = map[string]string{"baja:StatusNumeric": "baja:Double", "baja:StatusBoolean": "baja:Boolean"}

// defaultOf is _default_v: the omitted-default value of a captured slot, or nil when it has none.
func defaultOf(slot *pyjson.Object, parentT any) any {
	kind, typeless := get(slot, "t"), !has(slot, "t") || get(slot, "t") == nil
	if typeless && get(slot, "n") == "status" {
		kind = "baja:Status"
	} else if pt, ok := parentT.(string); typeless && get(slot, "n") == "value" && ok {
		kind = statusValueType[pt]
	}
	if k, ok := kind.(string); ok {
		if v, ok := defaultV[k]; ok {
			return v
		}
	}
	return nil
}

// differs is _differs: the names of the captured slots whose live value differs from the
// snapshot. A slot the station omits is at its default: a scalar is equal only when its
// captured value is the type default, a struct only when it has no own value and every
// captured child is at its default, recursively.
func differs(nodes *pyjson.Object, prefix string, captured *pyjson.Object) ([]any, error) {
	out := []any{}
	kids, _ := get(captured, "s").([]any)
	for _, k := range kids {
		slot := k.(*pyjson.Object)
		nv, err := need(slot, "n")
		if err != nil {
			return nil, err
		}
		n, ok := nv.(string)
		if !ok {
			return nil, &plan.PyError{Type: "TypeError", Msg: "can only concatenate str (not \"" + plan.TypeName(nv) + "\") to str"}
		}
		here := prefix + "/" + n
		live := box.Node(nodes, here)
		var same bool
		switch {
		case live == nil && pyjson.Truthy(get(slot, "s")):
			if get(slot, "v") == nil {
				inner, err := differs(nodes, here, slot)
				if err != nil {
					return nil, err
				}
				same = len(inner) == 0
			}
		case live == nil:
			v := get(slot, "v")
			same = v != nil && box.PyEqual(v, defaultOf(slot, get(captured, "t")))
		default:
			t, hasT := slot.Get("t")
			same = (!hasT || box.PyEqual(get(live, "t"), t)) && box.PyEqual(configV(live, slot), get(slot, "v"))
			if same {
				inner, err := differs(nodes, here, slot)
				if err != nil {
					return nil, err
				}
				same = len(inner) == 0
			}
		}
		if !same {
			out = append(out, n)
		}
	}
	return out, nil
}

// configV is _config_v: a live Status value without its runtime facets.
func configV(live, slot *pyjson.Object) any {
	v := get(live, "v")
	if v != nil && (get(slot, "t") == "baja:Status" || get(slot, "n") == "status") {
		return strings.SplitN(pyjson.Str(v), ";", 2)[0]
	}
	return v
}

// without is names less the ones listed in drop (a list of names, or nil).
func without(names []any, drop any) []any {
	skip, _ := drop.([]any)
	if len(skip) == 0 {
		return names
	}
	out := []any{}
	for _, n := range names {
		keep := true
		for _, s := range skip {
			keep = keep && s != n
		}
		if keep {
			out = append(out, n)
		}
	}
	return out
}

func has(o *pyjson.Object, k string) bool { _, ok := o.Get(k); return ok }

// set is d.update(k=v, ...) in argument order.
func set(o *pyjson.Object, kv ...any) {
	for i := 0; i < len(kv); i += 2 {
		o.Set(kv[i].(string), kv[i+1])
	}
}

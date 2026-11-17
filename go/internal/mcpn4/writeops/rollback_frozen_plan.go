package writeops

// The frozen-config plan of a rollback (retro 2026-11-16 row M18). A component the station
// creates with its parent (a writable's proxyExt, an alarm extension's offnormalAlgorithm) is a
// frozen slot: the rollback never re-adds it, so what it held comes back only by set ops after
// the parent's add. The station's type contracts tell which nested components are frozen and
// which of their slots are configuration; the plan records, on each such component spec, the
// type to swap in and the set ops to send ("frozen_config"), so the dry run shows them and the
// token covers them. Without contracts nothing is planned and the read-back reports the gap.

import (
	"strconv"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// The declared flags that keep a slot out of the frozen-config restore. A slot flagged
// neverWrittenFlags (r readonly, t transient) is never written; only a slot flagged
// notComparedFlags (t: runtime state, never persisted) also leaves the read-back's comparison, so
// a readonly slot that differs still makes the verdict partial. A status slot is neither.
const (
	neverWrittenFlags = "rt"
	notComparedFlags  = "t"
)

// planFrozenConfig sets "frozen_config" on each component spec the contracts declare as a frozen
// slot of its parent and that has something to restore; it returns the writes they may need.
// A type the station does not describe leaves its spec unplanned.
func planFrozenConfig(c plan.Client, ops, components []any) int {
	tl, ok := c.(plan.TypeLoader)
	if !ok || len(components) == 0 {
		return 0
	}
	ct := plan.NewContracts(tl)
	typeOf := map[string]string{}
	specs := []string{}
	for i, op := range ops {
		if t, ok := get(asObj(get(op.(*pyjson.Object), "b")), "t").(string); ok {
			typeOf[strconv.Itoa(i)+":"] = t
			specs = append(specs, t)
		}
	}
	for _, s := range components {
		if t, ok := get(asObj(get(s.(*pyjson.Object), "b")), "t").(string); ok {
			specs = append(specs, t)
		}
	}
	_ = ct.Load(specs) // one registry call; a type it leaves out is skipped below
	writes := 0
	for _, s := range components {
		spec := s.(*pyjson.Object)
		top := strconv.Itoa(get(spec, "top").(int)) + ":"
		body := asObj(get(spec, "b"))
		ctype, _ := get(body, "t").(string)
		typeOf[top+specPath(spec)] = ctype
		ptype := typeOf[top+get(spec, "parent_path").(string)]
		declared, err := ct.Slot(ptype, get(spec, "n").(string))
		if err != nil || declared == nil || ctype == "" {
			continue
		}
		cfg := frozenConfig(ct, body, ctype, get(asObj(get(declared, "v")), "t"))
		if cfg != nil {
			spec.Set("frozen_config", cfg)
			writes += len(get(cfg, "sets").([]any))
			if get(cfg, "swap") == true {
				writes++
			}
		}
	}
	return writes
}

// frozenConfig is the restore of one frozen component of captured type ctype (declared with a
// default of type dtype): swap when the types differ, a set per configuration slot whose captured
// value is not the declared default, and the slots the read-back must not compare (ignored:
// transient and status; a readonly slot is not written but is still compared);
// nil when there is nothing to record.
func frozenConfig(ct *plan.Contracts, body *pyjson.Object, ctype string, dtype any) *pyjson.Object {
	swap := dtype != nil && !box.PyEqual(dtype, ctype)
	sets, ignored := []any{}, []any{}
	kids, _ := get(body, "s").([]any)
	for _, k := range kids {
		kid := k.(*pyjson.Object)
		n, _ := get(kid, "n").(string)
		slot, err := ct.Slot(ctype, n)
		if err != nil || slot == nil || n == "" { // a dynamic slot or an unread type: not planned
			continue
		}
		flags, _ := get(slot, "f").(string)
		if n == "status" || get(kid, "t") == "baja:Status" || strings.ContainsAny(flags, notComparedFlags) {
			ignored = append(ignored, n)
			continue
		}
		if strings.ContainsAny(flags, neverWrittenFlags) { // readonly: compared, never written
			continue
		}
		def := asObj(get(slot, "v"))
		if !swap && atDeclaredDefault(kid, def, ctype) {
			continue
		}
		b := obj()
		for _, key := range kid.Keys() {
			if key != "n" {
				b.Set(key, get(kid, key))
			}
		}
		if _, typed := b.Get("t"); !typed {
			if get(def, "t") == nil {
				continue // no type to send the value as: the read-back reports it
			}
			b.Set("t", get(def, "t"))
		}
		sets = append(sets, obj("n", n, "b", b))
	}
	if !swap && len(sets) == 0 && len(ignored) == 0 {
		return nil
	}
	return obj("type", ctype, "swap", swap, "sets", sets, "ignored", ignored)
}

// atDeclaredDefault says whether a captured plain slot holds its declared default (a fresh
// frozen component already has it); a slot with children is always written.
func atDeclaredDefault(kid, def *pyjson.Object, parentT string) bool {
	if pyjson.Truthy(get(kid, "s")) {
		return false
	}
	v, dv := get(kid, "v"), get(def, "v")
	if dv == nil {
		dv = defaultOf(kid, parentT)
	}
	return box.PyEqual(v, dv)
}

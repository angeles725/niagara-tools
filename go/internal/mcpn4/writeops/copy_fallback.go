package writeops

import (
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// A writable point's fallback is its output when no priority input is set: a copy whose fallback
// is not null commands its device with that value on its first communication (field retro
// 2026-11-16, row U18: a power point with fallback 0 = OFF and a setpoint with 200 were about to be
// replicated onto every device). The copy body carries fallback only when it is off its default
// (null, live 4.15: a null fallback is absent); a status left out of the body is ok.

// fallbackNotes is one note per writable point inside body (body itself included, named label)
// whose fallback is not null: a dry-run warning, never a refusal.
func fallbackNotes(label string, body *pyjson.Object) []any {
	notes := []any{}
	walkWritables(body, label, func(path, value string) {
		notes = append(notes, path+": writable point with a non-null fallback ("+value+"): every copy commands "+
			"its device with it on first communication (the fallback is the output when no priority input is "+
			"set); confirm that value is safe on each target, or set fallback to null on the copies")
	})
	return notes
}

// walkWritables calls found for every node of type *Writable under (and including) node whose
// fallback is not null, with its path and fallback value text.
func walkWritables(node *pyjson.Object, path string, found func(path, value string)) {
	if t, _ := get(node, "t").(string); strings.HasSuffix(t, "Writable") {
		if fb := kidNamed(node, "fallback"); fb != nil && !fallbackNull(fb) {
			value, _ := get(kidNamed(fb, "value"), "v").(string)
			if value == "" {
				value, _ = get(fb, "v").(string)
			}
			found(path, value)
		}
	}
	kids, _ := get(node, "s").([]any)
	for _, k := range kids {
		if kid, ok := k.(*pyjson.Object); ok {
			if n := name(kid); n != "" && n != "fallback" {
				walkWritables(kid, path+"/"+n, found)
			}
		}
	}
}

// fallbackNull is a status value whose status child has the null bit; a missing or unreadable
// status is not null (the copy body omits a status at its default, ok).
func fallbackNull(v *pyjson.Object) bool {
	_, null, err := box.ParseStatus(get(kidNamed(v, "status"), "v"))
	return err == nil && null
}

package writeops_test

import (
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/writeops"
)

// A station names a slot with escapes ($20 is a space); a copy reads such a source by its escaped
// name, while every new name stays plain.
func TestACopyReadsASourceWhoseNameHasStationEscapes(t *testing.T) {
	src := "Numeric$20Writable40001"
	tree := obj("h", "40", "t", "control:NumericWritable", "s", []any{
		obj("nm", "p", "n", "fallback", "t", "baja:StatusNumeric", "s", []any{obj("nm", "p", "n", "value", "v", "0.0")})})
	parent := obj("", obj("h", "10", "t", "baja:Folder"), src, obj("h", "40", "t", "control:NumericWritable"))
	c := &loads{nodes: obj(net, parent, net+"/"+src, obj("", tree))}
	p, err := writeops.CopyComponent.Plan(c, obj("parent_ord", net, "name", src, "new_names", []any{"FanSpeed"}))
	if err != nil {
		t.Fatal(err)
	}
	if got := js(t, p.Ops[0]); !strings.Contains(got, `"n": "FanSpeed"`) || !strings.Contains(got, "control:NumericWritable") {
		t.Errorf("op = %s", got)
	}
	if _, err := writeops.CopyComponent.Plan(c, obj("parent_ord", net, "name", src, "new_names", []any{"Fan$20Speed"})); err == nil {
		t.Errorf("an escaped new name was accepted")
	}
}

// Only well-formed escapes pass: a '$' must be followed by two hex digits; separators never pass.
func TestAnExistingNameRefusesMalformedEscapesAndSeparators(t *testing.T) {
	for _, bad := range []string{"a$2", "a$zz", "a/b", "a|b", "$", "", "a b"} {
		c := &loads{nodes: obj(net, obj("", obj("h", "10"), bad, obj("h", "40")))}
		_, err := writeops.CopyComponent.Plan(c, obj("parent_ord", net, "name", bad, "new_names", []any{"X"}))
		if err == nil || !strings.Contains(err.Error(), "must be a plain slot name") {
			t.Errorf("%q: err = %v", bad, err)
		}
	}
	_ = pyjson.NewObject
}

package readtools

import (
	"slices"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/registry"
)

// readEnum is the "enum" of argument arg in read tool's input schema (of its items when items),
// as strings.
func readEnum(t *testing.T, tool, arg string, items bool) []string {
	t.Helper()
	for _, tl := range registry.Default().Read {
		if tl.Name != tool {
			continue
		}
		props, _ := lookup(tl.Schema, "properties").(*pyjson.Object)
		spec, _ := lookup(props, arg).(*pyjson.Object)
		if items {
			spec, _ = lookup(spec, "items").(*pyjson.Object)
		}
		values, _ := lookup(spec, "enum").([]any)
		out := []string{}
		for _, v := range values {
			out = append(out, v.(string))
		}
		return out
	}
	t.Fatalf("no read tool %s", tool)
	return nil
}

// n4_commissioning_report and n4_read_tags trust the shared validator for these enums (retro
// row M11b deleted their own checks), so each schema enum must be exactly what the tool handles:
// a value added to the schema only would pass validation and then build nothing.
func TestReadSchemaEnumsMatchTheToolTables(t *testing.T) {
	if got := readEnum(t, "n4_commissioning_report", "sections", true); !slices.Equal(got, commSections) {
		t.Errorf("sections: schema enum %v, report sections %v", got, commSections)
	}
	for _, name := range commSections {
		if commBuilders[name] == nil {
			t.Errorf("section %s has no builder", name)
		}
	}
	// tagEntry handles direct (the default), all and none.
	if got := readEnum(t, "n4_read_tags", "relations", false); !slices.Equal(got, []string{"direct", "all", "none"}) {
		t.Errorf("relations: schema enum %v, tagEntry handles direct, all, none", got)
	}
}

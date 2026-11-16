package writeops

import (
	"slices"
	"sort"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/registry"
)

// schemaEnum is the "enum" of argument arg in write tool's input schema, as strings.
func schemaEnum(t *testing.T, tool, arg string) []string {
	t.Helper()
	for _, tl := range registry.Default().Write {
		if tl.Name != tool {
			continue
		}
		props, _ := mustGet(tl.Schema, "properties").(*pyjson.Object)
		spec, _ := mustGet(props, arg).(*pyjson.Object)
		values, _ := mustGet(spec, "enum").([]any)
		out := []string{}
		for _, v := range values {
			out = append(out, v.(string))
		}
		return out
	}
	t.Fatalf("no write tool %s", tool)
	return nil
}

func mustGet(o *pyjson.Object, k string) any {
	if o == nil {
		return nil
	}
	v, _ := o.Get(k)
	return v
}

func keys(m map[string]string) []string {
	out := []string{}
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// The tools trust the shared validator for these enums (retro row M11b deleted their own
// checks), so each schema enum must be exactly the table the tool maps it with: a value added
// to one side only would pass validation and then map to nothing.
func TestWriteSchemaEnumsMatchTheToolTables(t *testing.T) {
	for _, c := range []struct {
		tool, arg string
		table     []string
	}{
		{"n4_add_history", "full_policy", keys(fullPolicies)},
		{"n4_add_history", "mode", keys(historyModes)},
		{"n4_add_history", "ext_type", historyExtTypes},
		{"n4_add_alarm", "kind", alarmKindNames()},
	} {
		got := schemaEnum(t, c.tool, c.arg)
		want := slices.Clone(c.table)
		sort.Strings(got)
		sort.Strings(want)
		if !slices.Equal(got, want) {
			t.Errorf("%s %s: schema enum %v, tool table %v", c.tool, c.arg, got, want)
		}
	}
}

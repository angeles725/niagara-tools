package registry_test

import (
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/jsonrpc"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/registry"
)

func toolNamed(t *testing.T, name string) registry.Tool {
	t.Helper()
	r := registry.Default()
	for _, tool := range append(append([]registry.Tool{}, r.Read...), r.Write...) {
		if tool.Name == name {
			return tool
		}
	}
	t.Fatalf("tool %s is not registered", name)
	return registry.Tool{}
}

func argsOf(t *testing.T, text string) *pyjson.Object {
	t.Helper()
	v, err := pyjson.Unmarshal([]byte(text))
	if err != nil {
		t.Fatal(err)
	}
	return v.(*pyjson.Object)
}

// #859: n4_add_history declares capacity_records (a number of records), so the schema check lets
// it through to the tool; the other unknown names stay refused with their suggestion.
func TestAddHistoryDeclaresCapacityRecords(t *testing.T) {
	t.Parallel()
	tool := toolNamed(t, "n4_add_history")
	base := `"points": ["station:|slot:/A"], "history_name": "h", "interval_ms": 60000`
	if err := jsonrpc.Validate(tool.Name, tool.Schema, argsOf(t, `{`+base+`, "capacity_records": 500}`)); err != nil {
		t.Errorf("capacity_records 500 refused: %v", err)
	}
	for args, want := range map[string]string{
		`{` + base + `, "capacity_records": "500"}`: "argument capacity_records must be integer",
		`{` + base + `, "capacity_records": 0}`:     "argument capacity_records must be >= 1",
		`{` + base + `, "capacity_recs": 5}`:        "unknown argument 'capacity_recs' for n4_add_history; did you mean 'capacity'?",
	} {
		err := jsonrpc.Validate(tool.Name, tool.Schema, argsOf(t, args))
		if err == nil || !strings.HasPrefix(err.Error(), want) {
			t.Errorf("%s: err = %v, want %q", args, err, want)
		}
	}
	if d, _ := tool.Descriptor.Get("description"); !strings.Contains(d.(string), "capacity_records") {
		t.Error("the description does not name capacity_records")
	}
}

// #876: BqlQuery decodes $xx inside the query text while slot names are stored escaped, so the
// description of n4_bql_query tells to write $24 for the literal $ of an escaped name.
func TestBqlQueryDescriptionTellsTheEscapedNameForm(t *testing.T) {
	t.Parallel()
	d, _ := toolNamed(t, "n4_bql_query").Descriptor.Get("description")
	for _, want := range []string{"'Numeric$2420Writable'", "$24", "hint"} {
		if !strings.Contains(d.(string), want) {
			t.Errorf("description lacks %q", want)
		}
	}
}

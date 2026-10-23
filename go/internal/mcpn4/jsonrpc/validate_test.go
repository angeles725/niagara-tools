package jsonrpc

import (
	"errors"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/registry"
)

func decodeObj(t *testing.T, text string) *pyjson.Object {
	t.Helper()
	v, err := pyjson.Unmarshal([]byte(text))
	if err != nil {
		t.Fatalf("Unmarshal(%s): %v", text, err)
	}
	return v.(*pyjson.Object)
}

func TestValidateSupportsNumberAndArray(t *testing.T) {
	t.Parallel()
	schema := decodeObj(t, `{"properties": {"n": {"type": "number", "minimum": 0}, "a": {"type": "array"},
		"i": {"type": "integer", "minimum": 1, "maximum": 3.5}, "u": {"description": "any"}}}`)
	for _, ok := range []string{`{"n": 1.5, "a": [1]}`, `{"n": 2}`, `{"i": 3}`, `{"n": NaN}`, `{"u": [{}]}`, `{"x": true}`} {
		if err := Validate(schema, decodeObj(t, ok)); err != nil {
			t.Errorf("%s: %v", ok, err)
		}
	}
	for args, want := range map[string]string{
		`{"n": "x"}`:            "argument n must be number",
		`{"n": true}`:           "argument n must be number",
		`{"a": {}}`:             "argument a must be array",
		`{"n": -1}`:             "argument n must be >= 0",
		`{"n": -1e-300}`:        "argument n must be >= 0",
		`{"i": 1.0}`:            "argument i must be integer",
		`{"i": 4}`:              "argument i must be <= 3.5",
		`{"i": 0, "n": "late"}`: "argument i must be >= 1",
		`{"n": -Infinity}`:      "argument n must be >= 0",
	} {
		var ip *InvalidParams
		if err := Validate(schema, decodeObj(t, args)); !errors.As(err, &ip) || ip.Msg != want {
			t.Errorf("%s: err = %v, want %q", args, err, want)
		}
	}
}

func TestValidateComparesBigIntsExactly(t *testing.T) {
	t.Parallel()
	schema := decodeObj(t, `{"properties": {"i": {"type": "integer", "maximum": 9007199254740992}}}`)
	if err := Validate(schema, decodeObj(t, `{"i": 9007199254740993}`)); err == nil {
		t.Errorf("2**53+1 passed a maximum of 2**53 (float rounding)")
	}
	schema = decodeObj(t, `{"properties": {"i": {"type": "integer", "maximum": 1e300}}}`)
	if err := Validate(schema, decodeObj(t, `{"i": 10}`)); err != nil {
		t.Errorf("10 vs 1e300: %v", err)
	}
}

func TestValidateRequiredKeysComeFirstInSchemaOrder(t *testing.T) {
	t.Parallel()
	schema := decodeObj(t, `{"properties": {"a": {"type": "string"}}, "required": ["b", "a"]}`)
	var ip *InvalidParams
	if err := Validate(schema, decodeObj(t, `{"a": 5}`)); !errors.As(err, &ip) || ip.Msg != "missing required argument: b" {
		t.Errorf("err = %v", err)
	}
}

func TestValidateFailsClearlyOnAnUnknownSchemaType(t *testing.T) {
	t.Parallel()
	var se *SchemaError
	err := Validate(decodeObj(t, `{"properties": {"x": {"type": "float"}}}`), decodeObj(t, `{"x": 1}`))
	if !errors.As(err, &se) || se.Msg != "schema for x uses unsupported type 'float'" {
		t.Errorf("err = %v", err)
	}
}

func TestEveryRegisteredToolSchemaUsesAKnownType(t *testing.T) {
	t.Parallel()
	r := registry.Default()
	for _, tool := range append(append([]registry.Tool{}, r.Read...), r.Write...) {
		props, _ := lookup(tool.Schema, "properties").(*pyjson.Object)
		if props == nil {
			t.Fatalf("%s has no properties", tool.Name)
		}
		for _, k := range props.Keys() {
			kind := lookup(lookup(props, k).(*pyjson.Object), "type")
			if name, _ := kind.(string); kind != nil && !known(name) {
				t.Errorf("%s.%s uses type %v", tool.Name, k, kind)
			}
		}
	}
}

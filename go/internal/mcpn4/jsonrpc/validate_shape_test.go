package jsonrpc

import (
	"errors"
	"testing"
)

// refusedWith fails t unless Validate refuses args under schema with exactly want (-32602).
func refusedWith(t *testing.T, schema, args, want string) {
	t.Helper()
	var ip *InvalidParams
	if err := Validate("tool", decodeObj(t, schema), decodeObj(t, args)); !errors.As(err, &ip) || ip.Msg != want {
		t.Errorf("%s: err = %v, want %q", args, err, want)
	}
}

func accepted(t *testing.T, schema, args string) {
	t.Helper()
	if err := Validate("tool", decodeObj(t, schema), decodeObj(t, args)); err != nil {
		t.Errorf("%s: %v", args, err)
	}
}

func TestValidateRefusesAnUnknownArgumentWithADidYouMean(t *testing.T) {
	t.Parallel()
	s := `{"properties": {"capacity": {"type": "integer"}, "depth": {"type": "integer"}, "dry_run": {}, "dry_mode": {}}}`
	refusedWith(t, s, `{"capacity_records": 5}`, "unknown argument 'capacity_records' for tool; did you mean 'capacity'?")
	refusedWith(t, s, `{"depht": 1}`, "unknown argument 'depht' for tool; did you mean 'depth'?")
	refusedWith(t, s, `{"zzz": 1}`, "unknown argument 'zzz' for tool")
	refusedWith(t, s, `{"dry": true}`, "unknown argument 'dry' for tool")           // a tie: no guess
	refusedWith(t, s, `{"depth": "x", "zzz": 1}`, "argument depth must be integer") // insertion order
	refusedWith(t, s, `{"zzz": 1, "depth": "x"}`, "unknown argument 'zzz' for tool")
}

func TestValidateKeepsAnOpenObjectOpen(t *testing.T) {
	t.Parallel()
	accepted(t, `{"properties": {"alpha": {}}, "additionalProperties": true}`, `{"b": 1}`)
	accepted(t, `{"properties": {"m": {"type": "object", "additionalProperties": {"type": "integer"}}}}`,
		`{"m": {"anyTag": 1}}`)
	accepted(t, `{"properties": {"m": {"type": "object"}}}`, `{"m": {"free": [1]}}`) // no declared properties
	accepted(t, `{}`, `{"anything": 1}`)                                             // no properties at all
	refusedWith(t, `{"properties": {"alpha": {}}, "additionalProperties": false}`, `{"b": 1}`, "unknown argument 'b' for tool")
}

func TestValidateRefusesAnEnumViolationNamingTheAllowedValues(t *testing.T) {
	t.Parallel()
	s := `{"properties": {"p": {"type": "string", "enum": ["roll", "stop"]}, "n": {"enum": [1, 2]},
		"l": {"type": "array", "items": {"type": "string", "enum": ["a", "b"]}}, "u": {"anyOf": [{"enum": ["x"]}]}}}`
	accepted(t, s, `{"p": "stop", "n": 2.0, "l": ["b", "a"], "u": "free"}`)
	refusedWith(t, s, `{"p": "drop"}`, "argument p must be one of roll, stop; got 'drop'")
	refusedWith(t, s, `{"p": "ROLL"}`, "argument p must be one of roll, stop; got 'ROLL'")
	refusedWith(t, s, `{"n": true}`, "argument n must be one of 1, 2; got True")
	refusedWith(t, s, `{"n": "1"}`, "argument n must be one of 1, 2; got '1'")
	refusedWith(t, s, `{"l": ["a", "c"]}`, "argument l[1] must be one of a, b; got 'c'")
	refusedWith(t, s, `{"p": 5}`, "argument p must be string") // the type first
}

func TestValidateChecksNestedObjectsOnlyWhereTheyDeclareProperties(t *testing.T) {
	t.Parallel()
	s := `{"properties": {"points": {"type": "array", "items": {"type": "object", "required": ["ord"],
		"properties": {"ord": {"type": "string"}, "kind": {"type": "string", "enum": ["a"]}}}},
		"cfg": {"type": "object", "properties": {"rate": {"type": "integer", "minimum": 1}}}}}`
	accepted(t, s, `{"points": [{"ord": "x", "kind": "a"}], "cfg": {"rate": 2}}`)
	refusedWith(t, s, `{"points": [{"ord": "x", "knd": "a"}]}`, "unknown argument 'points[0].knd' for tool; did you mean 'kind'?")
	refusedWith(t, s, `{"points": [{"ord": "x"}, {"kind": "a"}]}`, "missing required argument: points[1].ord")
	refusedWith(t, s, `{"points": [{"ord": "x", "kind": "b"}]}`, "argument points[0].kind must be one of a; got 'b'")
	refusedWith(t, s, `{"cfg": {"rate": 0}}`, "argument cfg.rate must be >= 1")
}

package jsonrpc

import (
	"fmt"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// validateObject checks args against an object schema: its required keys in schema order, then
// each argument in insertion order. prefix is the path of args ("" at the top, "points[0]."
// below it).
func validateObject(tool, prefix string, schema, args *pyjson.Object) error {
	props, _ := lookup(schema, "properties").(*pyjson.Object)
	if req, ok := lookup(schema, "required").([]any); ok {
		for _, key := range req {
			k, _ := key.(string)
			if _, ok := args.Get(k); !ok {
				return &InvalidParams{"missing required argument: " + prefix + pyjson.Str(key)}
			}
		}
	}
	closed := props != nil && !openObject(schema)
	for _, key := range args.Keys() {
		value, _ := args.Get(key)
		spec, ok := lookup(props, key).(*pyjson.Object)
		if !ok {
			if closed {
				return unknownArgument(tool, prefix+key, key, props)
			}
			continue
		}
		if err := checkValue(tool, prefix+key, spec, value); err != nil {
			return err
		}
	}
	return nil
}

// openObject is whether schema keeps its object open to keys it does not declare: any
// "additionalProperties" but false (alarm_values: a tag -> ordinal map).
func openObject(schema *pyjson.Object) bool {
	extra, ok := schema.Get("additionalProperties")
	return ok && extra != false
}

// checkNested descends into an object whose schema declares "properties", and into each item of
// an array whose "items" declare an "enum" or "properties".
func checkNested(tool, name string, spec *pyjson.Object, value any) error {
	switch v := value.(type) {
	case *pyjson.Object:
		if _, declared := lookup(spec, "properties").(*pyjson.Object); declared {
			return validateObject(tool, name+".", spec, v)
		}
	case []any:
		items, _ := lookup(spec, "items").(*pyjson.Object)
		if items == nil {
			return nil
		}
		for i, item := range v {
			path := fmt.Sprintf("%s[%d]", name, i)
			if err := checkEnum(path, items, item); err != nil {
				return err
			}
			if err := checkNested(tool, path, items, item); err != nil {
				return err
			}
		}
	}
	return nil
}

// checkEnum refuses a value that is not one of spec's "enum" values, naming them all.
func checkEnum(name string, spec *pyjson.Object, value any) error {
	allowed, ok := lookup(spec, "enum").([]any)
	if !ok {
		return nil
	}
	names := make([]string, len(allowed))
	for i, a := range allowed {
		if sameValue(a, value) {
			return nil
		}
		names[i] = pyjson.Str(a)
		if a == "" {
			names[i] = "''"
		}
	}
	return &InvalidParams{fmt.Sprintf("argument %s must be one of %s; got %s",
		name, strings.Join(names, ", "), echo(value))}
}

// sameValue is JSON equality for an enum member: numbers by value (1 == 1.0, never a bool),
// everything else by its representation.
func sameValue(a, b any) bool {
	if _, isNum := a.(pyjson.Number); isNum {
		c, comparable, err := compare(a, b)
		return err == nil && comparable && c == 0
	}
	_, aBool := a.(bool)
	_, bBool := b.(bool)
	return aBool == bBool && pyjson.Repr(a) == pyjson.Repr(b)
}

// unknownArgument is the refusal of an undeclared argument, with a did-you-mean when exactly one
// declared key is close to it.
func unknownArgument(tool, path, key string, props *pyjson.Object) error {
	msg := fmt.Sprintf("unknown argument %s for %s", pyjson.StrRepr(clip(path)), tool)
	if near := nearestKey(key, props.Keys()); near != "" {
		msg += fmt.Sprintf("; did you mean %s?", pyjson.StrRepr(near))
	}
	return &InvalidParams{msg}
}

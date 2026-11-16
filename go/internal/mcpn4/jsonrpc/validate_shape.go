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
	}
	return &InvalidParams{fmt.Sprintf("argument %s must be one of %s; got %s",
		name, strings.Join(names, ", "), pyjson.Repr(value))}
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
	msg := fmt.Sprintf("unknown argument %s for %s", pyjson.StrRepr(path), tool)
	if near := nearestKey(key, props.Keys()); near != "" {
		msg += fmt.Sprintf("; did you mean %s?", pyjson.StrRepr(near))
	}
	return &InvalidParams{msg}
}

// nearestKey is the declared key closest to key: one that starts the other (capacity_records,
// capacity), else the single key within edit distance max(1, len/4); "" when none or a tie.
func nearestKey(key string, known []string) string {
	best, bestScore, tie := "", -1, false
	for _, k := range known {
		score := -1
		switch {
		case strings.HasPrefix(key, k) || strings.HasPrefix(k, key):
			score = 0
		case editDistance(key, k) <= max(1, len(k)/4):
			score = editDistance(key, k)
		}
		if score < 0 {
			continue
		}
		if bestScore < 0 || score < bestScore {
			best, bestScore, tie = k, score, false
		} else if score == bestScore {
			tie = true
		}
	}
	if tie {
		return ""
	}
	return best
}

// editDistance is the optimal-string-alignment distance of a and b: insertions, deletions,
// substitutions and swaps of two neighbours (depht, depth) cost 1 each. Bytes: argument names
// are ASCII.
func editDistance(a, b string) int {
	rows := make([][]int, len(a)+1)
	for i := range rows {
		rows[i] = make([]int, len(b)+1)
		rows[i][0] = i
	}
	for j := range rows[0] {
		rows[0][j] = j
	}
	for i := 1; i <= len(a); i++ {
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			d := min(rows[i-1][j]+1, rows[i][j-1]+1, rows[i-1][j-1]+cost)
			if i > 1 && j > 1 && a[i-1] == b[j-2] && a[i-2] == b[j-1] {
				d = min(d, rows[i-2][j-2]+1)
			}
			rows[i][j] = d
		}
	}
	return rows[len(a)][len(b)]
}

package blackbox

import "sort"

// SchemaArgs are placeholder arguments that pass schema validation of a tool's inputSchema:
// every required property gets a value of its type (enum: the first value; string: Folder;
// integer/number: its minimum, else 1; array: minItems items, at least one). A guard that
// needs to call every tool with valid arguments derives them from tools/list with it, so a new
// tool is covered without a hand-written argument list.
func SchemaArgs(schema map[string]any) Args {
	props, _ := schema["properties"].(map[string]any)
	required, _ := schema["required"].([]any)
	out := Args{}
	for _, name := range required {
		key, _ := name.(string)
		prop, _ := props[key].(map[string]any)
		out[key] = placeholder(prop)
	}
	return out
}

func placeholder(prop map[string]any) any {
	if enum, _ := prop["enum"].([]any); len(enum) > 0 {
		return enum[0]
	}
	switch prop["type"] {
	case "integer", "number":
		if min, ok := prop["minimum"].(float64); ok {
			return min
		}
		return 1
	case "boolean":
		return false
	case "array":
		item, _ := prop["items"].(map[string]any)
		n, _ := prop["minItems"].(float64)
		list := []any{}
		for i := 0; i < max(int(n), 1); i++ {
			list = append(list, placeholder(item))
		}
		return list
	case "object":
		nested := SchemaArgs(prop)
		return nested
	}
	return Folder // "string", and a property of any type
}

// WriteTools are the names of the tools a tools/list marks as writing (annotations.readOnlyHint
// false), sorted.
func WriteTools(tools []map[string]any) []string {
	var out []string
	for _, tool := range tools {
		if Dig(tool, "annotations", "readOnlyHint") == false {
			out = append(out, tool["name"].(string))
		}
	}
	sort.Strings(out)
	return out
}

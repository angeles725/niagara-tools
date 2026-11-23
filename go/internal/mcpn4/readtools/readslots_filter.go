package readtools

import "github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"

// slotFilter is the slots argument: the names asked for and whether a filter was given (the
// schema guarantees a list of strings).
func slotFilter(args *pyjson.Object) ([]string, bool) {
	raw, ok := args.Get("slots")
	list, _ := raw.([]any)
	if !ok || len(list) == 0 {
		return nil, false
	}
	names := make([]string, 0, len(list))
	for _, v := range list {
		if n, ok := v.(string); ok {
			names = append(names, n)
		}
	}
	return names, true
}

// keepSlots is the entries of slots named in wanted, in their own (station) order, and the
// wanted names no entry carries, in the order asked and once each.
func keepSlots(slots []any, wanted []string) (kept, notFound []any) {
	want := map[string]bool{}
	for _, n := range wanted {
		want[n] = true
	}
	kept, found := []any{}, map[string]bool{}
	for _, e := range slots {
		name, _ := e.(*pyjson.Object).Get("name")
		if n, _ := name.(string); want[n] {
			kept = append(kept, e)
			found[n] = true
		}
	}
	seen := map[string]bool{}
	for _, n := range wanted {
		if !found[n] && !seen[n] {
			notFound = append(notFound, n)
		}
		seen[n] = true
	}
	return kept, notFound
}

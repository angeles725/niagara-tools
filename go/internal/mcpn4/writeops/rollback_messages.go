package writeops

import (
	"fmt"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

func rollbackNotes(view *pyjson.Object, bid string, relinks, unlinked, components, own, ops []any) []any {
	notes := []any{"rolls back batch " + bid + " (" + pyjson.Str(get(view, "tool")) + ")"}
	if len(relinks) > 0 {
		notes = append(notes, fmt.Sprintf("then re-creates %d link(s) between restored components, where both "+
			"ends exist; a skipped one is reported in link_inputs_not_restored with "+
			"the input's recorded value", len(relinks)))
	}
	if len(unlinked) > 0 {
		notes = append(notes, fmt.Sprintf("%d input(s) were linked from outside the removed subtree: their links "+
			"are not re-created; they are reported in link_inputs_not_restored with "+
			"their recorded value and the verdict is partial", len(unlinked)))
	}
	if len(components) > 0 {
		notes = append(notes, fmt.Sprintf("re-creates %d nested component(s) one add per component, parents "+
			"first: the station rejects an add that nests components", len(components)),
			"a child that already exists on a freshly created component is a frozen "+
				"slot (e.g. a writable's proxyExt): it is never re-added and is listed "+
				"as frozen_children_not_restored; its configuration is not written. The "+
				"read-back compares it with the snapshot and, when it differs, lists it "+
				"in frozen_config_not_restored and the verdict is partial")
	}
	if len(own) != len(ops) { // ops is never empty: no add means no inverse either
		notes = append(notes, "the rollback itself has no automatic inverse for slot restores or removals")
	}
	return notes
}

// laterWorkRefusal is the refusal of a create rollback that would remove later work: the later
// batches (newest first) and the children or links its create did not make.
func laterWorkRefusal(bid, created string, batches, extras []string) string {
	var found []string
	if len(batches) > 0 {
		found = append(found, "later batch(es) wrote under it: "+strings.Join(batches, ", "))
	}
	if len(extras) > 0 {
		found = append(found, "it holds child(ren) or link(s) its create did not make: "+strings.Join(extras, ", "))
	}
	return "batch " + bid + " created " + created + "; rolling it back removes it with everything under it, and " +
		strings.Join(found, "; ") + ". Roll back the later batches first (newest first) or remove that work by " +
		"hand, then retry: refusing"
}

package writeops

import (
	"fmt"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/safety"
)

// outgoingNote is the plan note of a rollback that re-creates n recorded outgoing links.
func outgoingNote(n int) string {
	return fmt.Sprintf("then re-creates %d outgoing link(s) the remove recorded (stored on components outside the "+
		"removed one, from a source inside it), where both ends exist and the target is the recorded one; the "+
		"read-back lists any missing in outgoing_links and the verdict is mismatch", n)
}

// msgFrozenNotWritten is the frozen-child note of a rollback without frozen_config (no contracts).
const msgFrozenNotWritten = "a child that already exists on a freshly created component is a frozen " +
	"slot (e.g. a writable's proxyExt): it is never re-added and is listed " +
	"as frozen_children_not_restored; its configuration is not written. The " +
	"read-back compares it with the snapshot and, when it differs, lists it " +
	"in frozen_config_not_restored and the verdict is partial"

func rollbackNotes(view *pyjson.Object, bid string, relinks, unlinked, components, own, ops []any) []any {
	frozen := frozenConfigNote(components)
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
		if frozen == "" {
			frozen = msgFrozenNotWritten
		}
		notes = append(notes, fmt.Sprintf("re-creates %d nested component(s) one add per component, parents "+
			"first: the station rejects an add that nests components", len(components)), frozen)
	}
	if len(own) != len(ops) { // ops is never empty: no add means no inverse either
		notes = append(notes, "the rollback itself has no automatic inverse for slot restores or removals")
	}
	return notes
}

// frozenConfigNote is the plan note of a rollback whose component specs carry frozen_config:
// what replaces the "its configuration is not written" of the frozen-child note.
func frozenConfigNote(components []any) string {
	n := 0
	for _, s := range components {
		if _, ok := s.(*pyjson.Object).Get("frozen_config"); ok {
			n++
		}
	}
	if n == 0 {
		return ""
	}
	return fmt.Sprintf("a child that already exists on a freshly created component is a frozen slot (e.g. a "+
		"writable's proxyExt, an alarm extension's offnormalAlgorithm): it is never re-added and is listed as "+
		"frozen_children_not_restored. %d of them carry frozen_config (from the station's type contracts): after "+
		"the parent's add, set ops swap in the captured type and write the captured configuration slots the "+
		"fresh child does not hold; readonly, transient and status slots are never written, transient "+
		"and status slots are not compared (ignored), and a readonly slot that differs stays a gap. A value the station refuses keeps the child in frozen_config_not_restored, "+
		"with the station's reason in refused, and the verdict is partial; a frozen child without frozen_config "+
		"is not written, and when it differs from the snapshot it is listed there too", n)
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
		strings.Join(found, "; ") + ". Roll back the later batches first (newest first) and remove by hand what no " +
		"batch made, then retry; to drop it all instead, n4_remove_component it (its snapshot keeps an inverse): refusing"
}

// budgetRefusal is the refusal of a rollback the write budget cannot hold; the outgoing links are
// named only when there are some (the Python port's text has none: native review R2-budget-label).
func budgetRefusal(need, components, frozen, relinks, outgoing, left, max int) string {
	links := fmt.Sprintf("%d relink(s)", relinks)
	if frozen > 0 {
		links = fmt.Sprintf("%d frozen config write(s) + ", frozen) + links
	}
	if outgoing > 0 {
		links += fmt.Sprintf(" + %d outgoing link(s)", outgoing)
	}
	return fmt.Sprintf("%s: this rollback needs %d write(s) (1 batch + %d component(s) + %s) but only %d remain "+
		"(--max-writes %d)", safety.ReasonBudgetSmall, need, components, links, left, max)
}

package writeops

import (
	"fmt"
	"sort"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// The history ids of a rename (retro R12 ΔM5): a history extension whose historyName holds
// %name% or %parent...name% takes the new name of a renamed component, so its id changes and its
// trend splits. The dry run lists those extensions; freeze_history_names sets each historyName to
// its current text, in the same batch and before the renames.
const (
	msgRenameHistoryIDs = "History ids that change with this rename (%d): %s. Trends split at the rename; " +
		"freeze_history_names: true sets each historyName to its current text in the same batch."
	msgRenameFrozen = "History names frozen to their current text in this batch (%d): %s."
	msgRenameUnread = "The history extensions below the renamed components could not be read, so the history " +
		"ids this rename changes are not listed."
	msgFreezeUnread = "freeze_history_names needs the station's history extensions, which could not be read: %s"
)

// frozenName is one extension whose historyName is frozen: its slot path before and after the
// renames, its handle, and the text it holds (what the pattern resolved to before the renames).
type frozenName struct {
	path, newPath string
	h             any
	pattern, text string
}

// slotPath is the slot path of an ORD ("/Folder/NW"), "" when the ORD names no slot.
func slotPath(ord string) string {
	i := strings.Index(ord, "slot:")
	if i < 0 {
		return ""
	}
	return strings.TrimSuffix(ord[i+len("slot:"):], "/")
}

// renamedPath is path after every item in order (a path below a renamed component follows it).
func renamedPath(items []renameItem, path string) string {
	for _, it := range items {
		old := slotPath(it.parentOrd) + "/" + it.oldName
		if path == old || strings.HasPrefix(path, old+"/") {
			path = slotPath(it.parentOrd) + "/" + it.newName + path[len(old):]
		}
	}
	return path
}

// renameHistory is the extensions whose id changes with items (one read of the station's
// extension table, then the historyName of each one below a renamed component), the note that
// lists them and, when freeze is set, the frozen names. An unreadable table is a fixed note, or
// a refusal when freeze asked for it.
func renameHistory(c plan.Client, items []renameItem, freeze bool) ([]frozenName, []any, error) {
	ids, cut, err := stationHistoryIDs(c)
	if cut && err == nil { // a cut table may hide an extension: never list from it
		err = fmt.Errorf("the extension table was cut at the read cap")
	}
	if err != nil {
		if freeze {
			return nil, nil, plan.ToolError(fmt.Sprintf(msgFreezeUnread, err.Error()))
		}
		return nil, []any{msgRenameUnread}, nil
	}
	byPath := map[string]string{} // slot path -> id, walked in path order: the notes and ops are stable
	paths := make([]string, 0, len(ids))
	for id, slot := range ids {
		byPath[strings.TrimPrefix(slot, "slot:")] = id
		paths = append(paths, strings.TrimPrefix(slot, "slot:"))
	}
	sort.Strings(paths)
	var out []frozenName
	var listed []string
	for _, path := range paths {
		id := byPath[path]
		newPath := renamedPath(items, path)
		if newPath == path {
			continue
		}
		extOrd := "station:|slot:" + path
		h, nodes, err := plan.Handle(c, extOrd, 1)
		if err != nil {
			return nil, nil, err
		}
		if nodes, err = plan.Resolve(c, nodes, "historyName"); err != nil {
			return nil, nil, err
		}
		pattern, err := plan.Observe(nodes, "historyName", "baja:Format")
		if err != nil {
			return nil, nil, err
		}
		oldText, ok1 := ResolveHistoryName(pyjson.Str(pattern), "slot:"+path)
		newText, ok2 := ResolveHistoryName(pyjson.Str(pattern), "slot:"+newPath)
		if !ok1 || !ok2 || oldText == newText {
			continue
		}
		out = append(out, frozenName{path, newPath, h, pyjson.Str(pattern), oldText})
		listed = append(listed, fmt.Sprintf("%s -> %s%s (%s, historyName %s)", id, id[:strings.LastIndex(id, "/")+1],
			newText, path, pyjson.Str(pattern)))
	}
	if len(out) == 0 {
		return nil, nil, nil
	}
	if !freeze {
		return nil, []any{fmt.Sprintf(msgRenameHistoryIDs, len(out), strings.Join(listed, "; "))}, nil
	}
	names := make([]string, len(out))
	for i, f := range out {
		names[i] = f.path + " = " + f.text
	}
	return out, []any{fmt.Sprintf(msgRenameFrozen, len(out), strings.Join(names, "; "))}, nil
}

package livesmoke

import (
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

func insideScratch(ord any) bool {
	s, ok := ord.(string)
	return ok && (s == FolderOrd || strings.HasPrefix(s, FolderOrd+"/"))
}

// AssertInBounds refuses any write outside McpSmoke (the only root writes are its create and
// remove), before anything is sent.
func AssertInBounds(tool string, args *pyjson.Object) error {
	for _, key := range []string{"ord", "source_ord", "target_ord"} {
		if v, ok := args.Get(key); ok && !insideScratch(v) {
			return smokef("refusing %s: %s %s is outside %s", tool, key, pyjson.Repr(v), Scratch)
		}
	}
	parent, ok := args.Get("parent_ord")
	if !ok {
		return nil
	}
	name, _ := args.Get("name")
	atRoot := parent == RootOrd
	if (atRoot && name == Scratch && (tool == "n4_create_component" || tool == "n4_remove_component")) ||
		insideScratch(parent) {
		return nil
	}
	return smokef("refusing %s: %s/%s is outside %s", tool, pyjson.Repr(parent), pyjson.Repr(name), Scratch)
}

package writeops

import (
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/ord"
)

// scopeRootOf is the default link-scan root of a remove (bench retro 2026-11-15 delta 3): the
// outermost ancestor of parent that the write scope still admits, i.e. the configured prefix that
// holds it (the operator's work area). parent itself is in scope (the remove's scope check).
func scopeRootOf(scope *ord.Scope, parent string) string {
	root := parent
	for {
		up, ok := ordParent(root)
		if !ok || scope.Check(up) != nil {
			return root
		}
		root = up
	}
}

// ordParent is the slot ORD one level above o; false at the station root.
func ordParent(o string) (string, bool) {
	t := strings.TrimRight(o, "/")
	if len(t) < len(ord.SlotOrdPrefix) {
		return "", false
	}
	up := t[:strings.LastIndex(t, "/")]
	if len(up) < len(ord.SlotOrdPrefix) {
		up = ord.SlotOrdPrefix
	}
	return up, true
}

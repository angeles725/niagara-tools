package writeops

import (
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// flagsEdit is what an n4_set_flags call asks for: the whole new set (flags), or letters to add
// to and remove from the set the slot holds, so a caller need not restate letters it does not know.
type flagsEdit struct {
	whole              bool
	flags, add, remove string
}

// flagsEditOf reads and checks the form of the call: exactly one of flags or add/remove, each a
// string of known letters given once, and no letter both added and removed.
func flagsEditOf(args *pyjson.Object) (flagsEdit, error) {
	_, hasFlags := args.Get("flags")
	_, hasAdd := args.Get("add")
	_, hasRemove := args.Get("remove")
	switch {
	case hasFlags && (hasAdd || hasRemove):
		return flagsEdit{}, plan.ToolError("give either flags or add/remove, not both")
	case hasFlags:
		flags := str(args, "flags")
		return flagsEdit{whole: true, flags: flags}, checkFlags(flags)
	case !hasAdd && !hasRemove:
		return flagsEdit{}, plan.ToolError("give flags (the whole new set) or add and/or remove (letters to change)")
	}
	e := flagsEdit{add: str(args, "add"), remove: str(args, "remove")}
	for _, part := range [][2]string{{"add", e.add}, {"remove", e.remove}} {
		if err := checkFlagLetters(part[0], part[1]); err != nil {
			return flagsEdit{}, err
		}
	}
	if e.add+e.remove == "" {
		return flagsEdit{}, plan.ToolError("add and remove name no letter: nothing to change")
	}
	for _, r := range e.add {
		if strings.ContainsRune(e.remove, r) {
			return flagsEdit{}, plan.ToolError("flag letter " + pyjson.StrRepr(string(r)) +
				" is both in add and in remove; give it in one of them")
		}
	}
	return e, nil
}

// result is the flag set the call asks the slot to hold, in table order; a letter of the current set
// the table does not know (a newer station's) is kept unless removed.
func (e flagsEdit) result(current string) string {
	if e.whole {
		return canonicalFlags(e.flags)
	}
	kept := ""
	for _, r := range current {
		if !strings.ContainsRune(e.remove, r) {
			kept += string(r)
		}
	}
	return canonicalFlags(kept + e.add)
}

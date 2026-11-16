package boxstation

import (
	"fmt"
	"strings"
)

// flagLetters are the slot flag letters in the order a station encodes them
// (javax.baja.sys.Flags.encodeToString).
const flagLetters = "rthsandcoxfApRmNL1234"

// setFlags applies a set-flags op (javax.baja.sync.SetFlagsOp, id 'f') to slot name of parent:
// the whole flag set is replaced, a letter the station does not know is ignored
// (Flags.decodeFromString) and the answer is the op id, as syncToMaster writes it.
func setFlags(parent *Node, name string, op map[string]any) (any, error) {
	flags, ok := op["f"].(string)
	slot := parent.Child(name)
	if slot == nil || !ok {
		return nil, fmt.Errorf("NullPointerException: set flags of '%s'", name)
	}
	slot.Flags = ""
	for _, letter := range flagLetters {
		if strings.ContainsRune(flags, letter) {
			slot.Flags += string(letter)
		}
	}
	return []any{map[string]any{"id": "f"}}, nil
}

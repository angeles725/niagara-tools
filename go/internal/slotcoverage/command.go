package slotcoverage

import (
	"fmt"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

const (
	usageSet     = "usage: slot-coverage.sh set-coverage <declared-csv> <required-csv>\n"
	usagePerSlot = "usage: slot-coverage.sh per-slot <module-include.xml> <module.lexicon> <src-dir>\n"
	usageParse   = "usage: slot-coverage.sh [--strict] [--facade] <module-include.xml> <module.lexicon>\n"
)

// Main is slot-coverage.sh: `per-slot` and `set-coverage` are read before any flag, then the parse mode's
// --strict and --facade (`--` ends them, any other "-..." is an unknown flag, exit 2).
func Main(args []string, s cli.Stdio) int {
	if len(args) >= 1 && args[0] == "per-slot" {
		if len(args) != 4 {
			fmt.Fprint(s.Err, usagePerSlot)
			return 2
		}
		return perSlot(args[1], args[2], args[3], s)
	}
	if len(args) >= 1 && args[0] == "set-coverage" {
		if len(args) != 3 {
			fmt.Fprint(s.Err, usageSet)
			return 2
		}
		fmt.Fprintln(s.Out, strings.Join(SetCoverage(args[1], args[2]), "\n"))
		return 0
	}
	strict, facade := false, false
flags:
	for len(args) > 0 {
		switch a := args[0]; {
		case a == "--strict":
			strict = true
		case a == "--facade":
			facade = true
		case a == "--":
			args = args[1:]
			break flags
		case strings.HasPrefix(a, "-"):
			fmt.Fprintf(s.Err, "slot-coverage: unknown flag: %s\n%s", a, usageParse)
			return 2
		default:
			break flags
		}
		args = args[1:]
	}
	if len(args) != 2 {
		fmt.Fprint(s.Err, usageParse)
		return 2
	}
	if !isFile(args[0]) {
		fmt.Fprintf(s.Err, "slot-coverage: not found: %s\n", args[0])
		return 3
	}
	return parseMode(args[0], args[1], strict, facade, s)
}

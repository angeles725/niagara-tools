package checkclientsource

import "strings"

// options is the parsed command line.
type options struct {
	declared     string // --declared <value>
	haveDeclared bool   // --declared was given (it wins over --build-state, even when empty)
	buildState   string // --build-state <file>
	haveState    bool   // --build-state was given (an empty <file> names nothing: a usage error when it is read)
	module       string // --module <MOD>
	root         string // the one <module-root>
	help         bool   // -h or --help was met before any parse error
}

// parseArgs reads the options up to the first argument that is not one (or past `--`), then exactly one
// module root. ok is false on an unknown option, an option with no value or a wrong root count.
func parseArgs(args []string) (o options, ok bool) {
	i := 0
options:
	for ; i < len(args); i++ {
		var value *string
		switch a := args[i]; {
		case a == "--declared":
			value, o.haveDeclared = &o.declared, true
		case a == "--build-state":
			value, o.haveState = &o.buildState, true
		case a == "--module":
			value = &o.module
		case a == "-h" || a == "--help":
			o.help = true
			return o, true
		case a == "--":
			i++
			break options
		case strings.HasPrefix(a, "-"):
			return o, false
		default:
			break options
		}
		if i+1 >= len(args) {
			return o, false
		}
		i++
		*value = args[i]
	}
	if len(args)-i != 1 {
		return o, false
	}
	o.root = args[i]
	return o, true
}

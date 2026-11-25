package foldaudit

import (
	"regexp"
	"strings"
)

var dateArg = regexp.MustCompile(`^[0-9]{4}-[0-9]{2}-[0-9]{2}$`)

// options is the parsed command line.
type options struct {
	strict, exactStems bool
	noResearchProbe    bool // --no-research-probe: skip the degraded() probe of niagara-research/retros
	since, index, root string
	externals          []string          // --external-root, then fold-roots.conf: retro directories outside the kit
	aliases            map[string]string // retro stem -> alias, the name an external retro is reported by
}

// parseArgs reads `[--strict] [--exact-stems] [--no-research-probe] [--deltas-since <YYYY-MM-DD>]
// [--external-root <dir>]... [--alias <stem>=<ALIAS>]... <INDEX.md> <kit-root>`; on a usage
// error it returns the exact stderr text of the bash tool.
func parseArgs(args []string) (options, string) {
	var o options
	i := 0
loop:
	for i < len(args) {
		a := args[i]
		switch {
		case a == "--strict":
			o.strict = true
			i++
		case a == "--exact-stems":
			o.exactStems = true
			i++
		case a == "--no-research-probe":
			o.noResearchProbe = true
			i++
		case a == "--deltas-since":
			if i+1 >= len(args) || !dateArg.MatchString(args[i+1]) {
				return options{}, "fold-audit: --deltas-since needs a YYYY-MM-DD date\n" + usage + "\n"
			}
			o.since = args[i+1]
			i += 2
		case a == "--external-root":
			if i+1 >= len(args) {
				return options{}, "fold-audit: --external-root needs a directory\n" + usage + "\n"
			}
			o.externals = append(o.externals, args[i+1])
			i += 2
		case a == "--alias":
			stem, alias, ok := strings.Cut(argAt(args, i+1), "=")
			if !ok || stem == "" || !aliasName.MatchString(alias) {
				return options{}, "fold-audit: --alias needs <stem>=<ALIAS> (ALIAS: letters, digits, . _ -)\n" + usage + "\n"
			}
			o.setAlias(stem, alias)
			i += 2
		case a == "--":
			i++
			break loop
		case strings.HasPrefix(a, "-"):
			return options{}, "fold-audit: unknown flag: " + a + "\n" + usage + "\n"
		default:
			break loop
		}
	}
	if len(args)-i != 2 {
		return options{}, usage + "\n"
	}
	o.index, o.root = args[i], args[i+1]
	return o, ""
}

// aliasName is what an external retro may be called in a report and in a citation.
var aliasName = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

func argAt(args []string, i int) string {
	if i < len(args) {
		return args[i]
	}
	return ""
}

// setAlias records an alias unless the stem already has one (the first --alias, then the config file's).
func (o *options) setAlias(stem, alias string) {
	if o.aliases == nil {
		o.aliases = map[string]string{}
	}
	if _, set := o.aliases[stem]; !set {
		o.aliases[stem] = alias
	}
}

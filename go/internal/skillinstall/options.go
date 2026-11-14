package skillinstall

import "strings"

// installOptions is the install-skill command line.
type installOptions struct {
	skill, home   string
	dryRun, force bool
}

// parseInstall reads the install-skill arguments. A malformed line returns its message, and whether
// the usage line follows it.
func parseInstall(args []string) (o installOptions, message string, withUsage bool) {
	o.skill = defaultSkill
	for i := 0; i < len(args); i++ {
		switch arg := args[i]; {
		case arg == "--home" || arg == "--skill":
			if i+1 >= len(args) {
				return o, arg + " requires an argument", false
			}
			i++
			if arg == "--home" {
				o.home = args[i]
			} else {
				o.skill = args[i]
			}
		case arg == "--dry-run":
			o.dryRun = true
		case arg == "--force":
			o.force = true
		case arg == "--":
			return o, "", false // the rest is not read
		case strings.HasPrefix(arg, "-"):
			return o, "unknown option: " + arg, true
		default:
			return o, "unexpected argument: " + arg, true
		}
	}
	return o, "", false
}

// driftOptions is the check-skill-drift command line.
type driftOptions struct {
	skill, home string
	help        bool
}

// parseDrift reads the check-skill-drift arguments left to right; ok is false on a malformed line.
// --help ends the reading where it stands, as the shell loop did.
func parseDrift(args []string, home string) (o driftOptions, ok bool) {
	o.skill, o.home = defaultSkill, home
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--skill", "--home":
			if i+1 >= len(args) {
				return o, false
			}
			if args[i] == "--skill" {
				o.skill = args[i+1]
			} else {
				o.home = args[i+1]
			}
			i++
		case "-h", "--help":
			o.help = true
			return o, true
		default:
			return o, false
		}
	}
	return o, true
}

package skillinstall

import "strings"

// emptyHome is the message for `--home ""`: an empty value would fall back to $HOME, the real home.
const emptyHome = "--home requires a non-empty directory"

// installOptions is the install-skill command line.
type installOptions struct {
	skill, home, harness string
	dryRun, force        bool
	homeGiven            bool // --home was on the line: it wins over a harness's directory variable
}

// parseInstall reads the install-skill arguments. A malformed line returns its message, and whether
// the usage line follows it.
func parseInstall(args []string) (o installOptions, message string, withUsage bool) {
	o.skill, o.harness = defaultSkill, defaultHarness
	for i := 0; i < len(args); i++ {
		switch arg := args[i]; {
		case arg == "--home" || arg == "--skill" || arg == "--harness":
			if i+1 >= len(args) {
				return o, arg + " requires an argument", false
			}
			i++
			switch arg {
			case "--home":
				if args[i] == "" {
					return o, emptyHome, true
				}
				o.home, o.homeGiven = args[i], true
			case "--skill":
				o.skill = args[i]
			default:
				o.harness = args[i]
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

// driftOptions is the check-skill-drift command line. explicit is whether --harness or --all was given:
// it adds the harness to each row's skill column.
type driftOptions struct {
	skill, home, harness string
	help, explicit       bool
	homeGiven            bool // --home was on the line: it wins over a harness's directory variable
}

// parseDrift reads the check-skill-drift arguments left to right; a malformed line returns a non-empty
// message ("" for one that only needs the usage line). --help ends the reading where it stands, as the
// shell loop did.
func parseDrift(args []string, home string) (o driftOptions, ok bool, message string) {
	o.skill, o.home, o.harness = defaultSkill, home, defaultHarness
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--skill", "--home", "--harness":
			if i+1 >= len(args) {
				return o, false, ""
			}
			switch args[i] {
			case "--skill":
				o.skill = args[i+1]
			case "--home":
				if args[i+1] == "" {
					return o, false, emptyHome
				}
				o.home, o.homeGiven = args[i+1], true
			default:
				o.harness, o.explicit = args[i+1], true
			}
			i++
		case "--all":
			o.harness, o.explicit = allHarnesses, true
		case "-h", "--help":
			o.help = true
			return o, true, ""
		default:
			return o, false, ""
		}
	}
	return o, true, ""
}

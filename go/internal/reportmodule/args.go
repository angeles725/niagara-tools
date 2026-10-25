package reportmodule

// usage is the one usage line of report-module (stderr, exit 2).
const usage = "usage: report-module.sh <module-root> [--target-version x.y] [--console-dir <dir>] [--profile hmi|lan|both|unknown] [--legacy] [--wiring-map <file>] [--first-deploy] [--lint-only]\n"

// options are the parsed arguments. An empty value is the same as an absent flag (the bash tests were
// `[ -n "$X" ]`), except for --profile, whose value must be one of the four profiles.
type options struct {
	root, targetVersion, consoleDir, profile, wiringMap string
	legacy, firstDeploy, lintOnly                       bool
}

// profiles are the --profile values rc-scan accepts.
var profiles = map[string]bool{"hmi": true, "lan": true, "both": true, "unknown": true}

// parseArgs reads the arguments like the bash loop: a value flag needs its value, `--` ends the scan (what
// follows it is ignored, not a positional), any other word starting with "-" and a second positional are usage
// errors, and a missing <module-root> is one too.
func parseArgs(args []string) (o options, ok bool) {
	values := map[string]*string{
		"--target-version": &o.targetVersion, "--console-dir": &o.consoleDir,
		"--profile": &o.profile, "--wiring-map": &o.wiringMap,
	}
	switches := map[string]*bool{"--legacy": &o.legacy, "--first-deploy": &o.firstDeploy, "--lint-only": &o.lintOnly}
	for i := 0; i < len(args); i++ {
		a := args[i]
		if v := values[a]; v != nil {
			if i+1 == len(args) || (a == "--profile" && !profiles[args[i+1]]) {
				return o, false
			}
			i++
			*v = args[i]
			continue
		}
		if b := switches[a]; b != nil {
			*b = true
			continue
		}
		switch {
		case a == "--":
			return o, o.root != ""
		case a != "" && a[0] == '-', o.root != "":
			return o, false
		default:
			o.root = a // an empty word is taken too, and a later positional may replace it
		}
	}
	return o, o.root != ""
}

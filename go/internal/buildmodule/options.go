package buildmodule

import (
	"fmt"
	"io"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/gradleenv"
)

// options is the command line: flags first, then <module-root> <MOD> [niagara_home].
type options struct {
	profiles  string // --profiles, comma separated; "" auto-detects
	target    string // --target-version
	plugin    string // --plugin-version, else $NIAGARA_PLUGIN_VERSION
	uiProfile string // --ui-profile
	restore   string // --restore-mirror <kept jar>: put a jar a mirror build replaced back, and nothing else
	// baselineRef is --schema-baseline-ref: the commit whose artifacts are the schema-risk baseline (baselineref.go).
	baselineRef  string
	buildState   string // --build-state: the file holding the module's build-state.v1 envelope (signing_expected.go)
	deployTarget string // --target: the station whose recorded deploy (.deploy-targets) is the baseline (deploytarget.go)
	// gradleProps are the --gradle-prop values as gradle takes them (-Pk=v), in order.
	gradleProps                                                                 []string
	legacy, werror, firstDeploy, intoHome, skipPreflight, skipReport, skipDrift bool
	root, mod, home                                                             string
}

// switches are the flags without a value.
func (o *options) switches() map[string]*bool {
	return map[string]*bool{
		"--legacy": &o.legacy, "--werror": &o.werror, "--first-deploy": &o.firstDeploy,
		"--install-into-home": &o.intoHome, "--no-preflight": &o.skipPreflight,
		"--no-report": &o.skipReport, "--no-drift-check": &o.skipDrift,
	}
}

// values are the flags whose value is taken as is.
func (o *options) values() map[string]*string {
	return map[string]*string{"--profiles": &o.profiles, "--target-version": &o.target, "--plugin-version": &o.plugin,
		"--restore-mirror": &o.restore, "--schema-baseline-ref": &o.baselineRef,
		"--target": &o.deployTarget, "--build-state": &o.buildState}
}

// parseArgs reads the command line. code is proceed, or the exit code after the usage or a message was
// printed. Flags stop at the first argument that does not start with "-".
func parseArgs(args []string, s cli.Stdio) (o options, code int) {
	o.plugin = s.Env("NIAGARA_PLUGIN_VERSION")
	switches, values := o.switches(), o.values()
	usageError := func() (options, int) { io.WriteString(s.Err, usageText(s)); return o, ExitUsage }
	for len(args) > 0 && strings.HasPrefix(args[0], "-") {
		flag, hasValue := args[0], len(args) >= 2
		switch {
		case switches[flag] != nil:
			*switches[flag] = true
			args = args[1:]
			continue
		case flag == "-h" || flag == "--help":
			io.WriteString(s.Out, usageText(s))
			return o, ExitOK
		case flag == gradleenv.PropFlag:
			prop, ok := "", false
			if hasValue {
				prop, ok = gradleenv.Prop(args[1])
			}
			if !ok {
				got := "nothing"
				if hasValue && args[1] != "" {
					got = args[1]
				}
				fmt.Fprintf(s.Err, msgGradleProp+"\n", got)
				return o, ExitUsage
			}
			o.gradleProps = append(o.gradleProps, prop)
		case values[flag] != nil || flag == "--ui-profile":
			if !hasValue || (flag == "--build-state" && args[1] == "") {
				return usageError() // an empty --build-state is no file, not the default one
			}
			if flag == "--ui-profile" {
				if !uiProfiles[args[1]] {
					fmt.Fprintf(s.Err, msgUIProfile+"\n", args[1])
					return o, ExitUsage
				}
				o.uiProfile = args[1]
			} else {
				*values[flag] = args[1]
			}
		default:
			fmt.Fprintf(s.Err, msgUnknownFlag+"\n", flag)
			return usageError()
		}
		args = args[2:]
	}
	if o.restore != "" {
		if len(args) != 0 {
			return usageError() // --restore-mirror runs alone: it builds nothing
		}
		return o, proceed
	}
	if len(args) < 2 || len(args) > 3 {
		return usageError()
	}
	if o.baselineRef != "" && o.firstDeploy {
		fmt.Fprintln(s.Err, msgBaselineRefFirstDeploy) // a never-deployed module has no deployed ref
		return o, ExitUsage
	}
	if o.deployTarget != "" && (o.baselineRef != "" || o.firstDeploy) {
		fmt.Fprintln(s.Err, msgTargetExclusive)
		return o, ExitUsage
	}
	o.root, o.mod, o.home = args[0], args[1], s.Env("niagara_home")
	if len(args) == 3 && args[2] != "" {
		o.home = args[2]
	}
	return o, proceed
}

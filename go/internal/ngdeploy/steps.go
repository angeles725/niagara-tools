package ngdeploy

// steps are the single steps the launcher offers a test that sources it with BATS_TEST_MODE=1, as the bash
// tool exposed its functions: tests/ng-deploy.bats calls verify_jar, verify_cachebuster and backup on their
// own. A step reads its settings from the environment only (no flags, no env file).
var steps = map[string]func(d *deploy, args []string) int{
	"verify_jar": func(d *deploy, args []string) int {
		if len(args) != 2 || !wholeNumber.MatchString(args[1]) {
			return d.fail(ExitEnv, msgStepUsage, "verify_jar <jar> <expected-count>")
		}
		return d.verifyJar(args[0], args[1])
	},
	"verify_cachebuster": func(d *deploy, args []string) int {
		if len(args) != 1 {
			return d.fail(ExitEnv, msgStepUsage, "verify_cachebuster <jar>")
		}
		return d.verifyCacheBuster(args[0])
	},
	"backup": func(d *deploy, _ []string) int {
		for _, name := range []string{"MODULE_NAME", "STATION_MODULES_DIR"} {
			if d.env(name) == "" {
				return d.fail(ExitEnv, msgRequired, name)
			}
		}
		if code := d.checkOptions(); code != proceed {
			return code
		}
		d.module, d.station = d.env("MODULE_NAME"), d.env("STATION_MODULES_DIR")
		return d.archive()
	},
}

// runStep runs one step and returns its exit code (0 where the deploy would go on).
func (d *deploy) runStep(name string, args []string) int {
	step := steps[name]
	if step == nil {
		return d.fail(ExitEnv, msgUnknownStep, name)
	}
	if code := step(d, args); code != proceed {
		return code
	}
	return ExitOK
}

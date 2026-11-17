package ngdeploy

// envGuard is what makes a line of the env file switch a guarded step off: the name the refusal gives
// the line, the step, the flag the operator gives to mean the same thing on the command line, and the
// variable whose flag, given, lets the line stand.
type envGuard struct {
	label, step, flag, liftedBy string
}

// guardedSkip reports whether the env-file line key=value would switch a guarded step off by itself
// (loadEnvFile refuses it with file and line). The guarded switches of the option table (NO_GATE,
// NO_BACKUP, INSTALL_INTO_HOME) at 1; MODE=B, a mode without the verify-module gate, unless --mode is given
// (the flag then wins over the file); a VERIFY_MODULE_BIN that is not empty, which puts any executable in
// the place of the gate, unless --no-gate is given (the gate does not run then). The environment of the
// command still sets VERIFY_MODULE_BIN (retro go-ng-deploy-port Δ12).
func guardedSkip(key, value string) (envGuard, bool) {
	switch key {
	case optMode:
		return envGuard{optMode + "=" + modeNoGate, guardGate, "--mode " + modeNoGate, optMode}, value == modeNoGate
	case optVerifyBin:
		return envGuard{optVerifyBin, guardGate, "--no-gate", optNoGate}, value != ""
	}
	option, _ := optionOf(key)
	return envGuard{key, option.guards, option.flag, key}, option.guards != "" && isOne(value)
}

// modeNoGate is the mode that runs no verify-module gate (the gate runs in modes A and C).
const modeNoGate = "B"

// guardGate is what NO_GATE, MODE=B and VERIFY_MODULE_BIN switch off (msgEnvFileGuarded).
const guardGate = "the verify-module gate"

// replacesSchemaTool reports whether the env-file line key=value would put another program in the place of
// the schema-risk precheck. No flag lifts it: the environment of the command still sets SCHEMA_RISK_BIN
// (retro p1-build-deploy-safety Δ5).
func replacesSchemaTool(key, value string) bool { return key == optSchemaBin && value != "" }

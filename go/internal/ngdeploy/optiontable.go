package ngdeploy

// flagOption is one row of the option table: a flag of the command line and the option variable it sets.
type flagOption struct {
	flag     string
	variable string
	value    bool   // the flag takes the next argument; without it the flag is a switch and sets "1"
	guards   string // the step the variable switches off when it is 1: the env file may not do that by itself
}

// optionTable is every flag that sets an option variable: the values first, then the switches in the order
// their values are checked. A variable a flag set is not replaced by the env file (the flag wins), and a
// guarded one is refused in the env file unless its flag was given (envguard.go). VERIFY_MODULE_BIN and
// SCHEMA_RISK_BIN are option variables with no flag: the environment or the env file sets them.
var optionTable = []flagOption{
	{flag: "--mode", variable: optMode, value: true},
	{flag: "--env-file", variable: optEnvFile, value: true},
	{flag: "--keep", variable: optKeep, value: true},
	{flag: "--target", variable: optTarget, value: true},
	{flag: "--no-backup", variable: optNoBackup, guards: "the backup"},
	{flag: "--no-deploy", variable: optNoDeploy},
	{flag: "--with-slotomatic", variable: optWithSlot},
	{flag: "--strict-slotomatic", variable: optStrictSlot},
	{flag: "--no-gate", variable: optNoGate, guards: guardGate},
	{flag: "--first-deploy", variable: optFirstDeploy},
	{flag: "--accept-lossy", variable: optAcceptLossy},
	{flag: "--full-backup", variable: optFullBackup},
	{flag: intoHomeFlag, variable: optIntoHome, guards: guardRealHome},
}

// switches are the variables that hold 0 or 1, in the order of the table.
var switches = func() (names []string) {
	for _, option := range optionTable {
		if !option.value {
			names = append(names, option.variable)
		}
	}
	return names
}()

// optionOf is the row of the table whose flag or variable is name.
func optionOf(name string) (flagOption, bool) {
	for _, option := range optionTable {
		if option.flag == name || option.variable == name {
			return option, true
		}
	}
	return flagOption{}, false
}

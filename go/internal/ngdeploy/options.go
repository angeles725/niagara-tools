package ngdeploy

import (
	"io"
	"strconv"
)

// The option globals of the bash tool. Each holds text: its default, then what the env file assigned, then
// what a flag set (the bash tool sourced the file after reading the flags, so a key replaced the flag; a
// flag wins now). optiontable.go pairs them with their flags.
const (
	optMode        = "MODE"
	optNoBackup    = "NO_BACKUP"
	optNoDeploy    = "NO_DEPLOY"
	optEnvFile     = "ENV_FILE"
	optWithSlot    = "WITH_SLOTOMATIC"
	optStrictSlot  = "STRICT_SLOTOMATIC"
	optNoGate      = "NO_GATE"
	optFirstDeploy = "FIRST_DEPLOY"
	optAcceptLossy = "ACCEPT_LOSSY"
	optKeep        = "KEEP_N"
	optFullBackup  = "FULL_BACKUP"
	optVerifyBin   = "VERIFY_MODULE_BIN"
	optSchemaBin   = "SCHEMA_RISK_BIN"
)

// noOpFlag is accepted and ignored: it once had to accompany --no-backup.
const noOpFlag = "--i-know-what-im-doing"

// Defaults of the globals that the environment cannot set.
const (
	defaultMode    = "A"
	defaultEnvFile = ".env.local"
	defaultKeep    = "3"
	off, on        = "0", "1"
)

// The tools the deploy runs, beside the scripts directory unless the environment names another.
const (
	verifyModulePath = "/../build-n4-module-kit/toolbelt/verify-module.sh"
	schemaRiskPath   = "/../build-n4-module-kit/toolbelt/schema-risk.sh"
)

// defaults are the option globals before the flags are read. FIRST_DEPLOY and the other switches never
// come from the environment: a variable left exported in the operator's shell must not skip a precheck.
// KEEP_N, FULL_BACKUP and the two tool paths do, when not empty.
func (d *deploy) defaults() map[string]string {
	orElse := func(name, fallback string) string {
		if v := d.s.Env(name); v != "" {
			return v
		}
		return fallback
	}
	raw := map[string]string{
		optMode: defaultMode, optEnvFile: defaultEnvFile,
		optKeep: orElse(optKeep, defaultKeep), optFullBackup: orElse(optFullBackup, off),
		optVerifyBin: orElse(optVerifyBin, d.scriptDir+verifyModulePath),
		optSchemaBin: orElse(optSchemaBin, d.scriptDir+schemaRiskPath),
	}
	for _, name := range switches {
		if _, set := raw[name]; !set {
			raw[name] = off
		}
	}
	return raw
}

// parseArgs reads the command line into d.raw, in order: --help and --version end the run where they
// stand. It returns proceed, or the exit code after the usage, the version or an error was printed.
func (d *deploy) parseArgs(args []string) int {
	for len(args) > 0 {
		flag := args[0]
		switch {
		case flag == "--help":
			io.WriteString(d.s.Out, usage)
			return ExitOK
		case flag == "--version":
			io.WriteString(d.s.Out, d.versionText()+"\n")
			return ExitOK
		case flag == noOpFlag:
		case d.setByFlag(flag, args[1:]):
			if option, _ := optionOf(flag); option.value {
				args = args[1:]
			}
		case isFlag(flag):
			return d.fail(ExitEnv, msgNeedsArgument, flag)
		default:
			code := d.fail(ExitEnv, msgUnknownFlag, flag)
			io.WriteString(d.s.Out, usage)
			return code
		}
		args = args[1:]
	}
	if profilesOf[d.raw[optMode]] == nil {
		return d.fail(ExitEnv, msgInvalidMode, d.raw[optMode])
	}
	return proceed
}

// setByFlag sets the option variable of a flag of the table from the command line (rest is what follows
// the flag) and notes that a flag set it. It reports false for a word that is not such a flag, and for a
// flag left without its value.
func (d *deploy) setByFlag(flag string, rest []string) bool {
	option, ok := optionOf(flag)
	if !ok || option.flag != flag || (option.value && len(rest) == 0) {
		return false
	}
	value := on
	if option.value {
		value = rest[0]
	}
	d.raw[option.variable], d.byFlag[option.variable] = value, true
	return true
}

// isFlag reports whether a word of the command line is a flag of the table.
func isFlag(word string) bool {
	option, ok := optionOf(word)
	return ok && option.flag == word
}

// isOn reads a switch global as bash's [[ "$X" -eq 1 ]] did for a value made of digits ("" is 0).
func (d *deploy) isOn(name string) bool { return isOne(d.raw[name]) }

// isOne is that reading of one value.
func isOne(value string) bool {
	n, err := strconv.Atoi(value)
	return err == nil && n == 1
}

// checkOptions refuses option values bash could not compare as numbers, and a mode or a keep count that
// would deploy or keep nothing. The mode of a flag was checked when read; the env file may have set one.
func (d *deploy) checkOptions() int {
	if profilesOf[d.raw[optMode]] == nil {
		return d.fail(ExitEnv, msgFileMode)
	}
	for _, name := range switches {
		if d.raw[name] != "" && !digits.MatchString(d.raw[name]) {
			return d.fail(ExitEnv, msgNotSwitch, name)
		}
	}
	if keep := d.raw[optKeep]; !wholeNumber.MatchString(keep) || keep == off {
		return d.fail(ExitEnv, msgKeep)
	}
	return proceed
}

// keep is the number of backups kept per module (checkOptions made it a whole number).
func (d *deploy) keep() int {
	n, _ := strconv.Atoi(d.raw[optKeep])
	return n
}

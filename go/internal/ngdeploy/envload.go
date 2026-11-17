package ngdeploy

import (
	"errors"
	"os"

	"github.com/angeles725/niagara-tools/go/internal/envfile"
	"github.com/angeles725/niagara-tools/go/internal/strerror"
)

// scriptIFS is the IFS the bash tool ran with.
const scriptIFS = "\n\t"

// owned are the names an env file may not assign: the bash tool kept them read-only, or relied on them
// (IFS shaped the lines it printed).
var owned = map[string]bool{"SCRIPT_DIR": true, "VERSION_FILE": true, "SCRIPT_VERSION": true, "IFS": true}

// loadEnvFile reads the env file the way `set -a; source` did for plain assignments: each one becomes part
// of the environment of this process and of every command it runs, and one named like an option global
// sets that option unless a flag of the command line did (the flag wins). A line that would switch the
// backup, the verify-module gate or the refusal of a real niagara_home off is refused with file and line
// unless its flag was given (guardedSkip, envguard.go): the file alone does not skip any of them, and it may not
// name the schema-risk tool (replacesSchemaTool). $NAME, ${NAME} and a leading ~ expand as bash expanded them (shellVariable). A
// missing file is tolerated; a directory is not (exit 10: the wrong path was named). A line that is not a plain assignment, and a variable that is not set, are
// refused with file and line (the bash tool ran the first and stopped on the second); no message quotes a
// value of the file.
func (d *deploy) loadEnvFile() int {
	path := d.raw[optEnvFile]
	var crLines []int
	assignments, found, err := envfile.Load(path, d.shellVariable, envfile.TrimCR(&crLines))
	var syntax *envfile.SyntaxError
	switch {
	case err == nil && !found && isDir(path):
		return d.fail(ExitEnv, msgEnvFileDir, path)
	case errors.As(err, &syntax):
		return d.fail(ExitEnv, msgEnvFileSyntax, syntax.Error())
	case err != nil:
		return d.fail(ExitEnv, msgEnvFileRead, path, strerror.Text(err))
	}
	for _, a := range assignments {
		if owned[a.Key] {
			return d.fail(ExitEnv, msgEnvFileOwned, path, a.Line, a.Key)
		}
		if replacesSchemaTool(a.Key, a.Value) {
			return d.fail(ExitEnv, msgEnvFileSchemaTool, path, a.Line)
		}
		if guard, skips := guardedSkip(a.Key, a.Value); skips && !d.byFlag[guard.liftedBy] {
			return d.fail(ExitEnv, msgEnvFileGuarded, path, a.Line, guard.label, guard.step, guard.flag)
		}
	}
	if len(crLines) > 0 { // a CRLF file: the values are used without the carriage return
		d.warn(warnEnvFileCR, path, len(crLines), crLines[0])
	}
	for _, a := range assignments {
		d.fileEnv[a.Key] = a.Value
		os.Setenv(a.Key, a.Value)
		if _, isOption := d.raw[a.Key]; isOption && !d.byFlag[a.Key] {
			d.raw[a.Key] = a.Value
		}
	}
	return proceed
}

// exportOptions hands the option globals on as bash did: a global the environment already exported (or
// the env file assigned) reaches every command with the value the deploy settled on.
func (d *deploy) exportOptions() int {
	for name, value := range d.raw {
		if _, exported := os.LookupEnv(name); exported {
			os.Setenv(name, value)
		}
	}
	return proceed
}

// env is a setting: the env file's value when it assigns the name, else the environment's.
func (d *deploy) env(name string) string {
	if v, ok := d.fileEnv[name]; ok {
		return v
	}
	return d.s.Env(name)
}

// shellVariable is a variable as the env file saw it when bash sourced it, after the file's own earlier
// lines: the option globals as the flags left them, the variables the script set for itself, then the
// environment. The file was sourced under set -u, so a name that is none of these is refused by envfile.
func (d *deploy) shellVariable(name string) (value string, set bool) {
	if v, ok := d.raw[name]; ok {
		return v, true
	}
	switch name {
	case "SCRIPT_DIR":
		return d.scriptDir, true
	case "VERSION_FILE":
		return d.scriptDir + versionFile, true
	case "SCRIPT_VERSION":
		return d.versionText(), true
	case "IFS":
		return scriptIFS, true
	}
	return os.LookupEnv(name)
}

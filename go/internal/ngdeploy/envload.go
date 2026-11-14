package ngdeploy

import (
	"errors"
	"os"

	"github.com/angeles725/niagara-tools/go/internal/envfile"
	"github.com/angeles725/niagara-tools/go/internal/strerror"
)

// owned are the names an env file may not assign: the bash tool kept them read-only, or relied on them
// (IFS shaped the lines it printed).
var owned = map[string]bool{"SCRIPT_DIR": true, "VERSION_FILE": true, "SCRIPT_VERSION": true, "IFS": true}

// loadEnvFile reads the env file the way `set -a; source` did for plain assignments: each one becomes part
// of the environment of this process and of every command it runs, and one named like an option global
// replaces that option. A missing file is tolerated. A line that is not a plain assignment is refused with
// its file and line (the bash tool ran it); no message quotes a value of the file.
func (d *deploy) loadEnvFile() int {
	path := d.raw[optEnvFile]
	assignments, _, err := envfile.Load(path)
	var syntax *envfile.SyntaxError
	switch {
	case errors.As(err, &syntax):
		return d.fail(ExitEnv, msgEnvFileSyntax, syntax.Error())
	case err != nil:
		return d.fail(ExitEnv, msgEnvFileRead, path, strerror.Text(err))
	}
	for _, a := range assignments {
		if owned[a.Key] {
			return d.fail(ExitEnv, msgEnvFileOwned, path, a.Line, a.Key)
		}
	}
	for _, a := range assignments {
		d.fileEnv[a.Key] = a.Value
		os.Setenv(a.Key, a.Value)
		if _, isOption := d.raw[a.Key]; isOption {
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

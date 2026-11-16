// Package ngdeploy is scripts/ng-deploy.sh: it backs up, builds, checks and copies a Niagara module's jars
// into a station modules directory. It follows the bash tool it replaces line for line: the same lines on
// the same stream, the same exit codes, and the same commands run with the same arguments and environment.
// The differences are declared in changelog.d/2026-11-14-ng-deploy-go.md, and the defects of the bash tool
// fixed since in changelog.d/2026-11-14-ng-deploy-silent-exits.md.
//
// One file per responsibility, in deploy order: options.go (flags and option globals), envload.go (the env
// file), config.go (required settings and paths), backup.go and purge.go, annotations.go (the slotomatic
// reminder), slotomatic.go, build.go, version.go and schema.go (the two prechecks), verify.go, gate.go,
// copy.go, record.go (the deploy record and the restart reminder). The data lives apart: exit.go, messages.go, patterns.go, usage.txt.
// Every command goes through proc.go. This file only composes them.
package ngdeploy

import (
	"fmt"
	"os"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

// ScriptEnv is the path of the invoked ng-deploy.sh: VERSION and the default verify-module and schema-risk
// tools are found from its directory. The launcher sets it; it is not handed on to the commands the deploy
// runs.
const ScriptEnv = "NG_DEPLOY_SCRIPT"

// deploy is one run: what the phases before decided, for the phases after.
type deploy struct {
	s         cli.Stdio
	scriptDir string            // the directory of the invoked ng-deploy.sh
	raw       map[string]string // the option globals (options.go)
	fileEnv   map[string]string // what the env file assigned

	module    string // MODULE_NAME
	station   string // STATION_MODULES_DIR
	moduleDir string // <directory of gradlew>/<MODULE_NAME>
}

// phases is the deploy after the command line was read; each returns an exit code, or proceed.
var phases = []func(*deploy) int{
	(*deploy).loadEnvFile,
	(*deploy).checkSettings,
	(*deploy).exportOptions,
	(*deploy).backup,
	(*deploy).slotomatic,
	(*deploy).build,
	(*deploy).precheckVersion,
	(*deploy).precheckSchema,
	(*deploy).verify,
	(*deploy).gate,
	(*deploy).copyJars,
	(*deploy).recordDeploy,
}

// Main implements ng-deploy.sh; usage.txt carries the usage text and the exit codes.
func Main(args []string, s cli.Stdio) int {
	d := &deploy{s: s, scriptDir: dirname(s.Env(ScriptEnv)), fileEnv: map[string]string{}}
	os.Unsetenv(ScriptEnv)
	d.raw = d.defaults()
	if code := d.parseArgs(args); code != proceed {
		return code
	}
	for _, phase := range phases {
		if code := phase(d); code != proceed {
			return code
		}
	}
	return ExitOK
}

// say prints one stdout line (a format of messages.go) after the tool's prefix.
func (d *deploy) say(format string, a ...any) { fmt.Fprintf(d.s.Out, prefix+format+"\n", a...) }

// warn prints one stderr line after the tool's prefix.
func (d *deploy) warn(format string, a ...any) { fmt.Fprintf(d.s.Err, prefix+format+"\n", a...) }

// fail prints one ERROR line on stderr and returns code, the exit code of the run.
func (d *deploy) fail(code int, format string, a ...any) int {
	fmt.Fprintf(d.s.Err, errorPrefix+format+"\n", a...)
	return code
}

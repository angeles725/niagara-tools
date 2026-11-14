// Package buildmodule is toolbelt/build.sh: the full automatic chain that builds a Niagara module —
// preflight, gradle on Java 8 (clean + slotomatic + jar), the deployed-baseline drift gate, the verify gate
// and the report-module punch-list. It follows the bash tool it replaces line for line: same rows on the same
// stream, same exit codes, and the same commands run with the same arguments, environment and directory.
//
// One file per responsibility, in chain order: options.go (flags), inputs.go (roots, JDK, niagara_home),
// location.go (WSL mount warning, preflight), profiles.go and gradleconf.go (what gradle builds and with which
// arguments), mirror.go (user.home isolation), baseline.go (pre-build snapshot and restore), gradle.go and
// diagnose.go (the run and its log), drift.go (drift verdict), gate.go (verify gate, report-module). The data
// lives apart: exit.go, messages.go, patterns.go; the usage text is the launcher header (usage.go). This file only composes them.
package buildmodule

import (
	"fmt"
	"os"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

// ToolbeltEnv names the directory of the invoked build.sh, unresolved: the gate, preflight and report-module
// scripts and lib/werror.init.gradle are taken from it, so a toolbelt copy with stubs beside a linked
// build.sh runs its stubs. toolbelt/build.sh sets it; it is not handed on to the commands the chain runs.
const ToolbeltEnv = "BUILD_MODULE_TOOLBELT"

// build is one run of the chain: what the steps before decided, for the steps after.
type build struct {
	s   cli.Stdio
	opt options

	here       string            // the toolbelt directory
	root       string            // <module-root>, absolute
	gradleRoot string            // root or the ancestor holding ./gradlew
	home       string            // niagara_home as given
	java8      string            // JDK 8 path
	plugin     string            // niagara plugin version: flag, env, else detected
	profiles   []string          // selected profiles, in build order
	gradleArgs []string          // arguments after the tasks
	setenv     map[string]string // environment the chain adds for every command it runs
	signingXML string            // the mirror user home's signing profile; "" outside a mirror
	temps      []string          // temporary paths of the run, removed when Main returns
	baseline   map[string]snapshot
}

// steps is the chain; each returns an exit code, or proceed.
var steps = []func(*build) int{
	(*build).resolveInputs,
	(*build).warnSlowMounts,
	(*build).preflight,
	(*build).warnPluginNotInM2,
	(*build).selectProfiles,
	(*build).configureGradle,
	(*build).isolateMirror,
	(*build).snapshotBaseline,
	(*build).runGradle,
	(*build).driftVerdict,
	(*build).gate,
}

// Main implements build.sh; toolbelt/build.sh carries the usage text and the exit codes.
func Main(args []string, s cli.Stdio) int {
	opt, code := parseArgs(args, s)
	if code != proceed {
		return code
	}
	b := &build{s: s, opt: opt, here: s.Env(ToolbeltEnv), plugin: opt.plugin, setenv: map[string]string{}}
	defer b.cleanup()
	stop := b.cleanupOnSignal()
	defer stop()
	for _, step := range steps {
		if code := step(b); code != proceed {
			return code
		}
	}
	return ExitOK
}

// say prints one stdout message (a format of messages.go) and its newline.
func (b *build) say(format string, a ...any) { fmt.Fprintf(b.s.Out, format+"\n", a...) }

// warn prints one stderr message and its newline.
func (b *build) warn(format string, a ...any) { fmt.Fprintf(b.s.Err, format+"\n", a...) }

// jarName is "<MOD>-<profile>", the name of a profile's project and of its jar without ".jar".
func (b *build) jarName(profile string) string { return b.opt.mod + "-" + profile }

// profileDir is the source directory of a profile.
func (b *build) profileDir(profile string) string {
	return b.root + "/" + b.opt.mod + "/" + b.jarName(profile)
}

// builtJar is the jar gradle assembles for a profile.
func (b *build) builtJar(profile string) string {
	return b.profileDir(profile) + "/build/libs/" + b.jarName(profile) + ".jar"
}

// deployedJar is the jar of a profile under <niagara_home>/modules, which gradle's :jar step replaces.
func (b *build) deployedJar(profile string) string {
	return b.home + "/modules/" + b.jarName(profile) + ".jar"
}

// makeTemp creates a temporary directory (`mktemp -d`) or file (`mktemp`) that cleanup removes. code is
// mktemp's status when it fails: the chain cannot go on, and ends with it as under bash's set -e.
func (b *build) makeTemp(directory bool) (path string, code int) {
	var args []string
	if directory {
		args = []string{"-d"}
	}
	if path, code = b.run("mktemp", args...); code == 0 {
		b.temps = append(b.temps, path)
	}
	return path, code
}

// cleanup removes the temporary paths.
func (b *build) cleanup() {
	for _, path := range b.temps {
		_ = os.RemoveAll(path)
	}
}

// Package buildmodule is toolbelt/build.sh: the full automatic chain that builds a Niagara module —
// preflight, gradle on Java 8 (clean + slotomatic + jar), the deployed-baseline drift gate, the verify gate
// and the report-module punch-list. It follows the bash tool it replaces line for line: same rows on the same
// stream, same exit codes, and the same commands run with the same arguments, environment and directory.
//
// One file per responsibility, in chain order: options.go (flags), inputs.go (roots, JDK, niagara_home),
// baselineref.go (--schema-baseline-ref), location.go (WSL mount warning, preflight), profiles.go and gradleconf.go (what gradle builds and with which
// arguments), mirror.go (user.home isolation), baseline.go (pre-build snapshot and restore), replaced.go (the
// mirror jar a passing build replaced: kept, and --restore-mirror; the installation the mirror was made from), gradle.go and
// diagnose.go (the run and its log; signing_key.go the keystore a missing key names), drift.go (drift verdict), gate.go (verify gate, report-module). The data
// lives apart: exit.go, messages.go, patterns.go; the usage text is the launcher header (usage.go). This file only composes them.
package buildmodule

import (
	"fmt"
	"os"
	"sync"
	"sync/atomic"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/strerror"
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
	tempsMu    sync.Mutex        // temps is appended by the chain and read by the signal handler
	// modulesMu serialises what puts modules/ back (restoreBaseline, removeFirstDeploy), what reads it after the run (keepReplaced) and the removal of
	// the snapshot (cleanup) between the chain and the signal handler, which holds it until the run ends.
	modulesMu  sync.Mutex
	gradle     gradleRun // the running gradle, which a signal stops before the restore (gradleproc.go)
	baseline   map[string]snapshot
	deployed   deployRecord    // --target: the recorded deploy of the module, which the drift gate compares with; zero without it
	undeployed map[string]bool // the profiles with no jar in modules/ before gradle ran (a first deploy)
	// unverified is set from the baseline snapshot until the chain passed: modules/ may then hold a jar the
	// gate has not passed, which an interrupt puts back (signal.go).
	unverified atomic.Bool
}

// steps is the chain; each returns an exit code, or proceed.
var steps = []func(*build) int{
	(*build).resolveInputs,
	(*build).applyExpectedSigner,
	(*build).extractBaselines,
	(*build).warnSlowMounts,
	(*build).pickMirrorPlugin,
	(*build).preflight,
	(*build).requireSigningBlock,
	(*build).acornPreflight,
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
	if opt.restore != "" {
		return b.restoreMirror(opt.restore)
	}
	defer b.cleanup()
	stop := b.cleanupOnSignal()
	defer stop()
	for _, step := range steps {
		if code = step(b); code != proceed {
			break
		}
	}
	if code == proceed {
		b.unverified.Store(false)
		code = ExitOK
	}
	b.keepReplaced() // whatever the exit: it looks at what modules/ holds now, before cleanup drops the snapshot
	if code == ExitOK {
		b.sayMirrorSource() // the last line of a passing mirror build
	}
	return code
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

// makeTemp creates a temporary directory or file under $TMPDIR (else /tmp) that cleanup removes, as `mktemp
// [-d]` did. code is 1 when it cannot: the chain cannot go on, and ends with it as under bash's set -e.
func (b *build) makeTemp(directory bool) (path string, code int) {
	dir, kind := b.s.Env("TMPDIR"), "file"
	if dir == "" {
		dir = "/tmp"
	}
	var err error
	if directory {
		kind = "directory"
		path, err = os.MkdirTemp(dir, tempPattern)
	} else {
		var f *os.File
		if f, err = os.CreateTemp(dir, tempPattern); err == nil {
			path = f.Name()
			err = f.Close()
		}
	}
	if err != nil {
		b.warn(msgNoTemp, kind, dir, strerror.Text(err))
		return "", ExitFault
	}
	b.tempsMu.Lock()
	b.temps = append(b.temps, path)
	b.tempsMu.Unlock()
	return path, 0
}

// cleanup removes the temporary paths (the baseline snapshot among them).
func (b *build) cleanup() {
	b.modulesMu.Lock()
	defer b.modulesMu.Unlock()
	b.cleanupHeld()
}

// cleanupHeld is cleanup for a caller that holds modulesMu (the signal handler).
func (b *build) cleanupHeld() {
	b.tempsMu.Lock()
	defer b.tempsMu.Unlock()
	for _, path := range b.temps {
		_ = os.RemoveAll(path)
	}
}

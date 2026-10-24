// Package reportmodule is report-module, the aggregated conformance report of a Niagara N4 module
// (toolbelt/report-module.sh header: members, rows, summary, exits). It runs every toolbelt member that applies
// to each profile artifact under the module root, then the once-per-module members and triage-console, through
// their toolbelt wrappers, and relays their rows prefixed by the artifact name.
package reportmodule

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

// ToolbeltEnv names the toolbelt directory whose members run. toolbelt/report-module.sh sets it to its own
// directory, unresolved, so a stubbed toolbelt copy (tests/helpers/stub-toolbelt.bash) runs its stubs.
const ToolbeltEnv = "REPORT_MODULE_TOOLBELT"

// Main runs report-module; the members inherit the process environment.
func Main(args []string, s cli.Stdio) int { return mainEnv(args, s, os.Environ()) }

// mainEnv is Main with the members' environment given (ToolbeltEnv is taken out of it: the bash members never
// saw it); tests pass a minimal one so no host variable leaks in.
func mainEnv(args []string, s cli.Stdio, environ []string) int {
	o, ok := parseArgs(args)
	if !ok {
		fmt.Fprint(s.Err, usage)
		return ExitUsage
	}
	if !isDir(o.root) {
		fmt.Fprintf(s.Err, "report-module: not a directory: %s\n", o.root)
		return ExitEnv
	}
	switch {
	case o.wiringMap != "":
		if !isFile(o.wiringMap) {
			fmt.Fprintf(s.Err, "report-module: --wiring-map file not found: %s\n", o.wiringMap)
			return ExitEnv
		}
	case isFile(o.root + "/docs/wiring-map.md"):
		o.wiringMap = o.root + "/docs/wiring-map.md"
	case isFile(o.root + "/../docs/wiring-map.md"):
		o.wiringMap = o.root + "/../docs/wiring-map.md"
	}
	artifacts, err := discover(o.root)
	if err != nil {
		fmt.Fprintf(s.Err, "find: '%s': %s\n", o.root, errText(err))
	}
	if len(artifacts) == 0 {
		fmt.Fprintf(s.Err, "report-module: no artifacts found under %s\n", o.root)
		return ExitEnv
	}
	env := make([]string, 0, len(environ))
	for _, kv := range environ {
		if !strings.HasPrefix(kv, ToolbeltEnv+"=") {
			env = append(env, kv)
		}
	}
	r := &report{out: s.Out, errw: s.Err, o: o, getenv: s.Env, n: map[string]int{},
		x: runner{tb: toolbelt(s), env: env}}
	for _, a := range artifacts {
		for _, m := range perArtifact {
			r.step(a, m)
		}
	}
	mod := artifact{"(module)", o.root}
	for _, m := range perModule {
		r.step(mod, m)
	}
	r.triageConsole()
	return r.summary(len(artifacts))
}

// discover lists the profile artifacts like `find -H <root> -mindepth 1 -maxdepth 1 -type d | sort` in the C
// locale: every entry of root (a symlinked root followed) that is a directory, not a symlink and not a
// dot-directory, in byte order, spelled <root>/<name> (no separator added when root already ends in one).
func discover(root string) ([]artifact, error) {
	entries, err := os.ReadDir(root)
	var out []artifact
	for _, e := range entries {
		if e.IsDir() && !strings.HasPrefix(e.Name(), ".") { // a dot-directory (.git, .gradle) is no artifact
			dir := root + "/" + e.Name()
			if strings.HasSuffix(root, "/") {
				dir = root + e.Name()
			}
			out = append(out, artifact{e.Name(), dir})
		}
	}
	return out, err
}

// errText is the strerror text of a file error ("Permission denied").
func errText(err error) string {
	if pe, ok := err.(*os.PathError); ok {
		err = pe.Err
	}
	msg := err.Error()
	return strings.ToUpper(msg[:1]) + msg[1:]
}

// toolbelt is the directory of the member wrappers: ToolbeltEnv, else build-n4-module-kit/toolbelt beside the
// go/ module this binary was built in (go/bin/<tool>).
func toolbelt(s cli.Stdio) string {
	if d := s.Env(ToolbeltEnv); d != "" {
		return d
	}
	exe, err := os.Executable()
	if err == nil {
		exe, err = filepath.EvalSymlinks(exe)
	}
	if err != nil {
		return ""
	}
	return filepath.Join(filepath.Dir(exe), "..", "..", "build-n4-module-kit", "toolbelt")
}

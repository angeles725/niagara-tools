// Package commissioningverify is commissioning-verify, the §6.b commissioning punch-list auditor
// (toolbelt/commissioning-verify.sh header: checks, rows, exits). It runs the per -rt/src source lints through
// their toolbelt wrappers, folds the bog-audit station checks in process, reads the values-owed table and
// prints the manual-only footer and the summary.
package commissioningverify

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

// Exit codes of commissioning-verify (unchanged from the bash tool).
const (
	ExitClean = 0 // no FAIL row
	ExitFail  = 1 // any FAIL row
	ExitEnv   = 3 // usage or environment error
)

// ToolbeltEnv names the toolbelt directory whose lint wrappers run. toolbelt/commissioning-verify.sh sets it to
// its own directory, unresolved, so a stubbed toolbelt copy (tests/helpers/stub-toolbelt.bash) runs its stubs.
const ToolbeltEnv = "COMMISSIONING_VERIFY_TOOLBELT"

const usage = "usage: commissioning-verify.sh <module-root> [--bog <config.bog>] [--module <MOD>] [--values-owed <file>] [--wiring-map <file>] [--strict]\n"

type options struct {
	root, bog, mod, owed, wmap string
	strict                     bool
}

// parseArgs reads the arguments like the bash loop: a value flag needs its value, `--` ends the scan (what
// follows is ignored), any other dash word or a second positional is a usage error.
func parseArgs(args []string) (o options, ok bool) {
	values := map[string]*string{"--bog": &o.bog, "--module": &o.mod, "--values-owed": &o.owed, "--wiring-map": &o.wmap}
	for i := 0; i < len(args); i++ {
		a := args[i]
		if v := values[a]; v != nil {
			if i+1 == len(args) {
				return o, false
			}
			i++
			*v = args[i]
			continue
		}
		switch {
		case a == "--strict":
			o.strict = true
		case a == "--":
			return o, true
		case a != "" && a[0] == '-', o.root != "":
			return o, false
		default:
			o.root = a
		}
	}
	return o, true
}

// Main runs commissioning-verify: exit 0 clean, 1 on a FAIL row, 3 on usage or an environment error. The source
// lints inherit the process environment (plus LC_ALL=C).
func Main(args []string, s cli.Stdio) int { return mainEnv(args, s, os.Environ()) }

// mainEnv is Main with the lints' environment given: tests pass a minimal one so no host variable leaks in.
func mainEnv(args []string, s cli.Stdio, environ []string) int {
	fail := func(format string, a ...any) int {
		fmt.Fprintf(s.Err, "commissioning-verify: "+format+"\n", a...)
		return ExitEnv
	}
	o, ok := parseArgs(args)
	if !ok || o.root == "" {
		fmt.Fprint(s.Err, usage)
		return ExitEnv
	}
	if !isDir(o.root) {
		return fail("not a directory: %s", o.root)
	}
	for _, f := range []struct {
		path      *string
		def, what string
	}{
		{&o.owed, "/docs/values-owed.md", "values-owed table"}, {&o.wmap, "/docs/wiring-map.md", "wiring map"},
	} {
		if *f.path != "" && !isFile(*f.path) {
			return fail("%s not found: %s", f.what, *f.path)
		}
		if *f.path == "" && isFile(o.root+f.def) {
			*f.path = o.root + f.def
		}
	}
	var owedText []byte // an unreadable table is env, never a PASS (the bash tool's awk failed and it PASSed)
	if o.owed != "" {
		var err error
		if owedText, err = os.ReadFile(o.owed); err != nil {
			return fail("values-owed table not readable: %s", o.owed)
		}
	}
	if o.bog != "" && o.mod == "" {
		fail("--bog requires --module")
		fmt.Fprint(s.Err, usage)
		return ExitEnv
	}
	srcs := sourceDirs(o.root)
	if len(srcs) == 0 {
		return fail("no -rt/src or src/ found under %s", o.root)
	}
	r := &report{out: s.Out, n: map[string]int{}, env: environ}
	tb := toolbelt(s)
	for _, src := range srcs {
		r.sourceChecks(tb, src, o.strict, o.wmap)
	}
	r.emit(pass, "wiring-map", "run generate-wiring-map.sh <facade-src-dir> to scaffold docs/wiring-map.md before commissioning")
	if o.bog == "" {
		for _, c := range []string{"proxy-link-safety", "station-logic", "numeric-to-bool-direct"} {
			r.emit(skip, c, "no --bog provided; pass --bog <config.bog> --module <MOD> to audit live station")
		}
	} else if !isFile(o.bog) {
		return fail("bog not found: %s", o.bog)
	} else {
		r.stationChecks(o)
	}
	r.valuesOwed(o.owed, string(owedText))
	for _, m := range footer {
		r.emit(manual, m[0], m[1])
	}
	verdict, code := "CLEAN", ExitClean
	if r.n[failed] > 0 {
		verdict, code = "ISSUES", ExitFail
	}
	fmt.Fprintf(s.Out, "commissioning-verify: %d PASS · %d FAIL · %d WARN · %d SKIP · %d MANUAL  ->  %s\n",
		r.n[pass], r.n[failed], r.n[warn], r.n[skip], r.n[manual], verdict)
	return code
}

// toolbelt is the directory of the lint wrappers: ToolbeltEnv, else build-n4-module-kit/toolbelt beside the go/
// module this binary was built in (go/bin/<tool>).
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

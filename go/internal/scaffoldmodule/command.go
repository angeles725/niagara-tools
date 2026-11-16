// Package scaffoldmodule implements scaffold-module: it copies the bundled fixtures/<Type> tree of the kit
// to <out-dir>/<ModuleName>, renaming the fixture's names in paths and text, then adds the station-test
// overlay (logic, dashboard) and the client version ledger. It never invokes version control.
//
// The kit is found from the toolbelt directory the launcher exports (ToolbeltEnv), exactly as the bash tool
// resolved it from its own location: <toolbelt>/../fixtures and <toolbelt>/../templates.
//
// One responsibility per file: types.go the type table and defaults (data), messages.go every printed
// text, args.go the command line, name.go the ModuleName rule, values.go the option values (literal
// text), subst.go the substitution rules, copytree.go the tree copy, overlay.go the station-test overlay
// and the ledger, concern.go the add-concern passthrough, command.go the order of the steps and the exits.
package scaffoldmodule

import (
	"fmt"
	"os"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

// ToolbeltEnv names the variable the launcher sets to the toolbelt directory it was invoked from.
const ToolbeltEnv = "SCAFFOLD_MODULE_TOOLBELT"

// Exit codes of scaffold-module (unchanged from the bash tool).
const (
	ExitOK    = 0
	ExitUsage = 2 // usage, invalid ModuleName, unknown --type, a refused option value
	ExitEnv   = 3 // fixture or template missing, out-dir not creatable, destination exists, a failed write
)

// failure is a refused run: the text for stderr and the exit code.
type failure struct {
	text string
	code int
}

func failf(code int, format string, a ...any) *failure {
	return &failure{fmt.Sprintf(format, a...), code}
}

// Main implements `scaffold-module <ModuleName> <out-dir> [options]` and the `add-concern` passthrough.
func Main(args []string, s cli.Stdio) int {
	toolbelt := s.Env(ToolbeltEnv)
	if len(args) > 0 && args[0] == concernCommand {
		return addConcern(toolbelt, args, s)
	}
	dest, f := scaffold(toolbelt, args)
	if f != nil {
		fmt.Fprint(s.Err, f.text)
		return f.code
	}
	fmt.Fprintf(s.Out, emittedLines, args[0], dest)
	return ExitOK
}

// scaffold runs the steps in the bash tool's order, so the first fault a run meets is the same one:
// arguments, --type, ModuleName, option values, fixture, out-dir, destination, copy, overlay, ledger.
func scaffold(toolbelt string, args []string) (string, *failure) {
	o, f := parseArgs(args)
	if f != nil {
		return "", f
	}
	typ, ok := lookupType(o.typ)
	if !ok {
		return "", failf(ExitUsage, unknownType+usageText, o.typ)
	}
	if f := checkName(o.module); f != nil {
		return "", f
	}
	if f := checkValues(o); f != nil {
		return "", f
	}
	n := newNames(o, typ)
	if toolbelt == "" {
		return "", failf(ExitEnv, toolbeltUnset)
	}
	fixture := toolbelt + "/../fixtures/" + typ.fixture
	if !isDir(fixture + "/gradle/wrapper") {
		return "", failf(ExitEnv, fixtureMissing, fixture)
	}
	if err := os.MkdirAll(o.outDir, 0o777); err != nil && !isDir(o.outDir) {
		return "", failf(ExitEnv, outDirFault, o.outDir)
	}
	dest := o.outDir + "/" + o.module
	if _, err := os.Stat(dest); err == nil {
		return "", failf(ExitEnv, destExists, dest)
	}
	if err := copyTree(fixture, dest, n.fixturePathRules(), n.fixtureTextRules(), verbatimFixtureFile); err != nil {
		return "", failf(ExitEnv, writeFault, err)
	}
	if f := stationTestOverlay(toolbelt, dest, n, typ); f != nil {
		return "", f
	}
	if f := versionLedger(toolbelt, dest); f != nil {
		return "", f
	}
	return dest, nil
}

func isDir(path string) bool {
	st, err := os.Stat(path)
	return err == nil && st.IsDir()
}

func isFile(path string) bool {
	st, err := os.Stat(path)
	return err == nil && st.Mode().IsRegular()
}

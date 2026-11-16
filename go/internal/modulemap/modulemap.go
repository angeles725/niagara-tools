// Package modulemap is the module-map tool behind toolbelt/module-map.sh (modularity-structure Δ1 and Δ4): it
// reads a module root's MODULE-MAP.md with the lint-module-map parser and
//
//	add-concern <Name> --root <module-root> [--part rt|ux|wb|se] [--adapter]
//	    creates the concern folder under the part's role folder, a pure <Name>Logic class and its
//	    <Name>LogicTest (an rt <Name>Adapter stub under adapter/<concern>/ with --adapter) and inserts the
//	    concern's MODULE-MAP row after the table's last row; refuses an existing concern or folder
//	where <file|concern> --root <module-root>
//	    prints the concern that lists the file (or the named concern): its part, files, tests and "change here when"
//	tests <concern> --root <module-root>
//	    runs the concern's pure JUnit tests through run-pure-test.sh (found in $MODULE_MAP_TOOLBELT) and names
//	    each station test (a javax.baja import or fully qualified name in code) and JS spec it skips
//
// Exit: 0 ok · 1 refused (add-concern), not found (where, tests) or a test failed (tests) · 2 usage · 3 env
// (no or invalid map, no such part, a write or the runner failing to start or reporting an env error).
package modulemap

import (
	"errors"
	"fmt"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	mmrule "github.com/angeles725/niagara-tools/go/internal/lint/rules/modulemap"
)

// exitErr is an error carrying the exit code it maps to.
type exitErr struct {
	code int
	msg  string
}

func (e exitErr) Error() string { return e.msg }

func fail(code int, format string, a ...any) error { return exitErr{code, fmt.Sprintf(format, a...)} }

// Main is the module-map command.
func Main(args []string, s cli.Stdio) int {
	o, err := parse(args)
	if err == nil {
		switch o.cmd {
		case "add-concern":
			err = addConcern(o, s)
		case "where":
			err = where(o, s)
		default:
			err = tests(o, s)
		}
	}
	if err == nil {
		return 0
	}
	fmt.Fprintln(s.Err, err.Error())
	var e exitErr
	if errors.As(err, &e) {
		return e.code
	}
	return 3
}

func readMap(root string) (mmrule.Map, error) {
	m, err := mmrule.ReadMap(root)
	if err != nil {
		return m, fail(3, "module-map: %v", err)
	}
	return m, nil
}

func findRow(m mmrule.Map, concern string) (mmrule.Row, bool) {
	for _, r := range m.Rows {
		if strings.EqualFold(r.Concern, concern) {
			return r, true
		}
	}
	return mmrule.Row{}, false
}

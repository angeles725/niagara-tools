// Package kitstate is the one reader command for the kit's generated state: the BUILD-STATE.md kit
// ledger and the retros/INDEX.md promotion registry. Neither file is committed (two PRs that each add
// a ledger.d/ fragment or a retro used to conflict on them); `kit-state` prints what the committed
// sources generate now, or with --write materializes the files in the kit. It renders nothing itself:
// the generators are go/internal/buildstate and go/internal/retroindex. [ev: retro ledger-generated-not-committed]
package kitstate

import (
	"fmt"
	"io"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/buildstate"
	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/gen"
	"github.com/angeles725/niagara-tools/go/internal/retroindex"
)

// DefaultRootEnv names the kit root used when no <kit-root> argument is given; the
// toolbelt/kit-state.sh wrapper exports it as its own ../ directory.
const DefaultRootEnv = "KIT_STATE_DEFAULT_ROOT"

const (
	tool  = "kit-state"
	usage = "usage: kit-state.sh build-state|retro-index [<kit-root>]\n" +
		"       kit-state.sh --write [build-state|retro-index] [<kit-root>]\n"
)

// view is one generated file: how it is generated and where --write puts it.
type view struct {
	generate func(root string, diag io.Writer) (string, int)
	output   func(tool, root, content string) gen.File
}

var views = map[string]view{
	"build-state": {buildstate.Generate, buildstate.Output},
	"retro-index": {retroindex.Generate, retroindex.Output},
}

// order is the --write order with no view named (both files).
var order = []string{"build-state", "retro-index"}

// Main implements `kit-state build-state|retro-index [<kit-root>]` (print the generated file to stdout)
// and `kit-state --write [build-state|retro-index] [<kit-root>]` (write it, or both, into the kit and
// print each path written). Every selected file is generated before any is written, so a violation
// writes nothing. Exits: 0 ok · 1 a fragment / source / retro-header violation · 3 usage or env.
func Main(args []string, s cli.Stdio) int {
	write, names, root, ok := parse(args)
	if !ok {
		fmt.Fprint(s.Err, usage)
		return gen.ExitEnv
	}
	if root == "" {
		if root = s.Env(DefaultRootEnv); root == "" {
			fmt.Fprintf(s.Err, "kit-state: no <kit-root> given and %s is unset (run toolbelt/kit-state.sh)\n", DefaultRootEnv)
			return gen.ExitEnv
		}
	}
	contents := make([]string, len(names))
	for i, n := range names {
		c, code := views[n].generate(root, s.Err)
		if code != gen.ExitOK {
			return code
		}
		contents[i] = c
	}
	for i, n := range names {
		if !write {
			fmt.Fprint(s.Out, contents[i])
			continue
		}
		if code := views[n].output(tool, root, contents[i]).Write(s); code != gen.ExitOK {
			return code
		}
	}
	return gen.ExitOK
}

// parse reads the command line: --write, at most one view name, at most one kit root. Printing needs
// exactly one view; --write with none writes both.
func parse(args []string) (write bool, names []string, root string, ok bool) {
	for _, a := range args {
		_, isView := views[a]
		switch {
		case a == "--write":
			write = true
		case isView && len(names) == 0:
			names = []string{a}
		case isView, strings.HasPrefix(a, "-"), root != "":
			return false, nil, "", false
		default:
			root = a
		}
	}
	if len(names) == 0 {
		if !write {
			return false, nil, "", false
		}
		names = order
	}
	return write, names, root, true
}

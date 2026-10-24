// Package kitstate is the one reader command for the kit's generated state: the BUILD-STATE.md kit
// ledger, the retros/INDEX.md promotion registry and the toolbelt/INDEX.md lint index. None of them is
// committed (two PRs that each add a ledger.d/ fragment, a retro or a lint used to conflict on them); `kit-state` prints what the committed
// sources generate now, or with --write materializes the files in the kit. It renders nothing itself:
// the generators are go/internal/buildstate, go/internal/retroindex and go/internal/lintindex. [ev: retro ledger-generated-not-committed]
package kitstate

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/buildstate"
	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/gen"
	"github.com/angeles725/niagara-tools/go/internal/lintindex"
	"github.com/angeles725/niagara-tools/go/internal/retroindex"
)

// DefaultRootEnv names the kit root used when no <kit-root> argument is given; the
// toolbelt/kit-state.sh wrapper exports it as its own ../ directory.
const DefaultRootEnv = "KIT_STATE_DEFAULT_ROOT"

const (
	tool  = "kit-state"
	usage = "usage: kit-state.sh build-state|retro-index|lint-index [<kit-root>]\n" +
		"       kit-state.sh --write [build-state|retro-index|lint-index] [<kit-root>]\n"
)

// view is one generated file: how it is generated and where --write puts it.
type view struct {
	generate func(root string, diag io.Writer) (string, int)
	output   func(tool, root, content string) gen.File
}

var views = map[string]view{
	"build-state": {buildstate.Generate, buildstate.Output},
	"retro-index": {retroindex.Generate, retroindex.Output},
	"lint-index":  {lintindex.Generate, lintindex.Output},
}

// order is the --write order with no view named (every generated file).
var order = []string{"build-state", "retro-index", "lint-index"}

// Main implements `kit-state build-state|retro-index|lint-index [<kit-root>]` (print the generated file)
// and `kit-state --write [build-state|retro-index|lint-index] [<kit-root>]` (write it, or all, into the kit and
// print each path written). Every selected file is generated before any is written, so a violation
// writes nothing, and a failed write removes the whole set (writeAll). Exits: 0 ok · 1 a fragment / source / retro-header violation · 3 usage or env.
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
	if !write {
		for _, c := range contents {
			fmt.Fprint(s.Out, c)
		}
		return gen.ExitOK
	}
	return writeAll(names, contents, root, s)
}

// writeAll writes each generated file in turn. When one write fails, every file of the set is removed
// (best effort; the ones written before it, and the failed target, which may hold an older version),
// so a reader never sees a fresh BUILD-STATE.md beside a stale or missing index. Each removal is reported.
func writeAll(names, contents []string, root string, s cli.Stdio) int {
	for i, n := range names {
		if code := views[n].output(tool, root, contents[i]).Write(s); code != gen.ExitOK {
			// Mutation: KS-pair -- keeping the files already written leaves a mixed pair
			for _, m := range names {
				path := views[m].output(tool, root, "").Path
				if err := os.Remove(path); err == nil {
					fmt.Fprintf(s.Err, "kit-state: removed %s (the set could not be written whole)\n", path)
				}
			}
			return code
		}
	}
	return gen.ExitOK
}

// parse reads the command line: --write, at most one view name, at most one kit root. Printing needs
// exactly one view; --write with none writes all of them.
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

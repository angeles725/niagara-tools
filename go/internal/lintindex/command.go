package lintindex

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/gen"
)

// Exit codes of gen-lint-index (unchanged from the bash tool).
const (
	ExitOK    = gen.ExitOK    // index written, or fresh under --check
	ExitStale = gen.ExitStale // stale or missing index under --check, or a header-contract violation
	ExitEnv   = gen.ExitEnv   // usage or environment error, or a lint header that cannot be read; nothing written
)

// DefaultRootEnv names the kit root used when no <kit-root> argument is given; the
// toolbelt/gen-lint-index.sh wrapper exports it as its own ../ directory.
const DefaultRootEnv = "GEN_LINT_INDEX_DEFAULT_ROOT"

const tool = "gen-lint-index"

// Problem is a lint whose header broke the contract (IO false) or could not be read (IO true).
type Problem struct {
	Name, Reason string
	IO           bool
}

// Scan reads every regular toolbelt/lint-*.sh under tb (symlinks followed, C byte order) and
// returns their rows, Auto set from tb/report-module.sh, plus the per-lint problems. A
// report-module.sh that cannot be read marks every lint Auto no and is reported in warn.
func Scan(tb string) (rows []Row, problems []Problem, warn string, err error) {
	paths, err := filepath.Glob(filepath.Join(tb, "lint-*.sh"))
	if err != nil {
		return nil, nil, "", err
	}
	report := ""
	if rm := filepath.Join(tb, "report-module.sh"); gen.IsRegular(rm) {
		b, rerr := os.ReadFile(rm)
		if rerr != nil {
			warn = fmt.Sprintf("cannot read %s: %v", rm, rerr)
		}
		report = string(b)
	}
	for _, p := range paths { // filepath.Glob sorts by byte order, as LC_ALL=C globbing did
		if !gen.IsRegular(p) {
			continue
		}
		name := filepath.Base(p)
		b, rerr := os.ReadFile(p)
		if rerr != nil {
			problems = append(problems, Problem{name, fmt.Sprintf("cannot read the header: %v", rerr), true})
			continue
		}
		row, problem := Extract(name, string(b))
		if problem != "" {
			problems = append(problems, Problem{name, problem, false})
			continue
		}
		row.Auto = report != "" && InvokedByReport(report, name)
		rows = append(rows, row)
	}
	return rows, problems, warn, nil
}

// Main implements `gen-lint-index [--check] [<kit-root>]`: it writes <kit-root>/toolbelt/INDEX.md,
// or with --check compares it with what the lint headers generate. Any problem means nothing is
// written.
func Main(args []string, s cli.Stdio) int {
	check, root, ok := gen.ParseArgs(tool, DefaultRootEnv, args, s)
	if !ok {
		return ExitEnv
	}
	return generate(root, check, s)
}

// generate renders <root>/toolbelt/INDEX.md and writes it, or compares it under check.
func generate(root string, check bool, s cli.Stdio) int {
	tb, ok := gen.SubDir(tool, root, "toolbelt", s)
	if !ok {
		return ExitEnv
	}
	out := tb + "/INDEX.md"

	rows, problems, warn, err := Scan(tb)
	if err != nil {
		fmt.Fprintf(s.Err, "gen-lint-index: cannot list %s: %v; index not written\n", tb, err)
		return ExitEnv
	}
	if warn != "" {
		fmt.Fprintf(s.Err, "gen-lint-index: %s\n", warn)
	}
	unreadable := false
	for _, p := range problems {
		fmt.Fprintf(s.Err, "gen-lint-index: %s: %s\n", p.Name, p.Reason)
		unreadable = unreadable || p.IO
	}
	// Mutation: GLI-awkfail -- treating an unreadable header like a missing one writes nothing but exits 1, not 3
	if unreadable {
		fmt.Fprint(s.Err, "gen-lint-index: a lint header could not be read; index not written\n")
		return ExitEnv
	}
	if len(problems) > 0 {
		fmt.Fprint(s.Err, "gen-lint-index: header-contract violation; index not written\n")
		return ExitStale
	}
	// Mutation: GLI-stale -- the shared compare in gen.File; dropping it makes --check vacuously clean
	return gen.File{Tool: tool, Path: out, Content: Render(rows) + "\n", TrimNewlines: true}.Run(check, s)
}

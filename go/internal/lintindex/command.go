package lintindex

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/gen"
	"github.com/angeles725/niagara-tools/go/internal/reportmodule"
)

// Exit codes of gen-lint-index (unchanged from the bash tool).
const (
	ExitOK    = gen.ExitOK    // index written, or the lint headers are valid under --check
	ExitStale = gen.ExitStale // a header-contract violation; nothing written
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
// returns their rows, Auto set when tb has a report-module.sh, plus the per-lint problems. Auto is
// yes for a lint of the Go report's member table (reportmodule.Scripts: report-module.sh is a
// wrapper of go/internal/reportmodule) or one a bash report-module.sh text invokes (InvokedByReport).
// A report-module.sh that cannot be read marks every lint Auto no and is reported in warn.
func Scan(tb string) (rows []Row, problems []Problem, warn string, err error) {
	paths, err := filepath.Glob(filepath.Join(tb, "lint-*.sh"))
	if err != nil {
		return nil, nil, "", err
	}
	members := map[string]bool{}
	for _, m := range reportmodule.Scripts() {
		members[m] = true
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
		// Mutation: GLI-members -- reading only the report-module.sh text marks every member Auto no
		row.Auto = report != "" && (members[name] || InvokedByReport(report, name))
		rows = append(rows, row)
	}
	return rows, problems, warn, nil
}

// Main implements `gen-lint-index [--check] [<kit-root>]`: it writes <kit-root>/toolbelt/INDEX.md, or
// with --check only validates the lint headers (the index is not committed, so there is nothing to
// compare). Any problem means nothing is written.
func Main(args []string, s cli.Stdio) int {
	check, root, ok := gen.ParseArgs(tool, DefaultRootEnv, args, s)
	if !ok {
		return ExitEnv
	}
	content, code := Generate(root, s.Err)
	if code != ExitOK {
		return code
	}
	if check {
		fmt.Fprintf(s.Out, "gen-lint-index: the lint headers under %s/toolbelt are valid\n", root)
		return ExitOK
	}
	return Output(tool, root, content).Write(s)
}

// Output is the generated toolbelt/INDEX.md of root as a file to write; tool prefixes its messages.
func Output(tool, root, content string) gen.File {
	return gen.File{Tool: tool, Path: root + "/toolbelt/INDEX.md", Content: content}
}

// Generate returns the toolbelt/INDEX.md that the toolbelt/lint-*.sh headers of root generate. A
// header-contract violation is reported to diag and returned as ExitStale; a missing toolbelt/ or an
// unreadable header is ExitEnv.
func Generate(root string, diag io.Writer) (string, int) {
	tb, ok := gen.SubDir(tool, root, "toolbelt", cli.Stdio{Out: io.Discard, Err: diag})
	if !ok {
		return "", ExitEnv
	}
	rows, problems, warn, err := Scan(tb)
	if err != nil {
		fmt.Fprintf(diag, "gen-lint-index: cannot list %s: %v; index not generated\n", tb, err)
		return "", ExitEnv
	}
	if warn != "" {
		fmt.Fprintf(diag, "gen-lint-index: %s\n", warn)
	}
	unreadable := false
	for _, p := range problems {
		fmt.Fprintf(diag, "gen-lint-index: %s: %s\n", p.Name, p.Reason)
		unreadable = unreadable || p.IO
	}
	// Mutation: GLI-awkfail -- treating an unreadable header like a missing one writes nothing but exits 1, not 3
	if unreadable {
		fmt.Fprint(diag, "gen-lint-index: a lint header could not be read; index not generated\n")
		return "", ExitEnv
	}
	if len(problems) > 0 {
		fmt.Fprint(diag, "gen-lint-index: header-contract violation; index not generated\n")
		return "", ExitStale
	}
	return Render(rows) + "\n", ExitOK
}

package lintindex

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

// Exit codes of gen-lint-index (unchanged from the bash tool).
const (
	ExitOK    = 0 // index written, or fresh under --check
	ExitStale = 1 // stale or missing index under --check, or a header-contract violation
	ExitEnv   = 3 // usage or environment error, or a lint header that cannot be read; nothing written
)

// DefaultRootEnv names the kit root used when no <kit-root> argument is given; the
// toolbelt/gen-lint-index.sh wrapper exports it as its own ../ directory.
const DefaultRootEnv = "GEN_LINT_INDEX_DEFAULT_ROOT"

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
	if rm := filepath.Join(tb, "report-module.sh"); isRegular(rm) {
		b, rerr := os.ReadFile(rm)
		if rerr != nil {
			warn = fmt.Sprintf("cannot read %s: %v", rm, rerr)
		}
		report = string(b)
	}
	for _, p := range paths { // filepath.Glob sorts by byte order, as LC_ALL=C globbing did
		if !isRegular(p) {
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

func isRegular(path string) bool {
	st, err := os.Stat(path)
	return err == nil && st.Mode().IsRegular()
}

// Main implements `gen-lint-index [--check] [<kit-root>]`: it writes <kit-root>/toolbelt/INDEX.md,
// or with --check compares it with what the lint headers generate. Any problem means nothing is
// written.
func Main(args []string, s cli.Stdio) int {
	check, root := false, ""
	for _, a := range args {
		switch {
		case a == "--check":
			check = true
		case strings.HasPrefix(a, "-") || root != "":
			fmt.Fprint(s.Err, "usage: gen-lint-index.sh [--check] [<kit-root>]\n")
			return ExitEnv
		default:
			root = a // an empty argument leaves the default in place, as the bash tool did
		}
	}
	if root == "" {
		if root = s.Env(DefaultRootEnv); root == "" {
			fmt.Fprintf(s.Err, "gen-lint-index: no <kit-root> given and %s is unset (run toolbelt/gen-lint-index.sh)\n", DefaultRootEnv)
			return ExitEnv
		}
	}
	tb := root + "/toolbelt"
	if st, err := os.Stat(tb); err != nil || !st.IsDir() {
		fmt.Fprintf(s.Err, "gen-lint-index: no toolbelt/ under %s\n", root)
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
	content := Render(rows)

	if check {
		if !isRegular(out) {
			fmt.Fprintf(s.Err, "gen-lint-index: %s is missing; run gen-lint-index.sh\n", out)
			return ExitStale
		}
		committed, err := os.ReadFile(out)
		// Mutation: GLI-stale -- dropping the generated-vs-committed compare makes --check vacuously clean
		// Trailing newlines are ignored on both sides, as the bash $(...) comparison did.
		if err != nil || strings.TrimRight(string(committed), "\n") != content {
			fmt.Fprintf(s.Err, "gen-lint-index: %s is stale; run gen-lint-index.sh and commit the result\n", out)
			showDiff(content+"\n", out, s)
			return ExitStale
		}
		fmt.Fprintf(s.Out, "gen-lint-index: %s is fresh\n", out)
		return ExitOK
	}

	if err := os.WriteFile(out, []byte(content+"\n"), 0o666); err != nil {
		fmt.Fprintf(s.Err, "gen-lint-index: cannot write %s: %v\n", out, err)
		return ExitEnv
	}
	fmt.Fprintf(s.Out, "gen-lint-index: wrote %s\n", out)
	return ExitOK
}

// showDiff writes `diff - <path>` (generated vs committed, POSIX normal format) to s.Err; it is
// diagnostic only, so a missing diff binary prints nothing.
func showDiff(generated, path string, s cli.Stdio) {
	cmd := exec.Command("diff", "-", path)
	cmd.Stdin = strings.NewReader(generated)
	cmd.Stdout, cmd.Stderr = s.Err, s.Err
	_ = cmd.Run()
}

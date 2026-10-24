// Package lint is the engine of the kit's Java lints: one Rule interface, one Finding type and row
// formatter, one argument parser (the parse modes of toolbelt/lib/awk-lint.sh), one fail-closed file
// walker (toolbelt/lib/scan-files.sh) and one runner that reads and scans each file once for every
// selected rule. A rule lives in its own package under rules/ and holds only its check; see go/README.md.
package lint

import (
	"errors"
	"fmt"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
)

// Severity is the first column of a row. Its zero value is Unset, so a Finding that leaves it out
// takes its rule's severity.
type Severity int

const (
	Unset    Severity = iota // a Finding's "use Rule.Severity()"; never printed (the runner fills it)
	Warn                     // a warning: exit 0, or 1 under --strict
	Fail                     // a failed check: exit 1 with or without --strict
	Pass                     // a check that held (rows print PASS); never affects the exit code
	Advisory                 // a note that is not a failure (rows print ADVISORY); never affects the exit code
	Info                     // information only (rows print INFO); never affects the exit code
)

func (s Severity) String() string {
	switch s {
	case Fail:
		return "FAIL"
	case Pass:
		return "PASS"
	case Advisory:
		return "ADVISORY"
	case Info:
		return "INFO"
	}
	return "WARN"
}

// Finding is one reported row. A rule sets Msg, Line (1-based; 0 for a finding about the whole file),
// Check only when the row names a sub-check of the rule, and Severity only when this finding differs
// from the rule's Severity(). The runner fills Rule, an Unset Severity and, for a per-file Rule, Path
// (a TreeRule sets Path itself).
type Finding struct {
	Rule     string
	Check    string // the sub-check shown in the check column; empty shows Rule
	Severity Severity
	Path     string
	Line     int
	Msg      string
}

// Row is the line the bash lints print: "WARN  <check>  <path>:<line>  <msg>", or "WARN  <check>  <path>: <msg>"
// for a file-level finding (Line 0). The check column is Check, or Rule when Check is empty.
func (f Finding) Row() string { return f.row(false, false) }

// row is Row, with a file-level finding printed "<path>  <msg>" when bare is set (Spec.BarePathRow) and the
// check column before the severity when checkFirst is set (Spec.CheckFirst).
func (f Finding) row(bare, checkFirst bool) string {
	check := f.Check
	if check == "" {
		check = f.Rule
	}
	first, second := f.Severity.String(), check
	if checkFirst {
		first, second = check, first
	}
	if f.Line == 0 && bare {
		return fmt.Sprintf("%s  %s  %s  %s", first, second, f.Path, f.Msg)
	}
	if f.Line == 0 {
		return fmt.Sprintf("%s  %s  %s: %s", first, second, f.Path, f.Msg)
	}
	return fmt.Sprintf("%s  %s  %s:%d  %s", first, second, f.Path, f.Line, f.Msg)
}

// Rule is one lint check over one Java file. Check returns its findings in report order; a non-nil
// error means the rule could not analyze the file (e.g. unbalanced braces) and fails the scan closed
// like an unreadable file: the runner reports it on stderr, prints the findings returned with it,
// exits 3 and goes on with the other files. A clean file is (nil, nil), never an error.
type Rule interface {
	ID() string         // the lint name printed in every row and accepted by n4lint, e.g. lint-null-context-write
	Severity() Severity // the severity of every finding that leaves its own Unset
	Check(f *javasrc.File) ([]Finding, error)
}

// TreeRule is a lint check over every file it selects under one root at once, for a lint whose report
// is ordered by check rather than by file. CheckTree gets the files in walk order (each read once and
// shared with the per-file rules) and sets Path on each finding. A non-nil error fails the scan closed:
// the runner prints the findings returned with it, then "<lint>: <error>" on stderr (the rule writes
// the whole text, e.g. "cannot scan <path>: <reason>"), and exits 3. A TreeRule whose selected files hold
// nothing it checks (lint-duplicate-helper: no -rt/-ux/-wb src/ file) returns ErrEmptyRoot, as a RootRule
// does: the Spec.EmptyError row instead of its findings, the rule stops and the exit is 3.
type TreeRule interface {
	ID() string
	Severity() Severity
	CheckTree(files []*javasrc.File) ([]Finding, error)
}

// RootRule is a lint check that reads its root itself instead of the files the shared walk lists: a
// structure lint over directories and over files above or beside the root (lint-structure, which walks
// up to the .git sentinel and reads sibling modules). CheckRoot gets each root as given and sets Path on
// each finding. Spec.Select and ContinueOnWalkError do not apply to it: the runner walks nothing for it.
// A root that holds nothing the rule checks (lint-structure: no profile) is an error like any empty root:
// CheckRoot returns ErrEmptyRoot and the runner prints the Spec.EmptyError row (its Msg is required)
// instead of any finding, stops the rule and exits 3. Any other non-nil error fails the scan closed: the
// runner prints the findings returned with it, then "<lint>: <error>" on stderr, and exits 3.
type RootRule interface {
	ID() string
	Severity() Severity
	CheckRoot(root string) ([]Finding, error)
}

// Configurer is implemented by a rule that reads a value option (Spec.Option, Mode Valued) before it scans:
// when the rule runs alone, Configure gets the parsed Args after the root check and before the walk. A
// non-nil error is printed verbatim on stderr (the rule writes the whole text) and exits 3 with no row.
type Configurer interface {
	Configure(a Args) error
}

// ErrEmptyRoot is what a RootRule or a TreeRule returns for a root that holds nothing it checks.
var ErrEmptyRoot = errors.New("lint: nothing to check under the root")

// Spec registers a rule with the command-line contract of the toolbelt wrapper that runs it alone.
// Exactly one of Rule, Tree and Root is set, and that value implements only its own interface;
// Command panics on any other spec (a registration bug).
type Spec struct {
	Rule      Rule      // a per-file rule, or
	Tree      TreeRule  // a whole-tree rule, or
	Root      RootRule  // a rule that reads its root itself
	Mode      Mode      // how the rule's arguments are parsed when it runs alone
	Usage     string    // printed verbatim on a usage error when it runs alone
	UsageExit int       // the exit code of that usage error; 0 means 3
	Select    Selection // the files under a root the rule reads
	// CheckFirst prints a finding's check column before its severity (`<check>  <SEV>  <path>:<line>  <msg>`),
	// the row of lint-wb-threading; the empty-root ERROR row keeps the standard shape.
	CheckFirst bool
	// Option is the value option Mode Valued parses (e.g. "--wiring-map"); the rule reads its value by
	// implementing Configurer.
	Option string
	// EmptyOK opts a rule out of the empty-root error for a root where it selects no file: no ERROR row,
	// exit 0. Only for a lint whose contract says so (lint-lexicon-ascii LA5: a module with no *.lexicon
	// has nothing to check, and report-module runs it on every module); a walk error still exits 3.
	EmptyOK bool
	// ModuleRoot marks a rule whose roots are module roots, not source roots (lint-duplicate-helper reads
	// every part's src/): n4lint --all, which runs on source roots, leaves it out as it leaves out a
	// RootRule; it runs by name or in a rule list.
	ModuleRoot bool
	// ContinueOnWalkError keeps scanning the files that were listed when the walk could not read a
	// directory (env line, rows, next root, exit 3), as awk_lint_run does. Its zero value stops the rule
	// (env line only, no rows, no later root, exit 3), as scan_files_or_exit does. Set it only with a
	// comment naming the reason, normally that the bash lint being ported used awk_lint_run.
	ContinueOnWalkError bool
	// EmptyError shapes the row of a root where the rule selects no file. Such a root is always an
	// environment error (a wrong path or an empty scaffold, never a clean pass), as the K20 guard of the
	// bash lints made it: the rule prints "ERROR  <check>  <root>  <msg>" on stdout and stops (no later
	// root, exit 3). It is decided per root after the walk, so a walk error wins and prints no ERROR row.
	// There is no opt-out: every registered rule reads source files, and none has a meaning on a root
	// without them. A RootRule decides emptiness itself and returns ErrEmptyRoot; its spec sets Msg. A TreeRule may
	// return ErrEmptyRoot too, when the files it selects hold nothing it checks.
	EmptyError EmptyRow
	// BarePathRow prints a file-level finding (Line 0) as "<path>  <msg>", two spaces and no colon, as
	// lint-structure printed its file and directory rows; a line finding keeps "<path>:<line>  <msg>".
	BarePathRow bool
	// BareUsage prints only Usage on an unknown flag (no "<lint>: unknown flag: <flag>" line), as a bash
	// lint with its own `-*) usage; exit 3` case does; the exit code stays UsageExit.
	BareUsage bool
	// StrictFails prints a WARN row as FAIL under --strict, for a bash lint that rewrote the severity
	// column itself (`v = strict ? "FAIL" : "WARN"`). The exit code is the same either way (1).
	StrictFails bool
	// RelPaths prints a per-file rule's row path with the "<root>/" prefix removed, as a bash lint that
	// printed ${file#"$root/"} did (a root given with a trailing "/" keeps the whole path, as there). Env
	// lines on stderr keep the walk path.
	RelPaths bool
	// IntFlags are the "<name> <n>" options (n a non-negative integer, e.g. "--max-class 800") the rule takes
	// when it runs alone, as a bash lint's AL_FLAG_HOOK took them: among the leading flags, before "--" and
	// the roots, any order with --strict, the last one given winning. A value that is missing or not all
	// digits prints "<lint>: <name> needs a non-negative integer" and the usage line (exit 3); an unknown flag
	// before it stays Parse's unknown-flag error. Configure (required with IntFlags) builds the rule from
	// the values, every flag not given at its Default; in a rule list or under --all the rule runs as
	// registered (its defaults).
	IntFlags  []IntFlag
	Configure func(vals map[string]int) Rule
}

// IntFlag is one Spec.IntFlags option: its name as typed ("--max-class") and its value when not given.
type IntFlag struct {
	Name    string
	Default int
}

// EmptyRow is the ERROR row of Spec.EmptyError.
type EmptyRow struct {
	Check string // the check column; empty shows the rule ID
	Msg   string // the message; empty shows "no <pattern>[ or <pattern>...] sources found (wrong path or empty source tree)"
}

// emptyMsg is the message of the spec's ERROR row on a root where it selects no file.
func (sp Spec) emptyMsg() string {
	if sp.EmptyError.Msg != "" {
		return sp.EmptyError.Msg
	}
	return "no " + strings.Join(sp.Select.Names, " or ") + " sources found (wrong path or empty source tree)"
}

// ID is the registered rule's ID.
func (sp Spec) ID() string { return sp.rule().ID() }

func (sp Spec) severity() Severity { return sp.rule().Severity() }

// rule is the one rule value the spec sets.
func (sp Spec) rule() interface {
	ID() string
	Severity() Severity
} {
	switch {
	case sp.Tree != nil:
		return sp.Tree
	case sp.Root != nil:
		return sp.Root
	}
	return sp.Rule
}

func (sp Spec) usageExit() int {
	if sp.UsageExit != 0 {
		return sp.UsageExit
	}
	return 3
}

// validate enforces that a spec holds exactly one rule kind, and a value of only that kind.
func (sp Spec) validate() error {
	if count(sp.Rule != nil, sp.Tree != nil, sp.Root != nil) != 1 {
		return errors.New("lint: a Spec sets exactly one of Rule, Tree and Root")
	}
	r := sp.rule()
	_, isRule := r.(Rule)
	_, isTree := r.(TreeRule)
	_, isRoot := r.(RootRule)
	if count(isRule, isTree, isRoot) != 1 {
		return errors.New("lint: rule " + r.ID() + " implements more than one of Rule, TreeRule and RootRule")
	}
	if len(sp.IntFlags) > 0 && (sp.Configure == nil || sp.Rule == nil) {
		return errors.New("lint: rule " + r.ID() + " has IntFlags without a per-file Rule and Configure")
	}
	return nil
}

func count(bs ...bool) int {
	n := 0
	for _, b := range bs {
		if b {
			n++
		}
	}
	return n
}

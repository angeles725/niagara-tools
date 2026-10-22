// Package lint is the engine of the kit's Java lints: one Rule interface, one Finding type and row
// formatter, one argument parser (the parse modes of toolbelt/lib/awk-lint.sh), one fail-closed file
// walker (toolbelt/lib/scan-files.sh) and one runner that reads and scans each file once for every
// selected rule. A rule lives in its own package under rules/ and holds only its check; see go/README.md.
package lint

import (
	"errors"
	"fmt"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
)

// Severity is the first column of a row. Its zero value is Unset, so a Finding that leaves it out
// takes its rule's severity.
type Severity int

const (
	Unset Severity = iota // a Finding's "use Rule.Severity()"; never printed (the runner fills it)
	Warn                  // advisory: exit 0, or 1 under --strict
	Fail                  // a failed check: exit 1 with or without --strict
	Pass                  // a check that held (rows print PASS); never affects the exit code
)

func (s Severity) String() string {
	switch s {
	case Fail:
		return "FAIL"
	case Pass:
		return "PASS"
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
func (f Finding) Row() string {
	check := f.Check
	if check == "" {
		check = f.Rule
	}
	if f.Line == 0 {
		return fmt.Sprintf("%s  %s  %s: %s", f.Severity, check, f.Path, f.Msg)
	}
	return fmt.Sprintf("%s  %s  %s:%d  %s", f.Severity, check, f.Path, f.Line, f.Msg)
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
// the whole text, e.g. "cannot scan <path>: <reason>"), and exits 3.
type TreeRule interface {
	ID() string
	Severity() Severity
	CheckTree(files []*javasrc.File) ([]Finding, error)
}

// Spec registers a rule with the command-line contract of the toolbelt wrapper that runs it alone.
// Exactly one of Rule and Tree is set, and that value implements only its own interface; Command
// panics on any other spec (a registration bug).
type Spec struct {
	Rule      Rule      // a per-file rule, or
	Tree      TreeRule  // a whole-tree rule
	Mode      Mode      // how the rule's arguments are parsed when it runs alone
	Usage     string    // printed verbatim on a usage error when it runs alone
	UsageExit int       // the exit code of that usage error; 0 means 3
	Select    Selection // the files under a root the rule reads
}

// ID is the registered rule's ID.
func (sp Spec) ID() string {
	if sp.Tree != nil {
		return sp.Tree.ID()
	}
	return sp.Rule.ID()
}

func (sp Spec) severity() Severity {
	if sp.Tree != nil {
		return sp.Tree.Severity()
	}
	return sp.Rule.Severity()
}

func (sp Spec) usageExit() int {
	if sp.UsageExit != 0 {
		return sp.UsageExit
	}
	return 3
}

// validate enforces that a spec holds exactly one rule kind.
func (sp Spec) validate() error {
	switch {
	case (sp.Rule == nil) == (sp.Tree == nil):
		return errors.New("lint: a Spec sets exactly one of Rule and Tree")
	case sp.Rule != nil:
		if _, ok := sp.Rule.(TreeRule); ok {
			return errors.New("lint: rule " + sp.Rule.ID() + " implements both Rule and TreeRule")
		}
	default:
		if _, ok := sp.Tree.(Rule); ok {
			return errors.New("lint: rule " + sp.Tree.ID() + " implements both Rule and TreeRule")
		}
	}
	return nil
}

// Package lint is the engine of the kit's Java lints: one Rule interface, one Finding type and row
// formatter, one argument parser (the parse modes of the former bash lib lib/awk-lint.sh), one fail-closed file
// walker (the walk of the former lib/scan-files.sh; both libs are removed) and one runner that reads and scans each file once for every
// selected rule. A rule lives in its own package under rules/ and holds only its check; see go/README.md. A rule's
// registration (Spec) is in spec.go.
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
	Unset    Severity = iota // a Finding's "use Rule.Severity()"; never printed (the runner fills it)
	Warn                     // a warning: exit 0, or 1 under --strict
	Fail                     // a failed check: exit 1 with or without --strict
	Pass                     // a check that held (rows print PASS); never affects the exit code
	Advisory                 // a note that is not a failure (rows print ADVISORY); never affects the exit code
	Info                     // information only (rows print INFO); never affects the exit code
	Summary                  // a run total (rows print SUMMARY); never affects the exit code
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
	case Summary:
		return "SUMMARY"
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
	Status   string // the first column printed instead of the severity's name (lint-write-path's STALE, DRIFT); it still counts as Severity
	Path     string
	Line     int
	Msg      string
}

// Row is the line the bash lints print: "WARN  <check>  <path>:<line>  <msg>", or "WARN  <check>  <path>: <msg>"
// for a file-level finding (Line 0), or "SUMMARY  <check>  <msg>" for a finding about the whole run (Line 0 and
// no Path, which only a TreeRule can leave empty). The check column is Check, or Rule when Check is empty.
func (f Finding) Row() string { return f.row(false, false) }

// row is Row, with a file-level finding printed "<path>  <msg>" when bare is set (Spec.BarePathRow) and the
// check column before the severity when checkFirst is set (Spec.CheckFirst).
func (f Finding) row(bare, checkFirst bool) string {
	check := f.Check
	if check == "" {
		check = f.Rule
	}
	first, second := f.Severity.String(), check
	if f.Status != "" {
		first = f.Status
	}
	if checkFirst {
		first, second = check, first
	}
	if f.Line == 0 && f.Path == "" {
		return fmt.Sprintf("%s  %s  %s", first, second, f.Msg)
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

// RowError is a TreeRule or RootRule error printed as Row on stdout, after the findings returned with it,
// instead of "<lint>: <error>" on stderr; it fails the scan closed like any error (exit 3). It is for a
// bash lint whose env errors were stdout rows (lint-write-path's "lint-write-path  ERROR  <root>  <why>").
type RowError struct{ Row string }

func (e *RowError) Error() string { return e.Row }

// Configurer is implemented by a rule that reads a value option (Spec.Option, Mode Valued) before it scans:
// when the rule runs alone, Configure gets the parsed Args after the root check and before the walk and
// returns a NEW rule value of the same kind (Rule, TreeRule or RootRule) holding what it read, which runs
// instead; the registered rule is never changed, so in one process a later run (alone or in a rule list,
// where Configure is not called) never sees an earlier run's values. A non-nil error is printed verbatim on
// stderr (the rule writes the whole text) and exits 3 with no row.
type Configurer interface {
	Configure(a Args) (Configured, error)
}

// Configured is the rule value a Configurer returns: a Rule, a TreeRule or a RootRule, of its spec's kind.
type Configured interface {
	ID() string
	Severity() Severity
}

// withRule is sp running r, the value its rule's Configure returned, in place of its registered rule; a
// value of another kind is a registration bug (panic).
func (sp Spec) withRule(r Configured) Spec {
	var ok bool
	switch {
	case sp.Tree != nil:
		sp.Tree, ok = r.(TreeRule)
	case sp.Root != nil:
		sp.Root, ok = r.(RootRule)
	default:
		sp.Rule, ok = r.(Rule)
	}
	if !ok {
		panic("lint: rule " + sp.ID() + ": Configure returned a value of another kind")
	}
	return sp
}

// ErrEmptyRoot is what a RootRule or a TreeRule returns for a root that holds nothing it checks.
var ErrEmptyRoot = errors.New("lint: nothing to check under the root")

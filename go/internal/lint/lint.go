// Package lint is the engine of the kit's Java lints: one Rule interface, one Finding type and row
// formatter, one argument parser (the parse modes of toolbelt/lib/awk-lint.sh), one fail-closed file
// walker (toolbelt/lib/scan-files.sh) and one runner that reads and scans each file once for every
// selected rule. A rule lives in its own package under rules/ and holds only its check; see go/README.md.
package lint

import (
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
)

func (s Severity) String() string {
	if s == Fail {
		return "FAIL"
	}
	return "WARN"
}

// Finding is one reported source line. A rule's Check sets Line (1-based) and Msg, and Severity only
// when this finding differs from Rule.Severity(); the runner fills Rule, Path and an Unset Severity.
type Finding struct {
	Rule     string
	Severity Severity
	Path     string
	Line     int
	Msg      string
}

// Row is the line the bash lints print: "WARN  <rule>  <path>:<line>  <msg>".
func (f Finding) Row() string {
	return fmt.Sprintf("%s  %s  %s:%d  %s", f.Severity, f.Rule, f.Path, f.Line, f.Msg)
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

// Spec registers a rule with the command-line contract of the toolbelt wrapper that runs it alone.
type Spec struct {
	Rule   Rule
	Mode   Mode      // how the rule's arguments are parsed when it runs alone
	Usage  string    // printed verbatim on a usage error when it runs alone
	Select Selection // the files under a root the rule reads
}

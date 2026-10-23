// Package delays is the lint-delays rule: a Clock.schedule / Clock.schedulePeriodically delay or
// period with no strictly-positive floor, which Niagara rejects at runtime ("time <= 0") and drops the
// schedule. Port of the awk in toolbelt/lint-delays.sh (the wrapper keeps the user-facing contract and
// its header documents the rule and its mutation pins).
//
// It is a TreeRule because a helper method proven to floor its result (D2b) may be declared in any file
// under the root; the report is still one file after another, in walk order.
package delays

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
	"github.com/angeles725/niagara-tools/go/internal/lint"
)

const id = "lint-delays"

// check is the bash row's check column: printf "%-14s" of the lint name.
var check = padRight(id, 14)

// Spec registers the rule with the argument contract of toolbelt/lint-delays.sh: the first argument is
// the root and no flag is parsed; the walk is scan_files_or_exit (a walk error stops the rule). A root
// with no *.java keeps the K20 row of the bash lint's scan_files_nonempty_or_exit.
func Spec() lint.Spec {
	return lint.Spec{
		Tree:       rule{},
		Mode:       lint.Raw,
		Usage:      "usage: lint-delays.sh <java-src-dir>",
		Select:     lint.Selection{Names: []string{"*.java"}},
		EmptyError: lint.EmptyRow{Msg: "no files to scan (K20: wrong path or empty scaffold)"},
	}
}

type rule struct{}

func (rule) ID() string              { return id }
func (rule) Severity() lint.Severity { return lint.Fail }

func (rule) CheckTree(files []*javasrc.File) ([]lint.Finding, error) {
	helpers := map[string]bool{}
	for _, f := range files {
		positiveHelpers(f.Raw, helpers)
	}
	var out []lint.Finding
	for _, f := range files {
		out = append(out, scanFile(f, helpers)...)
	}
	return out, nil
}

// re compiles an awk regular expression: POSIX ERE syntax, leftmost-longest matching.
func re(s string) *regexp.Regexp { return regexp.MustCompilePOSIX(s) }

var (
	helperHead = re(`static[[:space:]]+(final[[:space:]]+)?long[[:space:]]+[A-Za-z_][A-Za-z0-9_]*[[:space:]]*\(`)
	helperMods = re(`static[[:space:]]+(final[[:space:]]+)?long[[:space:]]+`)
	ident      = re(`^[A-Za-z_][A-Za-z0-9_]*`)
	floors     = []*regexp.Regexp{
		re(`Math\.max\([^)]+,[[:space:]]*[1-9]`),
		re(`>=[[:space:]]*1[^0-9]`),
		re(`<[[:space:]]*1[L]?[[:space:]]*\?[[:space:]]*1`),
		re(`return[[:space:]]+1[L]?[[:space:]]*;`),
	}
)

// positiveHelpers adds to set every `static [final] long <fn>(` method of one file whose body (raw text,
// braces counted anywhere) holds a strictly-positive floor: Math.max(.., N>=1), a >= 1 comparison, a
// `< 1 ? 1` ternary or a `return 1;` branch (D2b). A method still open at the end of the file is not one.
func positiveHelpers(lines []string, set map[string]bool) {
	fn, body, depth, inMeth := "", "", 0, false
	adjust := func(ln string) { depth += strings.Count(ln, "{") - strings.Count(ln, "}") }
	done := func() {
		for _, r := range floors {
			if r.MatchString(body) {
				set[fn] = true
				break
			}
		}
		fn, body, depth, inMeth = "", "", 0, false
	}
	for _, ln := range lines {
		if !inMeth && helperHead.MatchString(ln) {
			// The name follows the FIRST `static [final] long` of the line, as awk's match() reads it.
			loc := helperMods.FindStringIndex(ln)
			fn = ident.FindString(ln[loc[1]:])
			inMeth, body, depth = true, ln, 0
			adjust(ln)
			if depth == 0 && strings.Contains(ln, "{") {
				done()
			}
			continue
		}
		if inMeth {
			body += "\n" + ln
			adjust(ln)
			if depth <= 0 {
				done()
			}
		}
	}
}

// num prints an awk number the way awk turns an integral value into a string.
func num(v float64) string { return strconv.FormatFloat(v, 'f', 0, 64) }

// awkIndex is awk's index(s, t): the 1-based position of t in s, 0 when absent (1 for an empty t).
func awkIndex(s, t string) int { return strings.Index(s, t) + 1 }

func padRight(s string, n int) string {
	if len(s) >= n {
		return s
	}
	return s + strings.Repeat(" ", n-len(s))
}

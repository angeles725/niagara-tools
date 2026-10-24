// Package statusparity is the lint-status-parity rule: a facade whose per-instance config slots (a
// @NiagaraProperty name holding Interval, Duration or Setpoint; N > 1) outnumber its scalar status slots
// (a name ending in Status, Active, Start or Since; M < N) WARNs, as operators cannot tell which of the
// N configured instances is active. Port of toolbelt/lint-status-parity.sh (the wrapper keeps the
// user-facing contract and its header documents the check and its mutation pins). Per-file rule: one
// row per file, its subject the file's basename without ".java" (the bash `basename "$f" .java`).
package statusparity

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
	"github.com/angeles725/niagara-tools/go/internal/lint"
)

// ID is the rule's lint name.
const ID = "lint-status-parity"

// Spec registers the rule with the argument contract of toolbelt/lint-status-parity.sh: [--strict] [--]
// and only the first root is read (First), an unknown flag prints the usage line alone, a WARN prints as
// FAIL under --strict, the row subject is the class with no ":" after it, and an incomplete walk or a
// root with no *.java stops the lint (scan_files_nonempty_or_exit).
func Spec() lint.Spec {
	return lint.Spec{
		Rule:        rule{},
		Mode:        lint.First,
		Usage:       "usage: lint-status-parity.sh [--strict] <java-src-dir>",
		Select:      lint.Selection{Names: []string{"*.java"}},
		EmptyError:  lint.EmptyRow{Msg: "no files to scan (K20: wrong path or empty scaffold)"},
		BareUsage:   true,
		StrictFails: true,
		BarePathRow: true,
	}
}

type rule struct{}

func (rule) ID() string              { return ID }
func (rule) Severity() lint.Severity { return lint.Warn }

var (
	nameAttr   = regexp.MustCompile(`name[[:space:]]*=[[:space:]]*"([^"]+)"`)
	configWord = []string{"Interval", "Duration", "Setpoint"}
	statusEnd  = []string{"Status", "Active", "Start", "Since"}
)

// Check counts the file's config and status slot names and WARNs once when N > 1 and M < N. It never
// fails: a file it cannot read is the runner's "cannot scan" error.
func (rule) Check(f *javasrc.File) ([]lint.Finding, error) {
	n, m := 0, 0
	properties(f.Raw, func(text string) {
		sub := nameAttr.FindStringSubmatch(text)
		if sub == nil {
			return
		}
		if hasAny(sub[1], configWord, strings.Contains) {
			n++
		}
		if hasAny(sub[1], statusEnd, strings.HasSuffix) {
			m++
		}
	})
	if n <= 1 || m >= n {
		return nil, nil
	}
	return []lint.Finding{{Path: class(f.Path),
		Msg: fmt.Sprintf("N=%d per-instance config slots but %d status slots", n, m)}}, nil
}

// properties calls fn with the text of every @NiagaraProperty annotation as the bash awk joined it: the
// opening line and the next ones, joined by " ", closed from the second line on once a "(" was seen and
// the parentheses balance. An annotation still open at the end of the file is dropped.
func properties(raw []string, fn func(text string)) {
	open, buf := false, ""
	for _, ln := range raw {
		if !open {
			if strings.Contains(ln, "@NiagaraProperty") {
				open, buf = true, ln
			}
			continue
		}
		buf += " " + ln
		if strings.Count(buf, "(")-strings.Count(buf, ")") <= 0 && strings.Contains(buf, "(") {
			fn(buf)
			open, buf = false, ""
		}
	}
}

// hasAny reports whether match(s, w) holds for a word w of words.
func hasAny(s string, words []string, match func(s, w string) bool) bool {
	for _, w := range words {
		if match(s, w) {
			return true
		}
	}
	return false
}

// class is `basename "$path" .java`: the last path element without its ".java" suffix (kept when it is
// the whole name).
func class(path string) string {
	base := path[strings.LastIndexByte(path, '/')+1:]
	if base == ".java" {
		return base
	}
	return strings.TrimSuffix(base, ".java")
}

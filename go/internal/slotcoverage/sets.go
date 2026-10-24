// Package slotcoverage is toolbelt/slot-coverage.sh: the MM2 exposed-set coverage of a module (set-coverage,
// the type-set parse mode and the per-slot mode). It is not a lint row emitter (three modes, each with its
// own output lines and exit contract), so it is a command of its own over the shared javasrc scanner and
// lint.Walk, not a lint.Spec. Its text model follows the bash it replaces: a shell variable is the string a
// command substitution keeps (subst) and a here-string loop reads its lines back (here).
package slotcoverage

import (
	"fmt"
	"sort"
	"strings"
)

// subst is $(printf '%s\n' lines...): the lines joined, trailing newlines dropped.
func subst(lines []string) string { return strings.TrimRight(strings.Join(lines, "\n"), "\n") }

// here is the lines a `while read` loop reads from <<< "$v" (one empty line for an empty v).
func here(v string) []string { return strings.Split(v, "\n") }

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

// sortU is sort -u in byte order (LC_ALL=C or C.UTF-8).
func sortU(xs []string) []string {
	out := append([]string(nil), xs...)
	sort.Strings(out)
	n := 0
	for i, x := range out {
		if i == 0 || x != out[n-1] {
			out[n] = x
			n++
		}
	}
	return out[:n]
}

// minus is comm -23: the lines of a not in b.
func minus(a, b []string) []string {
	var out []string
	for _, x := range a {
		if !contains(b, x) {
			out = append(out, x)
		}
	}
	return out
}

func nonEmpty(xs []string) []string {
	var out []string
	for _, x := range xs {
		if x != "" {
			out = append(out, x)
		}
	}
	return out
}

// csvLines is csv_to_lines: the comma-separated values of v, empty ones dropped, sorted and deduplicated.
func csvLines(v string) []string { return sortU(nonEmpty(strings.Split(v, ","))) }

// tenths is the integer-tenths percentage of present out of r > 0, as verify-module.sh rounds it.
func tenths(present, r int) string {
	t := (1000*present + r/2) / r
	return fmt.Sprintf("%d.%d", t/10, t%10)
}

// SetCoverage is run_set_coverage: the pct=, missing= and extra= lines of declared against required.
func SetCoverage(declared, required string) []string {
	req, decl := csvLines(required), csvLines(declared)
	pct := "N/A"
	if len(req) > 0 {
		pct = tenths(len(req)-len(minus(req, decl)), len(req))
	}
	return []string{
		"pct=" + pct,
		"missing=" + strings.Join(minus(req, decl), ","),
		"extra=" + strings.Join(minus(decl, req), ","),
	}
}

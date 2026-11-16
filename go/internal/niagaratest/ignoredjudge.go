package niagaratest

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
)

// judgeIgnored compares the root ignored count with the allowance. A scanned source is judged by name
// (judgeNamed); under the count fallback, a scan that names every ignored method refuses the count (the names
// are the allowance), and otherwise judgeCount decides, the fallback's case printed on its row.
func (r *TestNGResult) judgeIgnored(a IgnoredAllowance) {
	switch {
	case a.Scanned && a.Count == 0:
		r.judgeNamed(a)
	case a.Scanned && r.Ignored > 0 && len(a.Source) == r.Ignored:
		r.Reasons = append(r.Reasons, fmt.Sprintf("--allow-ignored-count %d refused: the source names every ignored test-method (%s): "+
			"use --allow-ignored names (--allow-ignored <Class.method>... --allow-ignored-reason <why>); no reason waives a set the source names",
			a.Count, named(a.Source)))
	default:
		r.judgeCount(a, r.fallbackCase(a))
	}
}

// fallbackCase is why the count fallback applies, for its row: the scan refused (i), or the source and TestNG
// counts differ (ii); "" when no source was read (verdict --testng).
func (r *TestNGResult) fallbackCase(a IgnoredAllowance) string {
	switch {
	case a.Refused != "":
		return "case (i): the source scan refused"
	case a.Scanned:
		return fmt.Sprintf("case (ii): the source disables %d, TestNG ignored %d", len(a.Source), r.Ignored)
	}
	return ""
}

// judgeCount compares the root ignored count with the allowed count: ignored methods with no allowance, more than
// allowed (a new disable) and fewer than allowed (a stale allowance) are each a reason; an exact match sets
// Allowed, the row that prints the reason (and why, the fallback's case, when set).
func (r *TestNGResult) judgeCount(a IgnoredAllowance, why string) {
	switch {
	case a.Count == 0 && r.Ignored == 0:
	case a.Count == 0:
		r.Reasons = append(r.Reasons, fmt.Sprintf("%s (@Test(enabled = false) or excluded; testng-results.xml names none): "+
			"waive them with --allow-ignored-count %d --allow-ignored-reason <why>", ignoredMethods(r.Ignored), r.Ignored))
	case r.Ignored > a.Count:
		r.Reasons = append(r.Reasons, fmt.Sprintf("%s, more than the %d allowed (%s): a new disable is never waived",
			ignoredMethods(r.Ignored), a.Count, a.Reason))
	case r.Ignored < a.Count:
		r.Reasons = append(r.Reasons, fmt.Sprintf("%s, fewer than the %d allowed: a stale allowance, lower --allow-ignored-count to %d",
			ignoredMethods(r.Ignored), a.Count, r.Ignored))
	default:
		if why != "" {
			why = ", " + why
		}
		r.Allowed = fmt.Sprintf("%s allowed (--allow-ignored-count %d%s): %s", ignoredMethods(r.Ignored), a.Count, why, a.Reason)
	}
}

func ignoredMethods(n int) string { return plural(n, "ignored test-method", "ignored test-methods") }

// judgeNamed compares the disabled test-methods the source names with the root ignored count (a disagreement is a
// reason: a disable the scan cannot see) and with the allowed names: a disabled method not allowed (new) and an
// allowed name no source disables (stale) are each a reason. Only an exact match on both sets Allowed.
func (r *TestNGResult) judgeNamed(a IgnoredAllowance) {
	before := len(r.Reasons)
	if n := len(a.Source); n != r.Ignored {
		fallback := ""
		if r.Ignored > 0 {
			fallback = fmt.Sprintf("; when the source cannot name them, fall back to --allow-ignored-count %d --allow-ignored-reason <why>", r.Ignored)
		}
		list := ""
		if n > 0 {
			list = " (" + named(a.Source) + ")"
		}
		r.Reasons = append(r.Reasons, fmt.Sprintf("%s, but the registered types' source disables %d%s: source and TestNG disagree "+
			"(a disable the scan cannot see: inherited from outside srcTest/, a group exclusion, an unregistered type)%s",
			ignoredMethods(r.Ignored), n, list, fallback))
	}
	if fresh := missing(a.Source, a.Names); len(fresh) > 0 {
		r.Reasons = append(r.Reasons, fmt.Sprintf("%s not allowed: %s (@Test(enabled = false) in the source; allow each with "+
			"--allow-ignored <Class.method> --allow-ignored-reason <why>)",
			plural(len(fresh), "new disabled test-method", "new disabled test-methods"), named(fresh)))
	}
	if stale := missing(a.Names, a.Source); len(stale) > 0 {
		r.Reasons = append(r.Reasons, fmt.Sprintf("%s: %s (no registered type disables it in the source: drop it)",
			plural(len(stale), "stale --allow-ignored name", "stale --allow-ignored names"), named(stale)))
	}
	if len(r.Reasons) == before && len(a.Source) > 0 {
		r.Allowed = fmt.Sprintf("%s allowed by name (--allow-ignored): %s: %s", ignoredMethods(len(a.Source)), named(a.Source), a.Reason)
	}
}

// missing is every distinct entry of have that want lacks, in have's order.
func missing(have, want []string) []string {
	var out []string
	for _, h := range have {
		if !slices.Contains(want, h) && !slices.Contains(out, h) {
			out = append(out, h)
		}
	}
	return out
}

// named lists names comma-separated, the first maxListed of them.
func named(names []string) string {
	if len(names) <= maxListed {
		return strings.Join(names, ", ")
	}
	return strings.Join(names[:maxListed], ", ") + " and " + strconv.Itoa(len(names)-maxListed) + " more"
}

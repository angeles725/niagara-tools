package niagaratest

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

// maxReason is the longest --allow-ignored-reason, in bytes: the reason is printed on one row.
const maxReason = 200

// IgnoredAllowance waives ignored test-methods (@Test(enabled = false), or excluded by a group). The TestNG that
// test.exe runs (bundled in test-wb.jar) puts only their count in testng-results.xml: XMLReporter adds
// ITestContext.getExcludedMethods() to the root ignored attribute, and XMLSuiteResultWriter writes test-method
// elements for the passed, failed and skipped results alone. So an allowance is a count with its reason, never a
// list of names. Count 0 is no allowance.
type IgnoredAllowance struct {
	Count  int
	Reason string
}

// parseAllowance checks the values of --allow-ignored-count and --allow-ignored-reason (set says each was given):
// both or neither, a count from 1, and a reason of one printable line.
func parseAllowance(count string, countSet bool, reason string, reasonSet bool) (IgnoredAllowance, error) {
	reason = strings.TrimSpace(reason)
	switch {
	case !countSet && !reasonSet:
		return IgnoredAllowance{}, nil
	case !countSet:
		return IgnoredAllowance{}, errors.New("--allow-ignored-reason needs --allow-ignored-count <n>")
	case reason == "":
		return IgnoredAllowance{}, errors.New("--allow-ignored-count needs --allow-ignored-reason <why>")
	case len(reason) > maxReason || strings.IndexFunc(reason, func(r rune) bool { return !unicode.IsPrint(r) }) >= 0:
		return IgnoredAllowance{}, fmt.Errorf("--allow-ignored-reason must be one line of printable text, %d bytes at most", maxReason)
	}
	n, err := strconv.ParseUint(count, 10, 31)
	if err != nil || n == 0 {
		return IgnoredAllowance{}, fmt.Errorf("--allow-ignored-count %q is not a count from 1", count)
	}
	return IgnoredAllowance{int(n), reason}, nil
}

// judgeIgnored compares the root ignored count with the allowance: ignored methods with no allowance, more than
// allowed (a new disable) and fewer than allowed (a stale allowance) are each a reason; an exact match sets
// Allowed, the row that prints the reason.
func (r *TestNGResult) judgeIgnored(a IgnoredAllowance) {
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
		r.Allowed = fmt.Sprintf("%s allowed (--allow-ignored-count %d): %s", ignoredMethods(r.Ignored), a.Count, a.Reason)
	}
}

func ignoredMethods(n int) string { return plural(n, "ignored test-method", "ignored test-methods") }

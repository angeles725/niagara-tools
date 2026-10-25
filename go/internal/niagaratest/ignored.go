package niagaratest

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode"
)

// maxReason is the longest --allow-ignored-reason, in bytes: the reason is printed on one row.
const maxReason = 200

// IgnoredAllowance waives ignored test-methods (@Test(enabled = false), or excluded by a group). The TestNG that
// test.exe runs (bundled in test-wb.jar) puts only their count in testng-results.xml: XMLReporter adds
// ITestContext.getExcludedMethods() to the root ignored attribute, and XMLSuiteResultWriter writes test-method
// elements for the passed, failed and skipped results alone. So verdict's allowance is a count with its reason: the
// file names no method. run names them from the source instead (ignoredsrc.go): Names waives exactly that set
// (--allow-ignored), and Count stays the fallback that skips the source. Count 0 and no Names is no allowance.
type IgnoredAllowance struct {
	Count  int
	Reason string
	Names  []string // --allow-ignored <Class.method>, sorted; never set with Count
	// Source is the disabled test-methods the source names (disabledInSource), set by run when Scanned; then the
	// gate compares them with the root count and with Names.
	Source  []string
	Scanned bool
}

// maxNames is how many --allow-ignored names run takes; methodRe is the shape of one (<Class>.<method>, a
// nested class as Outer$Inner).
const maxNames = 200

var methodRe = regexp.MustCompile(`^[A-Za-z_$][A-Za-z0-9_$]*\.[A-Za-z_$][A-Za-z0-9_$]*$`)

// namesFlag is the repeatable --allow-ignored.
type namesFlag struct{ names *[]string }

func (f namesFlag) String() string { return "" }
func (f namesFlag) Set(v string) error {
	switch {
	case len(*f.names) == maxNames:
		return fmt.Errorf("more than %d names", maxNames)
	case !methodRe.MatchString(v):
		return fmt.Errorf("%q is not <Class>.<method>", v)
	case slices.Contains(*f.names, v):
		return fmt.Errorf("%s given twice", v)
	}
	*f.names = append(*f.names, v)
	return nil
}

// parseRunAllowance is parseAllowance with run's --allow-ignored names: names and a count are refused together,
// and names need a reason as a count does.
func parseRunAllowance(count string, countSet bool, reason string, reasonSet bool, names []string) (IgnoredAllowance, error) {
	if len(names) == 0 {
		return parseAllowance(count, countSet, reason, reasonSet)
	}
	if countSet {
		return IgnoredAllowance{}, errors.New("--allow-ignored names the methods and --allow-ignored-count counts them: give one, not both")
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return IgnoredAllowance{}, errors.New("--allow-ignored needs --allow-ignored-reason <why>")
	}
	if err := checkReason(reason); err != nil {
		return IgnoredAllowance{}, err
	}
	sorted := slices.Clone(names)
	slices.Sort(sorted)
	return IgnoredAllowance{Reason: reason, Names: sorted}, nil
}

func checkReason(reason string) error {
	if len(reason) > maxReason || strings.IndexFunc(reason, func(r rune) bool { return !unicode.IsPrint(r) }) >= 0 {
		return fmt.Errorf("--allow-ignored-reason must be one line of printable text, %d bytes at most", maxReason)
	}
	return nil
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
	}
	if err := checkReason(reason); err != nil {
		return IgnoredAllowance{}, err
	}
	n, err := strconv.ParseUint(count, 10, 31)
	if err != nil || n == 0 {
		return IgnoredAllowance{}, fmt.Errorf("--allow-ignored-count %q is not a count from 1", count)
	}
	return IgnoredAllowance{Count: int(n), Reason: reason}, nil
}

// judgeIgnored compares the root ignored count with the allowance: ignored methods with no allowance, more than
// allowed (a new disable) and fewer than allowed (a stale allowance) are each a reason; an exact match sets
// Allowed, the row that prints the reason. A scanned source is judged by judgeNamed instead.
func (r *TestNGResult) judgeIgnored(a IgnoredAllowance) {
	if a.Scanned {
		r.judgeNamed(a)
		return
	}
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

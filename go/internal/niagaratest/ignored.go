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
// (--allow-ignored), and Count is the fallback for a source that cannot name it; run scans the source under the
// count too, and a scan that names every ignored method FAILs the count. Count 0 and no Names is no allowance.
type IgnoredAllowance struct {
	Count  int
	Reason string
	Names  []string // --allow-ignored <Class.method>, sorted; never set with Count
	// Source is the disabled test-methods the source names (disabledInSource), set by run when Scanned; then the
	// gate compares them with the root count and with Names. Refused is why the scan could not name them (set by
	// run under Count only: with names or no allowance a refusal stops the run).
	Source  []string
	Scanned bool
	Refused string
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

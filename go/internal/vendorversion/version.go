// Package vendorversion is the one Go reader and comparator of a module's version: the retired bash lib
// toolbelt/lib/vendor-version.sh (vv_module_version, vv_source_version, vv_cmp) answered the same way.
// Consumers: build-module (deployed-baseline drift gate), ng-deploy (version precheck) and
// check-installed-line (downgrade guard).
package vendorversion

import (
	"regexp"
	"strings"
)

var (
	// sourceCall is an uncommented `defaultModuleVersion("X")` call at the start of a line.
	sourceCall = regexp.MustCompile(`^[ \t\v\f\r]*defaultModuleVersion\([ \t\v\f\r]*"([^"]*)"`)
	// dottedNumber is the leading number of a version: digits, then dot-separated digit groups.
	dottedNumber = regexp.MustCompile(`^[0-9]+(\.[0-9]+)*`)
)

// SourceVersion is the SOURCE-side module version of a GROUP build.gradle.kts text: the version of its
// first uncommented defaultModuleVersion("X") call, "" when there is none (callers fail closed on "").
func SourceVersion(text string) string {
	for _, line := range strings.Split(text, "\n") {
		if m := sourceCall.FindStringSubmatch(line); m != nil {
			return m[1]
		}
	}
	return ""
}

// Cmp compares two versions: -1 (a < b), 0 or 1. Each dotted component compares as a number
// (1.10 > 1.9), a missing component is 0 (1.3 == 1.3.0, 1.03 == 1.3). ok is false when the pair is
// incomparable: a side with no leading number, or text after the number (1.3-rc) that differs.
func Cmp(a, b string) (c int, ok bool) {
	na, nb := dottedNumber.FindString(a), dottedNumber.FindString(b)
	if na == "" || nb == "" || a[len(na):] != b[len(nb):] {
		return 0, false
	}
	xs, ys := strings.Split(na, "."), strings.Split(nb, ".")
	for i := 0; i < len(xs) || i < len(ys); i++ {
		if c := cmpNumber(component(xs, i), component(ys, i)); c != 0 {
			return c, true
		}
	}
	return 0, true
}

// component is the i-th digit group without its leading zeros ("" is zero, also when i is past the end).
func component(groups []string, i int) string {
	if i >= len(groups) {
		return ""
	}
	return strings.TrimLeft(groups[i], "0")
}

// cmpNumber compares two digit strings with no leading zero by value, at any length.
func cmpNumber(x, y string) int {
	if len(x) != len(y) {
		if len(x) < len(y) {
			return -1
		}
		return 1
	}
	return strings.Compare(x, y)
}

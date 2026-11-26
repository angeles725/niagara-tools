package vendored

import (
	"errors"
	"regexp"
	"strconv"
	"strings"
)

// version is a vendored library's version: a three.js style revision ("r161": one integer) or a semver
// ("1.4.2", a leading v and missing minor/patch parts allowed, an optional "-pre-release" tag).
type version struct {
	revision bool   // true: an r-number; false: a semver
	nums     [3]int // the revision in nums[0], or major, minor, patch
	pre      string // a semver pre-release tag; "" for a release
}

var (
	revisionRE = regexp.MustCompile(`^[rR](\d+)$`)
	semverRE   = regexp.MustCompile(`^[vV]?(\d+)(?:\.(\d+))?(?:\.(\d+))?(?:-([0-9A-Za-z.-]+))?(?:\+[0-9A-Za-z.-]+)?$`)
)

// parseVersion reads s as an r-number or a semver; ok is false for any other text.
func parseVersion(s string) (v version, ok bool) {
	s = strings.TrimSpace(s)
	if m := revisionRE.FindStringSubmatch(s); m != nil {
		n, err := strconv.Atoi(m[1])
		return version{revision: true, nums: [3]int{n}}, err == nil
	}
	m := semverRE.FindStringSubmatch(s)
	if m == nil {
		return v, false
	}
	for i := 0; i < 3; i++ {
		if m[i+1] == "" {
			continue
		}
		n, err := strconv.Atoi(m[i+1])
		if err != nil {
			return v, false
		}
		v.nums[i] = n
	}
	v.pre = m[4]
	return v, true
}

// compare returns -1, 0 or 1 as v is below, equal to or above o; ok is false when an r-number meets a semver
// (no order between them). A release is above its pre-releases; pre-release tags compare as plain text.
func (v version) compare(o version) (int, bool) {
	if v.revision != o.revision {
		return 0, false
	}
	for i := range v.nums {
		if v.nums[i] != o.nums[i] {
			return sign(v.nums[i] - o.nums[i]), true
		}
	}
	switch {
	case v.pre == o.pre:
		return 0, true
	case v.pre == "":
		return 1, true
	case o.pre == "":
		return -1, true
	}
	return sign(strings.Compare(v.pre, o.pre)), true
}

func sign(n int) int {
	switch {
	case n < 0:
		return -1
	case n > 0:
		return 1
	}
	return 0
}

// constraint is one comparison of a known-defects "Affected" cell: op is <, <=, >, >= or =.
type constraint struct {
	op  string
	ver version
	raw string // the comparison as written ("< r161")
}

// parseAffected reads an "Affected" cell: comparisons joined by commas ("< r161", ">= 1.0.0, < 1.4.2").
func parseAffected(cell string) ([]constraint, error) {
	var out []constraint
	for _, part := range strings.Split(cell, ",") {
		part = strings.TrimSpace(part)
		op := ""
		for _, o := range []string{"<=", ">=", "<", ">", "="} {
			if strings.HasPrefix(part, o) {
				op = o
				break
			}
		}
		v, ok := parseVersion(strings.TrimPrefix(part, op))
		if op == "" || !ok {
			return nil, errors.New("Affected " + strconv.Quote(strings.TrimSpace(cell)) +
				" is not a comparison (< r161, >= 1.0.0, < 1.4.2)")
		}
		out = append(out, constraint{op: op, ver: v, raw: op + " " + strings.TrimSpace(strings.TrimPrefix(part, op))})
	}
	return out, nil
}

// affects reports whether v meets every comparison; ok is false when one of them cannot be compared with v.
func affects(cs []constraint, v version) (hit, ok bool) {
	hit = true
	for _, c := range cs {
		n, comparable := v.compare(c.ver)
		if !comparable {
			return false, false
		}
		switch c.op {
		case "<":
			hit = hit && n < 0
		case "<=":
			hit = hit && n <= 0
		case ">":
			hit = hit && n > 0
		case ">=":
			hit = hit && n >= 0
		default:
			hit = hit && n == 0
		}
	}
	return hit, true
}

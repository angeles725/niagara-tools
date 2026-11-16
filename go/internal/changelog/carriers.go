package changelog

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

// MaxReasonLen caps the reason of a `Changelog: none (<reason>)` trailer, in characters.
const MaxReasonLen = 120

var trailer = regexp.MustCompile(`^Changelog: none \((.*)\)$`)

// CheckReason returns why a trailer reason is not acceptable, or "": it must be non-empty, one printable
// line and at most MaxReasonLen characters.
func CheckReason(reason string) string {
	r := strings.TrimSpace(reason)
	switch {
	case r == "":
		return "the reason is empty"
	case utf8.RuneCountInString(r) > MaxReasonLen:
		return fmt.Sprintf("the reason is longer than %d characters", MaxReasonLen)
	case strings.IndexFunc(r, func(c rune) bool { return !unicode.IsPrint(c) }) >= 0:
		return "the reason holds a non-printable character"
	}
	return ""
}

// fragmentsIn checks each changelog.d/ path the range adds or modifies (as head holds it) and returns the
// valid ones and one problem line per invalid one; a path outside changelog.d/ is skipped.
func fragmentsIn(paths []string, head string, git gitRunner) (fragments, problems []string, err error) {
	for _, p := range paths {
		if !strings.HasPrefix(p, FragmentDir+"/") {
			continue
		}
		why := CheckName(strings.TrimPrefix(p, FragmentDir+"/"))
		if why == "" {
			body, err := git("show", head+":"+p)
			if err != nil {
				return nil, nil, err
			}
			why = CheckFragment(body)
		}
		if why != "" {
			problems = append(problems, fmt.Sprintf("%s: %s", p, why))
			continue
		}
		fragments = append(fragments, p)
	}
	return fragments, problems, nil
}

// trailersIn returns the reasons of the valid `Changelog: none (<reason>)` lines in the range's commit
// messages and one problem line per `Changelog:` line that is not of that form.
func trailersIn(logOut string) (reasons, problems []string) {
	for _, l := range strings.Split(logOut, "\n") {
		l = strings.TrimSpace(l)
		if !strings.HasPrefix(l, "Changelog:") {
			continue
		}
		m := trailer.FindStringSubmatch(l)
		why := "not the form 'Changelog: none (<reason>)'"
		if m != nil {
			why = CheckReason(m[1])
		}
		if why != "" {
			problems = append(problems, fmt.Sprintf("trailer %q: %s", l, why))
			continue
		}
		reasons = append(reasons, strings.TrimSpace(m[1]))
	}
	return reasons, problems
}

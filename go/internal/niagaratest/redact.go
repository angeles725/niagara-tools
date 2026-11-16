package niagaratest

import (
	"regexp"
	"sort"
	"strings"
)

// redaction is one path the log hides behind label, in its Windows form (also spelt with forward slashes)
// and its WSL form; an empty form is skipped.
type redaction struct{ win, wsl, label string }

type redactor []struct {
	re    *regexp.Regexp
	label string
}

// newRedactor matches every form case-insensitively and only as a whole path component (C:\Users\you is not
// a prefix of C:\Users\young), the longest path first so the home copy wins over the scratch dir around it.
func newRedactor(rs ...redaction) redactor {
	type form struct{ path, label string }
	var forms []form
	for _, r := range rs {
		for _, p := range []string{r.win, strings.ReplaceAll(r.win, `\`, "/"), r.wsl} {
			if p = strings.TrimRight(p, `\/`); len(p) > 1 {
				forms = append(forms, form{p, r.label})
			}
		}
	}
	sort.SliceStable(forms, func(i, j int) bool { return len(forms[i].path) > len(forms[j].path) })
	var out redactor
	for _, f := range forms {
		re := regexp.MustCompile(`(?i)` + regexp.QuoteMeta(f.path) + `([^A-Za-z0-9_.\-]|$)`)
		out = append(out, struct {
			re    *regexp.Regexp
			label string
		}{re, f.label})
	}
	return out
}

func (r redactor) apply(s string) string {
	for _, x := range r {
		s = x.re.ReplaceAllString(s, x.label+"${1}")
	}
	return s
}

// redactorFor hides the scratch dir, its home copy, %USERPROFILE% and the WSL home; install paths stay, the
// user's own and needed to diagnose.
func (h host) redactorFor(p *plan) redactor {
	return newRedactor(redaction{p.home().win, p.home().wsl, "<copy>"}, redaction{p.dir.win, p.dir.wsl, "<scratch>"},
		redaction{p.profile.win, p.profile.wsl, "<you>"}, redaction{"", h.home, "<you>"})
}

// Package timers is lint-timers: the timer-lifecycle checks of Niagara N4 module Java, ported
// byte for byte from the bash/awk toolbelt/lint-timers.sh. Its report is ordered by check, each check
// over every file in walk order, so it is a lint.TreeRule. The check families live one per file:
// ticket.go (discarded-ticket, timer-ticket), companion.go (companion-flag), threads.go (jdk-thread,
// job-cancel), sched.go (changed-sched, atSteadyState-only-timer) and lifecycle.go (orphan-flag,
// release-gate-parity). awk.go keeps gawk's matching semantics for all of them.
package timers

import (
	"regexp"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
	"github.com/angeles725/niagara-tools/go/internal/lint"
)

// ID is the rule ID: the n4lint rule name and the prefix of its stderr lines.
const ID = "lint-timers"

// Spec registers lint-timers with the argument contract of toolbelt/lint-timers.sh: exactly one root,
// no flags, a usage error exits 2.
func Spec() lint.Spec {
	return lint.Spec{
		Tree:      rule{},
		Mode:      lint.Exact,
		Usage:     "usage: lint-timers.sh <java-root>",
		UsageExit: 2,
		Select:    lint.Selection{Names: []string{"*.java"}},
	}
}

type rule struct{}

func (rule) ID() string              { return ID }
func (rule) Severity() lint.Severity { return lint.Fail }

// CheckTree runs the check passes in the order the bash lint ran them. A module-permissions.xml that
// cannot be read stops the run there (the bash lint exited 3 at that file).
func (rule) CheckTree(files []*javasrc.File) ([]lint.Finding, error) {
	r := &run{rx: rxCache{}, perms: map[string]grant{}}
	srcs := make([]*source, len(files))
	for i, f := range files {
		srcs[i] = &source{File: f}
	}
	r.ticketPass(srcs)
	r.companionPass(srcs)
	if err := r.threadPass(srcs); err != nil {
		return r.out, err
	}
	r.changedSchedPass(srcs)
	r.steadyStatePass(srcs)
	r.lifecyclePass(srcs)
	return r.out, nil
}

// run is one CheckTree call: the rows so far and the caches shared by the passes.
type run struct {
	out   []lint.Finding
	rx    rxCache
	perms map[string]grant // nearest module-permissions.xml lookup, by directory
}

// row appends a file-level row "<SEV>  <check>  <path>: <msg>".
func (r *run) row(sev lint.Severity, check string, f *source, msg string) {
	r.out = append(r.out, lint.Finding{Severity: sev, Check: check, Path: f.Path, Msg: msg})
}

// source is one file plus the views the awk programs derived from it, each computed once.
type source struct {
	*javasrc.File
	code    []string // blank_str over the stripped lines (ticket.awk, threads.awk)
	methods []javasrc.Method
}

// blanked is code[] of ticket.awk and threads.awk: comments stripped, literal contents blanked.
func (s *source) blanked() ([]string, []javasrc.Method) {
	if s.code == nil {
		st := s.Stripped()
		s.code = make([]string, len(st))
		for i, ln := range st {
			s.code[i] = blankStr(ln)
		}
		s.methods = javasrc.Methods(s.code)
	}
	return s.code, s.methods
}

// joinLines is the awk "for (i = a; i <= b; i++) s = s line[i] sep" accumulation (a, b 1-based).
func joinLines(lines []string, a, b int, before, after string) string {
	n := 0
	for i := a; i <= b; i++ {
		n += len(before) + len(lines[i-1]) + len(after)
	}
	buf := make([]byte, 0, n)
	for i := a; i <= b; i++ {
		buf = append(buf, before...)
		buf = append(buf, lines[i-1]...)
		buf = append(buf, after...)
	}
	return string(buf)
}

// inMethod marks the lines inside any method body (inm[] of the awk programs).
func inMethod(n int, ms []javasrc.Method) []bool {
	in := make([]bool, n+1)
	for _, m := range ms {
		for i := m.Start; i <= m.End; i++ {
			in[i] = true
		}
	}
	return in
}

var scheduleCall = regexp.MustCompile(`Clock\.schedule(Periodically)?\(`)

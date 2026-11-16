package reportmodule

import (
	"fmt"
	"strings"
)

// How the member table (members.go) is run: one step per member, the common lint and advisory row shapes and
// the fail-closed rows.

// step runs one member on a, when a's name carries the member's suffix.
func (r *report) step(a artifact, m member) {
	if !strings.HasSuffix(a.name, m.suffix) {
		return
	}
	r.inFloor = m.floor
	if m.special != nil {
		m.special(r, a)
		return
	}
	dir := a.dir
	switch m.in {
	case inSrc:
		if !r.haveSrc(a, m.script) {
			return
		}
		dir += "/src"
	case inRc:
		if dir += "/src/rc"; !isDir(dir) {
			r.emit(a.name, skip, m.script, "no src/rc/")
			return
		}
	}
	out, code, ok := r.x.member(m.script, dir)
	if r.truncated(a.name, m.script, m.script+".sh", ok) {
		return
	}
	switch {
	case m.advisory:
		r.advisoryMember(a.name, m.script, code, out, m.modeAware)
	case m.noSource != "" && code == 3 && strings.Contains(out, m.noSource):
		r.emit(a.name, skip, m.script, m.noSource)
	default:
		r.lintRows(a.name, m.script, code, out, m.f, m.v)
	}
}

// lintRows is the common lint shape: exit 3 is an ERROR env row; any other exit relays the selected rows and
// then the verdict row, or a FAIL row when the exit was non-zero and no FAIL row was relayed.
func (r *report) lintRows(label, check string, code int, out string, f filter, v verdict) {
	if code == 3 {
		r.envFault(label, check, "env fault (exit 3)")
		return
	}
	rl := r.memberRows(label, out, f)
	switch {
	case code != 0 && !rl.fail:
		r.exitedNoFail(label, check, code)
	case v == rows && rl.rows == 0 || v == failRows && !rl.fail:
		r.emit(label, pass, check, "clean")
	}
}

// exitedNoFail is the fail-closed row of a member that exited non-zero (not 3) with no FAIL row: 127 (the lint
// missing), a crash or an unexpected row format is never PASS "clean" and never silent.
func (r *report) exitedNoFail(label, check string, code int) {
	r.emit(label, failed, check, fmt.Sprintf("exited %d with no FAIL row (crash or unexpected output) -- run %s.sh manually", code, check))
}

// advisoryMember is a lint run without --strict whose rows do not fail the report by themselves: exit 0 relays its
// rows and then the verdictAfterRelay row (PASS, as its rows are WARN/ADVISORY). A mode-aware member (lint-size,
// whose rows FAIL in a `modularity: strict` module) exiting 1 with a FAIL row relays its rows, the FAIL ones
// counted, and adds no PASS row. Any other exit is an ERROR row carrying the member's first output line that is
// neither blank nor one of its own WARN/FAIL/ADVISORY rows, never a silent PASS.
func (r *report) advisoryMember(label, check string, code int, out string, modeAware bool) {
	if code == 0 || modeAware && code == 1 && hasFailRow(out) {
		r.verdictAfterRelay(label, check, r.relayRows(label, out))
		return
	}
	detail := fmt.Sprintf("env fault (exit %d)", code)
	for _, ln := range strings.Split(out, "\n") {
		if len(awkFields(ln)) > 0 && !hasPrefixIn(ln, "WARN  |FAIL  |ADVISORY  ") {
			detail += ": " + ln
			break
		}
	}
	r.envFault(label, check, detail)
}

// hasFailRow reports whether out holds a `FAIL  <check>  ...` row (the status exactly FAIL).
func hasFailRow(out string) bool {
	for _, w := range parseRows(out, failOnly) {
		if w.st == failed {
			return true
		}
	}
	return false
}

// truncated reports a run whose output passed the bound (ok false) as an ERROR row: the rows past the bound, a
// FAIL among them, are lost, so the member is never judged on the rest.
func (r *report) truncated(label, check, what string, ok bool) bool {
	if !ok {
		r.envFault(label, check, fmt.Sprintf("env fault (output over %d bytes, truncated) -- run %s manually", r.x.bound(), what))
	}
	return !ok
}

// haveSrc is true when <artifact>/src is a directory; otherwise it emits the `SKIP <check> no src/` row.
func (r *report) haveSrc(a artifact, check string) bool {
	if isDir(a.dir + "/src") {
		return true
	}
	r.emit(a.name, skip, check, "no src/")
	return false
}

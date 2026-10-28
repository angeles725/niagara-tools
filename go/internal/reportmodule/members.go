package reportmodule

import (
	"fmt"
	"strings"
)

// input is the directory a member reads.
type input int

const (
	inSrc      input = iota // <artifact>/src; a SKIP "no src/" row when it is not a directory
	inArtifact              // <artifact> itself (the module root for a module member), ungated
	inRc                    // <artifact>/src/rc; a SKIP "no src/rc/" row when it is not a directory
)

// verdict is the row a lint member adds after its relayed rows.
type verdict int

const (
	none     verdict = iota // no verdict row: the member's own rows speak
	rows                    // PASS "clean" when no row was relayed
	failRows                // PASS "clean" when no FAIL row was relayed (WARN rows do not block)
)

// member is one step of the report. A table member runs <toolbelt>/<script>.sh on its input and relays the
// rows filter f selects, then its verdict; an advisory one relays its WARN rows with a PASS verdict row and turns
// any non-zero exit into an ERROR row; a module member with a noSource text SKIPs on its own "no ... sources"
// exit 3; a table member's rows carry its script name as the check. A special member (special.go) does its own
// rows and names its own checks (plano, vendor-floor, eslint, dup-keys, ...).
type member struct {
	script   string // toolbelt member, without .sh ("" for ESLint, which is not a toolbelt script)
	suffix   string // the artifact name must end with it ("" any artifact)
	in       input
	f        filter
	v        verdict
	advisory bool
	// modeAware: an advisory member whose rows FAIL in a `modularity: strict` module (lint-size), so its exit 1
	// with a FAIL row is relayed as FAIL; any other advisory member's exit 1 is an env fault.
	modeAware bool
	noSource  string
	special   func(r *report, a artifact)
}

// artifact is what a member runs on: a profile artifact (its directory name labels the rows) or, for the
// once-per-module members, the module root labelled "(module)".
type artifact struct{ name, dir string }

var (
	failWarn = filter{sel: "FAIL|WARN"}
	warnOnly = filter{sel: "WARN"}
	failOnly = filter{sel: "FAIL"}
)

// perArtifact is every per-artifact step, in report order (the bash §1-§6).
var perArtifact = []member{
	{script: "verify-module", special: (*report).verifyModule},
	{script: "slot-coverage", special: (*report).slotCoverage},
	{script: "lint-timers", special: (*report).lintTimers},
	{script: "verify-module", special: (*report).plano},
	{script: "lint-lexicon-ascii", special: (*report).lexiconASCII},
	// kit-backlog B11.6: the lexicon-ascii companion; a clean or lexicon-less artifact prints nothing.
	{script: "lint-lexicon-apostrophe", in: inArtifact, f: warnOnly},
	{script: "lint-delays", f: failWarn},
	{script: "lint-demand-scope", f: warnOnly},
	{script: "lint-silent-protection", f: filter{sel: "FAIL|WARN|ADVISORY"}},
	{script: "lint-link-target-flags", special: (*report).linkTargetFlags},
	{script: "lint-ext-writable-shape", f: failWarn},
	{script: "lint-no-system-out", f: failOnly, v: rows},
	{script: "lint-slotomatic-markers", f: failWarn, v: rows},
	{script: "lint-clock-zero-floor", f: warnOnly},
	{script: "lint-null-context-write", f: warnOnly},
	{script: "lint-bql-string-concat", f: warnOnly},
	{script: "lint-arbitrary-ord", f: warnOnly},
	{script: "lint-persist-hot-write", f: warnOnly},
	{script: "lint-changed-hot-write", f: warnOnly},
	{script: "lint-inert-coordination", f: warnOnly},
	{script: "lint-session-store-lazy-evict", f: warnOnly},
	{script: "lint-bformat-call", f: warnOnly},
	// kit-backlog B10.8: the course-coverage source lints (corpus B1417, B1419). No verdict row, as lint-service-types.
	{script: "lint-charset-default", f: warnOnly},
	{script: "lint-slot-default-type", f: failOnly},
	// kit-backlog B11.6: the course-coverage run B source lints (corpus B1242 K3, K4), advisory WARN rows.
	{script: "lint-lifecycle-guard", f: warnOnly},
	{script: "lint-intrinsic-lock", f: warnOnly},
	{script: "lint-service-types", f: failWarn},
	{script: "lint-driver-scaffold", f: failWarn},
	{script: "lint-se-display", suffix: "-se", f: failOnly, v: rows},
	{script: "lint-jasmine-ux", suffix: "-ux", in: inArtifact, f: filter{sel: "WARN", mode: nosite}},
	{script: "lint-wb-external-ord-value", suffix: "-wb", f: warnOnly},
	{script: "lint-spa-poll-no-recovery", suffix: "-ux", in: inRc, f: warnOnly},
	{script: "lint-subscribe-without-unsubscribe", suffix: "-rt", f: warnOnly},
	{script: "lint-recovery-path", suffix: "-rt", f: failOnly, v: rows},
	{script: "lint-config-sanity", suffix: "-rt", f: failWarn, v: failRows},
	{script: "lint-status-parity", suffix: "-rt", f: warnOnly},
	{script: "lint-servlet", suffix: "-ux", special: (*report).servlet},
	{script: "rc-scan", suffix: "-ux", special: (*report).rcScan},
	{suffix: "-ux", special: (*report).eslint},
	{script: "lint-vendor-floor", suffix: "-ux", special: (*report).vendorFloor},
	{script: "lint-wb-threading", suffix: "-wb", f: filter{mode: swap, keep: "WARN"}},
	{script: "lint-no-md5-credential-digest", f: warnOnly, v: rows},
	{script: "lint-wb-file-chooser", suffix: "-wb", f: warnOnly},
	{script: "lint-size", advisory: true, modeAware: true},
	{script: "lint-license-isoperational-gate", advisory: true},
	{script: "lint-set-null-ord", advisory: true},
	{script: "lint-ws-placement", advisory: true},
	{script: "schema-risk", special: (*report).schemaRisk},
}

// perModule is every once-per-module step on the module root, in report order (the bash §8-§13).
var perModule = []member{
	{script: "lint-agent-on-shape", in: inArtifact, f: failOnly, v: rows},
	{script: "lint-agent-on-include-sync", in: inArtifact, f: failOnly, v: rows},
	{script: "lint-uberjar-api-conflict", in: inArtifact, f: filter{sel: "WARN", mode: base}},
	{script: "lint-structure", in: inArtifact, f: failWarn, v: failRows},
	{script: "lint-module-map", in: inArtifact, f: filter{sel: "FAIL|WARN", mode: base}, v: failRows},
	{special: (*report).modularity},
	{script: "lint-write-path", special: (*report).writePath},
	{script: "lint-bundled-jar-class-version", in: inArtifact, f: filter{sel: "FAIL", mode: base}, v: rows},
	{script: "lint-duplicate-helper", in: inArtifact, f: warnOnly, noSource: "no *.java under a -rt, -ux or -wb part's src/"},
	{script: "lint-machine-default-path", in: inArtifact, f: warnOnly, noSource: "no *.java or *.js or *.properties sources found"},
	{script: "lint-inline-site-literal", in: inArtifact, f: warnOnly, noSource: "no *.java under a -rt or -ux part's src/ and no *.js under an rc/"},
	{script: "lint-private-api-import", in: inArtifact, f: warnOnly, noSource: "no *.java under a -rt, -ux, -wb or -se part's src/"},
	{script: "lint-ndriver-static-collection", in: inArtifact, f: warnOnly, noSource: "no *.java under a -rt part's src/"},
	{script: "lint-ndriver-tagless-correlation", in: inArtifact, f: warnOnly, noSource: "no *.java under a -rt part's src/"},
	// a module with no srcTest/ *.java has nothing to judge: its exit 3 is a SKIP row, never an env fault
	// Mutation: RM-vacuous-nosource -- dropping noSource turns every test-less module's report into an env fault
	{script: "lint-vacuous-tests", in: inArtifact, f: failOnly, v: rows, noSource: "no *.java under a srcTest/"},
	// kit-backlog B5.5: the run-3 module lints. A module without their sources SKIPs, never an env fault.
	// Mutation: RM-new-members-nosource -- dropping noSource turns a module with no permissions file or -rt Java into an env fault
	{script: "lint-permissions-xml", in: inArtifact, f: failWarn, v: failRows, noSource: "no module-permissions.xml and no *.java under a -rt part's src/"},
	{script: "lint-structural-tests", in: inArtifact, f: warnOnly, noSource: "no *.java under a srcTest/"},
	{script: "lint-logger-name", in: inArtifact, f: warnOnly, noSource: "no *.java under a -rt, -ux, -wb or -se part's src/"},
	{script: "lint-adapter-cfg-default", in: inArtifact, f: warnOnly, noSource: "no *.java under a -rt part's src/"},
}

// Scripts lists every toolbelt member the report runs, as script file names (lint-timers.sh, ...), each once
// and in report order; triage-console.sh is the last, run once per report. toolbelt/INDEX.md's Auto column
// reads it (go/internal/lintindex).
func Scripts() []string {
	var out []string
	seen := map[string]bool{}
	for _, list := range [][]member{perArtifact, perModule, {{script: "triage-console"}}} {
		for _, m := range list {
			if m.script != "" && !seen[m.script] {
				seen[m.script] = true
				out = append(out, m.script+".sh")
			}
		}
	}
	return out
}

// step runs one member on a, when a's name carries the member's suffix.
func (r *report) step(a artifact, m member) {
	if !strings.HasSuffix(a.name, m.suffix) {
		return
	}
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

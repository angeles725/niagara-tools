package reportmodule

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// The members whose rows the shared parser does not fully cover, in report order.

// verifyModule is §1: verify-module --src <module-root> on the artifact's first build/libs/*.jar; no jar is a
// floor SKIP.
func (r *report) verifyModule(a artifact) {
	jar := firstJar(a.dir + "/build/libs")
	if jar == "" {
		r.emit(a.name, skip, "verify-module", "no built jar under build/libs (floor check not run)")
		r.floorSkip = true
		return
	}
	var args []string
	if r.o.targetVersion != "" {
		args = append(args, "--target-version", r.o.targetVersion)
	}
	out, code := r.x.member("verify-module", append(args, "--src", r.o.root, jar)...)
	if code == 3 {
		r.envFault(a.name, "verify-module", "env fault (exit 3)")
		return
	}
	r.memberRows(a.name, out, filter{sel: "PASS|FAIL|SKIP|WARN", mode: ws})
}

// firstJar is `find <libs> -maxdepth 1 -name '*.jar' | head -1`: the first entry, in directory order, whose
// name ends in .jar (of any type) in libs, a directory that is not a symlink; "" when there is none.
func firstJar(libs string) string {
	if !isRealDir(libs) {
		return ""
	}
	d, err := os.Open(libs)
	if err != nil {
		return ""
	}
	defer d.Close()
	names, _ := d.Readdirnames(-1) // directory order, as find reads it
	for _, n := range names {
		if strings.HasSuffix(n, ".jar") {
			return libs + "/" + n
		}
	}
	return ""
}

// slotCoverage is §2: slot-coverage on module-include.xml and module.lexicon when both are files: the dup-keys
// count, every other "slot-coverage: FAIL" line, then the pct= coverage as PASS (>= 100), WARN or SKIP N/A.
func (r *report) slotCoverage(a artifact) {
	mixin, lex := a.dir+"/module-include.xml", a.dir+"/module.lexicon"
	if !isFile(mixin) || !isFile(lex) {
		return
	}
	out, code := r.x.member("slot-coverage", mixin, lex)
	if code == 3 {
		r.envFault(a.name, "slot-coverage", "env fault (exit 3)")
		return
	}
	lines := strings.Split(out, "\n")
	dups := 0
	for _, ln := range lines {
		if strings.HasPrefix(ln, "slot-coverage: FAIL dup-keys:") {
			dups++
		}
	}
	if dups > 0 {
		r.emit(a.name, failed, "dup-keys", itoa(dups))
	} else {
		r.emit(a.name, pass, "dup-keys", "0")
	}
	for _, ln := range lines {
		if rest, ok := strings.CutPrefix(ln, "slot-coverage: FAIL "); ok && !strings.HasPrefix(ln, "slot-coverage: FAIL dup-keys:") {
			r.emit(a.name, failed, "slot-coverage", rest)
		}
	}
	pct := cutBefore(strings.TrimPrefix(firstWithPrefix(lines, "pct="), "pct="), " ")
	if pct == "N/A" {
		r.emit(a.name, skip, "slot-coverage", "N/A (no types declared)")
		return
	}
	missing := strings.TrimPrefix(firstWithPrefix(lines, "missing="), "missing=")
	// [ "<pct up to its .>" -ge 100 ]: a value that is not an integer tests false (bash also printed an
	// "integer expression expected" diagnostic on stderr, which the port does not).
	if n, ok := bashInt(cutBefore(pct, ".")); ok && n >= 100 {
		r.emit(a.name, pass, "slot-coverage", pct+"%")
	} else if missing != "" {
		r.emit(a.name, warn, "slot-coverage", pct+"% (missing "+missing+")")
	} else {
		r.emit(a.name, warn, "slot-coverage", pct+"%")
	}
}

func firstWithPrefix(lines []string, p string) string {
	for _, ln := range lines {
		if strings.HasPrefix(ln, p) {
			return ln
		}
	}
	return ""
}

// bashInt parses s as the integer operand of bash's test: leading whitespace, an optional sign, decimal digits,
// trailing blanks; false for anything else and for an int64 overflow.
func bashInt(s string) (int64, bool) {
	t := strings.TrimRight(strings.TrimLeft(s, " \t\n\v\f\r"), " \t")
	n, err := strconv.ParseInt(t, 10, 64)
	return n, err == nil
}

// lintTimers is §3: lint-timers on src/ (no row when there is none), relaying only its PASS* and FAIL* lines
// with its own split: status, check, then `<path>:<line>  <message>` as `<basename>:<line>  <message>`.
func (r *report) lintTimers(a artifact) {
	if !isDir(a.dir + "/src") {
		return
	}
	out, code := r.x.member("lint-timers", a.dir+"/src")
	if code == 3 {
		r.envFault(a.name, "lint-timers", "env fault (exit 3)")
		return
	}
	for _, ln := range strings.Split(out, "\n") {
		if !strings.HasPrefix(ln, "PASS") && !strings.HasPrefix(ln, "FAIL") {
			continue
		}
		rest := cutAfter(ln, "  ")
		detail := cutAfter(rest, "  ")
		at := cutBefore(detail, "  ")
		path, line := at, at // ${site%:*} and ${site##*:}
		if i := strings.LastIndexByte(at, ':'); i >= 0 {
			path, line = at[:i], at[i+1:]
		}
		r.emit(a.name, cutBefore(ln, "  "), cutBefore(rest, "  "), baseName(path)+":"+line+"  "+cutAfter(detail, "  "))
	}
}

// baseName is basename(1) of a path: trailing slashes dropped, then the last component ("/" for all slashes,
// "" for ""). (A path starting with "-" was an option to the bash's basename; the lints never print one.)
func baseName(p string) string {
	t := strings.TrimRight(p, "/")
	if t == "" {
		if p == "" {
			return ""
		}
		return "/"
	}
	return t[strings.LastIndexByte(t, '/')+1:]
}

// plano is §4: verify-module --plano on src/rc/index.html when it is a file.
func (r *report) plano(a artifact) {
	html := a.dir + "/src/rc/index.html"
	if !isFile(html) {
		return
	}
	out, code := r.x.member("verify-module", "--plano", html)
	if code == 3 {
		r.envFault(a.name, "plano", "env fault (exit 3)")
		return
	}
	r.memberRows(a.name, out, filter{sel: "PASS|FAIL|SKIP|WARN", mode: ws})
}

// lexiconASCII is §4b: lint-lexicon-ascii on the artifact when a *.lexicon is within two levels of it; its FAIL*
// lines are relayed with the detail verbatim, else one PASS row.
func (r *report) lexiconASCII(a artifact) {
	if !hasLexicon(a.dir) {
		r.emit(a.name, skip, "lint-lexicon-ascii", "no *.lexicon under "+a.name)
		return
	}
	out, code := r.x.member("lint-lexicon-ascii", a.dir)
	if code == 3 {
		r.envFault(a.name, "lint-lexicon-ascii", "env fault (exit 3)")
		return
	}
	fail := false
	for _, ln := range strings.Split(out, "\n") {
		if strings.HasPrefix(ln, "FAIL") {
			rest := cutAfter(ln, "  ")
			r.emit(a.name, cutBefore(ln, "  "), cutBefore(rest, "  "), cutAfter(rest, "  "))
			fail = true
		}
	}
	switch {
	case !fail && code != 0:
		r.exitedNoFail(a.name, "lint-lexicon-ascii", code)
	case !fail:
		r.emit(a.name, pass, "lint-lexicon-ascii", "all lexicon files are ASCII-clean")
	}
}

// hasLexicon is `find <dir> -maxdepth 2 -name '*.lexicon' -print -quit`: dir itself, an entry of it or an entry
// of one of its (non-symlink) subdirectories is named *.lexicon; unreadable directories are skipped.
func hasLexicon(dir string) bool {
	if strings.HasSuffix(filepath.Base(dir), ".lexicon") {
		return true
	}
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".lexicon") {
			return true
		}
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		sub, _ := os.ReadDir(dir + "/" + e.Name())
		for _, s := range sub {
			if strings.HasSuffix(s.Name(), ".lexicon") {
				return true
			}
		}
	}
	return false
}

// linkTargetFlags is §5.5b: lint-link-target-flags [--wiring-map <map>] on src/. The chosen map is named in its
// Table 2 reason and in the PASS detail; a non-zero exit with no FAIL row is a FAIL row, never PASS.
func (r *report) linkTargetFlags(a artifact) {
	const check = "lint-link-target-flags"
	if !r.haveSrc(a, check) {
		return
	}
	var args []string
	if r.o.wiringMap != "" {
		args = []string{"--wiring-map", r.o.wiringMap}
	}
	out, code := r.x.member(check, append(args, a.dir+"/src")...)
	if code == 3 {
		r.envFault(a.name, check, "env fault (exit 3)")
		return
	}
	if r.o.wiringMap != "" {
		out = strings.ReplaceAll(out, "(wiring-map Table 2)", "(wiring-map Table 2: "+r.o.wiringMap+")")
	}
	rl := r.relayRows(a.name, out)
	switch {
	case rl.fail:
	case code != 0:
		r.emit(a.name, failed, check, fmt.Sprintf("exited %d with no FAIL row (crash or unexpected output) -- run lint-link-target-flags.sh manually", code))
	case r.o.wiringMap != "":
		r.emit(a.name, pass, check, "clean (wiring map: "+r.o.wiringMap+")")
	default:
		r.emit(a.name, pass, check, "clean (no wiring map)")
	}
}

// servlet is §5.18: lint-servlet on src/; an INFO line proves a servlet was scanned (PASS), no row and no INFO
// line is SKIP "no BWebServlet".
func (r *report) servlet(a artifact) {
	const check = "lint-servlet"
	if !r.haveSrc(a, check) {
		return
	}
	out, code := r.x.member(check, a.dir+"/src")
	if code == 3 {
		r.envFault(a.name, check, "env fault (exit 3)")
		return
	}
	rl := r.memberRows(a.name, out, failWarn)
	switch {
	case rl.rows == 0 && !strings.HasPrefix(out, "INFO") && !strings.Contains(out, "\nINFO"):
		r.emit(a.name, skip, check, "no BWebServlet")
	case !rl.fail:
		r.emit(a.name, pass, check, "clean")
	}
}

// rcScan is §5.19: rc-scan <artifact> [--profile P] [--legacy] when src/rc/ is a directory.
func (r *report) rcScan(a artifact) {
	if !isDir(a.dir + "/src/rc") {
		r.emit(a.name, skip, "rc-scan", "no src/rc/")
		return
	}
	args := []string{a.dir}
	if r.o.profile != "" {
		args = append(args, "--profile", r.o.profile)
	}
	if r.o.legacy {
		args = append(args, "--legacy")
	}
	out, code := r.x.member("rc-scan", args...)
	if code == 3 {
		r.envFault(a.name, "rc-scan", "env fault (exit 3)")
		return
	}
	r.passAfterRelay(a.name, "rc-scan", r.relayRows(a.name, out))
}

// vendorFloor is §5.19c: lint-vendor-floor on src/rc when src/rc/vendor/ is a directory: exit 0/1 relays, exit
// 4 (a tool unavailable) is a SKIP naming it, any other exit is an env fault.
func (r *report) vendorFloor(a artifact) {
	rc := a.dir + "/src/rc"
	if !isDir(rc + "/vendor") {
		return
	}
	out, code := r.x.member("lint-vendor-floor", rc)
	switch code {
	case 0, 1:
		r.passAfterRelay(a.name, "vendor-floor", r.relayRows(a.name, out))
	case 4:
		why := "unavailable"
		for _, ln := range strings.Split(out, "\n") { // sed 's/.*unavailable: /unavailable: /p' | head -1
			if i := strings.LastIndex(ln, "unavailable: "); i >= 0 {
				why = ln[i:]
				break
			}
		}
		r.emit(a.name, skip, "vendor-floor", why)
	default:
		r.envFault(a.name, "vendor-floor", fmt.Sprintf("env fault (exit %d)", code))
	}
}

// schemaRisk is §6: schema-risk <artifact>/.deploy-baseline <artifact>, its output discarded and its exit the
// verdict; no baseline is a floor SKIP unless --first-deploy.
func (r *report) schemaRisk(a artifact) {
	const check = "schema-risk"
	baseline := a.dir + "/.deploy-baseline"
	if !isDir(baseline) {
		if r.o.firstDeploy {
			r.emit(a.name, skip, check, "N/A (--first-deploy: never deployed, no baseline to diff)")
		} else {
			r.emit(a.name, skip, check, "no .deploy-baseline (floor check not run; extract the deployed baseline, build-verify.md § Schema-risk deploy workflow, or --first-deploy)")
			r.floorSkip = true
		}
		return
	}
	switch code := r.x.quiet(r.x.tb+"/schema-risk.sh", baseline, a.dir); code {
	case 0:
		r.emit(a.name, pass, check, "verdict=SAFE")
	case 1:
		r.emit(a.name, warn, check, "verdict=LOSSY")
	case 2:
		r.emit(a.name, failed, check, "verdict=OUTAGE") // D9a: OUTAGE is a finding, never an env fault
	case 3, 4:
		r.envFault(a.name, check, fmt.Sprintf("env fault (exit %d)", code))
	default:
		r.envFault(a.name, check, fmt.Sprintf("unexpected exit %d", code))
	}
}

// writePath is §11 (module root): its no-matrix exit 3 is a SKIP, any other exit 3 an ERROR naming the first
// output line; its FAIL rows are relayed with the STALE/DRIFT advisories as WARN, then PASS when no FAIL* row
// (a non-zero exit with none is a FAIL row).
func (r *report) writePath(a artifact) {
	const check = "lint-write-path"
	out, code := r.x.member(check, a.dir)
	switch {
	case code == 3 && strings.Contains(out, "no write-path-matrix.md found"):
		r.emit(a.name, skip, check, "no write-path-matrix.md")
	case code == 3:
		r.envFault(a.name, check, "env fault (exit 3): "+cutBefore(out, "\n"))
	default:
		switch rl := r.memberRows(a.name, out, filter{sel: "FAIL", asw: "STALE|DRIFT"}); {
		case rl.failP:
		case code != 0:
			r.exitedNoFail(a.name, check, code)
		default:
			r.emit(a.name, pass, check, "clean")
		}
	}
}

// triageConsole is the once-per-run triage-console --console-dir <dir> (a SKIP row without the flag): its
// non-empty output lines are printed as they are and counted by their leading status.
func (r *report) triageConsole() {
	if r.o.consoleDir == "" {
		fmt.Fprint(r.out, "(run)  SKIP  triage-console  no --console-dir\n")
		r.count(skip)
		return
	}
	out, code := r.x.member("triage-console", "--console-dir", r.o.consoleDir)
	if code == 3 {
		fmt.Fprint(r.out, "(run)  ERROR  triage-console  env fault (exit 3)\n")
		r.hadEnv = true
		return
	}
	printed := false
	for _, ln := range strings.Split(out, "\n") {
		if ln == "" {
			continue
		}
		fmt.Fprintln(r.out, ln)
		printed = true
		for _, st := range []string{failed, warn, pass, skip} {
			if strings.HasPrefix(ln, st) {
				r.count(st)
				break
			}
		}
	}
	if !printed {
		fmt.Fprint(r.out, "(run)  PASS  triage-console  0 rows\n")
		r.count(pass)
	}
}

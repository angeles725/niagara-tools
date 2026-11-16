package reportmodule

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// plano is §4: verify-module --plano on src/rc/index.html when it is a file.
func (r *report) plano(a artifact) {
	html := a.dir + "/src/rc/index.html"
	if !isFile(html) {
		return
	}
	out, code, ok := r.x.member("verify-module", "--plano", html)
	if r.truncated(a.name, "plano", "verify-module.sh", ok) {
		return
	}
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
	out, code, ok := r.x.member("lint-lexicon-ascii", a.dir)
	if r.truncated(a.name, "lint-lexicon-ascii", "lint-lexicon-ascii.sh", ok) {
		return
	}
	if code == 3 {
		r.envFault(a.name, "lint-lexicon-ascii", "env fault (exit 3)")
		return
	}
	fail := false
	for _, ln := range strings.Split(out, "\n") {
		if strings.HasPrefix(ln, "FAIL") {
			st, chk, detail := splitStatusCheck(ln)
			r.emit(a.name, st, chk, detail)
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
	out, code, ok := r.x.member(check, append(args, a.dir+"/src")...)
	if r.truncated(a.name, check, check+".sh", ok) {
		return
	}
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
	out, code, ok := r.x.member(check, a.dir+"/src")
	if r.truncated(a.name, check, check+".sh", ok) {
		return
	}
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
	out, code, ok := r.x.member("rc-scan", args...)
	if r.truncated(a.name, "rc-scan", "rc-scan.sh", ok) {
		return
	}
	if code == 3 {
		r.envFault(a.name, "rc-scan", "env fault (exit 3)")
		return
	}
	r.verdictAfterRelay(a.name, "rc-scan", r.relayRows(a.name, out))
}

// vendorFloor is §5.19c: lint-vendor-floor on src/rc when src/rc/vendor/ is for the lint to judge (toJudge): exit
// 0/1 relays, exit 4 (a tool unavailable) is a SKIP naming it, any other exit is an env fault.
func (r *report) vendorFloor(a artifact) {
	rc := a.dir + "/src/rc"
	if !toJudge(rc + "/vendor") {
		return
	}
	out, code, ok := r.x.member("lint-vendor-floor", rc)
	if r.truncated(a.name, "vendor-floor", "lint-vendor-floor.sh", ok) {
		return
	}
	switch code {
	case 0, 1:
		r.verdictAfterRelay(a.name, "vendor-floor", r.relayRows(a.name, out))
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

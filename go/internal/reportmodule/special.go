package reportmodule

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// The members whose rows the shared parser does not fully cover, in report order: §1-§3 here, §4-§5.19c in
// special_lints.go, §6, §11 and the run-wide triage in special_floor.go.

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
	out, code, ok := r.x.member("verify-module", append(args, "--src", r.o.root, jar)...)
	if r.truncated(a.name, "verify-module", "verify-module.sh", ok) {
		return
	}
	if code == 3 {
		r.envFault(a.name, "verify-module", "env fault (exit 3)")
		return
	}
	r.memberRows(a.name, out, filter{sel: "PASS|FAIL|SKIP|WARN", mode: ws})
}

// firstJar is the jar verify-module checks: among the entries of libs (a directory that is not a symlink, as
// `find <libs> -maxdepth 1 -name '*.jar'` reads it) named *.jar, the first in byte order that is not a
// -sources or -javadoc jar, else the first one; "" when there is none. (The bash took `| head -1` in directory
// order, so a -sources jar beside the module jar could be the one verified.)
func firstJar(libs string) string {
	if !isRealDir(libs) {
		return ""
	}
	entries, _ := os.ReadDir(libs) // byte order
	first := ""
	for _, e := range entries {
		n := e.Name()
		if !strings.HasSuffix(n, ".jar") {
			continue
		}
		if !strings.HasSuffix(n, "-sources.jar") && !strings.HasSuffix(n, "-javadoc.jar") {
			return libs + "/" + n
		}
		if first == "" {
			first = libs + "/" + n
		}
	}
	return first
}

// slotCoverage is §2: slot-coverage on module-include.xml and module.lexicon when both are files: the dup-keys
// count, every other "slot-coverage: FAIL" line, then the pct= coverage as PASS (>= 100), WARN or SKIP N/A;
// output with no pct= line is an env fault.
func (r *report) slotCoverage(a artifact) {
	mixin, lex := a.dir+"/module-include.xml", a.dir+"/module.lexicon"
	if !isFile(mixin) || !isFile(lex) {
		return
	}
	out, code, ok := r.x.member("slot-coverage", mixin, lex)
	if r.truncated(a.name, "slot-coverage", "slot-coverage.sh", ok) {
		return
	}
	if code == 3 {
		r.envFault(a.name, "slot-coverage", "env fault (exit 3)")
		return
	}
	lines := strings.Split(out, "\n")
	pctLine := firstWithPrefix(lines, "pct=")
	if pctLine == "" { // no coverage was computed: never a "%" WARN after a PASS dup-keys 0
		r.envFault(a.name, "slot-coverage", fmt.Sprintf("env fault (exit %d): no pct= line", code))
		return
	}
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
	pct := cutBefore(strings.TrimPrefix(pctLine, "pct="), " ")
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

// lintTimers is §3: lint-timers on src/ (no row when there is none), relaying its PASS*, FAIL* and WARN* lines
// with its own split: status, check, then `<path>:<line>  <message>` as `<basename>:<line>  <message>`.
func (r *report) lintTimers(a artifact) {
	if !isDir(a.dir + "/src") {
		return
	}
	out, code, ok := r.x.member("lint-timers", a.dir+"/src")
	if r.truncated(a.name, "lint-timers", "lint-timers.sh", ok) {
		return
	}
	if code == 3 {
		r.envFault(a.name, "lint-timers", "env fault (exit 3)")
		return
	}
	for _, ln := range strings.Split(out, "\n") {
		if !hasPrefixIn(ln, "PASS|FAIL|WARN") { // the rule's WARN rows too (job-cancel, jdk-thread, ...)
			continue
		}
		st, chk, detail := splitStatusCheck(ln)
		at := cutBefore(detail, "  ")
		path, line := at, at // ${site%:*} and ${site##*:}
		if i := strings.LastIndexByte(at, ':'); i >= 0 {
			path, line = at[:i], at[i+1:]
		}
		r.emit(a.name, st, chk, baseName(path)+":"+line+"  "+cutAfter(detail, "  "))
	}
}

// splitStatusCheck is the shell split `st=${l%%  *}; r=${l#*  }; chk=${r%%  *}; rest=${r#*  }` of a row line
// (a missing separator leaves the text whole at each step).
func splitStatusCheck(ln string) (st, check, rest string) {
	r := cutAfter(ln, "  ")
	return cutBefore(ln, "  "), cutBefore(r, "  "), cutAfter(r, "  ")
}

// baseName is basename(1) of a path: trailing slashes dropped, then the last component ("/" for all slashes,
// "" for ""). Declared waiver: the bash handed a path starting with "-" to basename(1) as an option (an error,
// an empty name); this cuts it as a path. No lint prints one: their paths start with the module root, and a root
// starting with "-" is a usage error.
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

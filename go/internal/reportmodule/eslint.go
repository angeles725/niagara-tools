package reportmodule

import (
	"fmt"
	"os"
	"strings"
)

const eslintHint = "(npm install --prefix toolbelt/eslint, or set KIT_ESLINT)"

// eslint is §5.19b: ESLint with the kit flat config on the artifact's own src/rc js. The binary is KIT_ESLINT,
// else toolbelt/eslint/node_modules/.bin/eslint, else an eslint on PATH whose --version (bounded by timeout(1)
// when it exists) reports major >= 9. No own js, no binary or an older PATH eslint is a SKIP row naming why;
// exit > 1 is an env fault and exit 1 with no FAIL row a FAIL row.
func (r *report) eslint(a artifact) {
	rc := a.dir + "/src/rc"
	if !isDir(rc) {
		return
	}
	ownJS := hasOwnJS(rc)
	bin := r.getenv("KIT_ESLINT")
	if local := r.x.tb + "/eslint/node_modules/.bin/eslint"; bin == "" && isExec(local) {
		bin = local
	}
	old := ""
	if bin == "" {
		bin = commandV(r.getenv("PATH"), "eslint")
		if bin != "" && ownJS {
			old = r.oldESLint(bin)
		}
	}
	switch {
	case !ownJS:
		r.emit(a.name, skip, "eslint", "no rc js")
	case !isExec(bin):
		r.emit(a.name, skip, "eslint", "unavailable: eslint not installed "+eslintHint)
	case old != "":
		r.emit(a.name, skip, "eslint", "unavailable: eslint on PATH is "+old+"; the kit flat config needs ESLint >= 9 "+eslintHint)
	default:
		r.runESLint(a, rc, bin)
	}
}

// oldESLint probes a PATH eslint's version: "" when its --version line names major >= 9, else that line (or
// "unknown version" when it printed none).
func (r *report) oldESLint(bin string) string {
	var ver string
	if to := commandV(r.getenv("PATH"), "timeout"); to != "" {
		limit := r.getenv("KIT_ESLINT_PROBE_TIMEOUT")
		if limit == "" {
			limit = "15"
		}
		ver = r.x.firstLine(to, limit, bin, "--version")
	} else {
		ver = r.x.firstLine(bin, "--version")
	}
	major := versionMajor(ver)
	if n, ok := bashInt(major); major != "" && (!ok || n >= 9) {
		return "" // an overflowing major failed bash's -lt test, so it was not "old" either
	}
	if ver == "" {
		return "unknown version"
	}
	return ver
}

// versionMajor is sed 's/^v\{0,1\}\([0-9][0-9]*\)\..*/\1/p': the digits after an optional leading v, when a
// "." follows them; "" otherwise.
func versionMajor(v string) string {
	s := strings.TrimPrefix(v, "v")
	i := 0
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
	}
	if i == 0 || i == len(s) || s[i] != '.' {
		return ""
	}
	return s[:i]
}

// runESLint runs `cd src/rc && eslint --config ... --format ... --no-warn-ignored .` and relays its rows. A
// binary named without a slash is looked up on PATH, as the shell did after the cd.
func (r *report) runESLint(a artifact, rc, bin string) {
	if !strings.Contains(bin, "/") {
		bin = commandV(r.getenv("PATH"), bin)
	}
	code, out, ok := 127, "", true
	if bin != "" {
		out, code, ok = r.x.run(rc, bin, "--config", r.x.tb+"/eslint.config.mjs",
			"--format", r.x.tb+"/eslint/rows-formatter.cjs", "--no-warn-ignored", ".")
	}
	if r.truncated(a.name, "eslint", "eslint", ok) {
		return
	}
	if code > 1 {
		r.envFault(a.name, "eslint", fmt.Sprintf("env fault (exit %d)", code))
		return
	}
	rl := r.relayRows(a.name, out)
	if code == 0 {
		r.passAfterRelay(a.name, "eslint", rl)
	}
	if !rl.fail && code == 1 {
		r.emit(a.name, failed, "eslint", "exit 1 with no parsable row")
	}
}

// hasOwnJS is `find <rc> \( -path '*/vendor' -o -path '*/ext' -o -name '.*' \) -prune -o -type f -name '*.js'
// ! -name '*.min.js' -print | head -1` being non-empty: a regular *.js file (not *.min.js) under rc outside
// any vendor/, ext/ or dot entry; symlinks are not followed and unreadable directories are skipped.
func hasOwnJS(dir string) bool {
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		n := e.Name()
		if n == "vendor" || n == "ext" || strings.HasPrefix(n, ".") {
			continue
		}
		switch {
		case e.Type().IsRegular() && strings.HasSuffix(n, ".js") && !strings.HasSuffix(n, ".min.js"):
			return true
		case e.IsDir() && hasOwnJS(dir+"/"+n):
			return true
		}
	}
	return false
}

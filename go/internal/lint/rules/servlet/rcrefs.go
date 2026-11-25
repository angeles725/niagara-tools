package servlet

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
	"github.com/angeles725/niagara-tools/go/internal/lint"
	"github.com/angeles725/niagara-tools/go/internal/rcscan"
	"github.com/angeles725/niagara-tools/go/internal/strerror"
)

const (
	msgCacheRC = "Cache-Control max-age>0 applies only to ?v= requests, but %s -- those requests are never cached; " +
		"add ?v=<module version>"
	maxClimb = 10 // directories above the servlet looked at for the rc/ directory
)

var servletName = regexp.MustCompile(`getServletName\s*\(\s*\)\s*\{\s*return\s*"([^"]*)"`)

// cacheRows are the cache-nofinger rows of a servlet file: every Cache-Control line with a max-age>0. One that
// is not gated on a fingerprinted request is the old row; either kind names the first rc/ reference to the
// servlet's static paths that carries no ?v= (rcEvidence) when there is one, and a gated line with such a
// reference WARNs as well, since the requests it sees never take the cached branch (issue #1035).
func cacheRows(path string, code []string, toks []javasrc.Token, decls []javasrc.Decl) ([]lint.Finding, error) {
	var out []lint.Finding
	var rcErr error
	evidence, loaded := "", false
	for i, ln := range code {
		if !strings.Contains(ln, "Cache-Control") || !maxAge.MatchString(ln) {
			continue
		}
		if !loaded {
			evidence, rcErr = rcEvidence(path, code)
			loaded = true
		}
		switch gated := fingerprinted(toks, decls, i); {
		case !gated && evidence == "":
			out = append(out, lint.Finding{Check: "cache-nofinger", Severity: lint.Unset, Line: i + 1, Msg: msgCache})
		case !gated:
			out = append(out, lint.Finding{Check: "cache-nofinger", Severity: lint.Unset, Line: i + 1, Msg: msgCache + "; " + evidence})
		case evidence != "":
			out = append(out, lint.Finding{Check: "cache-nofinger", Severity: lint.Unset, Line: i + 1, Msg: fmt.Sprintf(msgCacheRC, evidence)})
		}
	}
	return out, rcErr
}

// rcEvidence is `rc/index.html:3 references js/app.js without ?v=` (plus ` (and N more)`) for the first
// unfingerprinted reference to this servlet's static paths in the rc/ directory above path, or "". An rc/ page
// that cannot be read is an error (the engine prints it as the file's `cannot scan` line, exit 3, as it does an
// unreadable source file), never silence: the rows are then the ones the servlet alone gives.
func rcEvidence(path string, code []string) (string, error) {
	dir := rcDirAbove(filepath.Dir(path))
	if dir == "" {
		return "", nil
	}
	var mounts []string
	if m := servletName.FindStringSubmatch(strings.Join(code, "\n")); m != nil && m[1] != "" {
		mounts = append(mounts, strings.Trim(m[1], "/"))
	}
	all, err := rcscan.Unfingerprinted(dir, mounts)
	var refs []rcscan.RefFile
	for _, r := range all { // /module/ and /file/ are the station's FileServlet, not this servlet
		if !strings.HasPrefix(r.URL, "/module/") && !strings.HasPrefix(r.URL, "/file/") {
			refs = append(refs, r)
		}
	}
	if err != nil {
		return "", fmt.Errorf("rc/ pages could not be read: %s", strerror.Text(err))
	}
	if len(refs) == 0 {
		return "", nil
	}
	s := fmt.Sprintf("%s:%d references %s without ?v=", refs[0].File, refs[0].Line, refs[0].URL)
	if len(refs) > 1 {
		s += fmt.Sprintf(" (and %d more)", len(refs)-1)
	}
	return s, nil
}

// rcDirAbove is the rc/ directory (rc/ or src/rc/) of the nearest directory at or above dir that has one,
// looking at most maxClimb levels up and never past the root of the servlet's own module: a directory that
// holds a gradle project script or a module part (partRoot) is looked at for its own rc/ and ends the climb,
// so another module's rc/ above it is never read. "" for none.
func rcDirAbove(dir string) string {
	for n := 0; n <= maxClimb; n++ {
		for _, c := range []string{"rc", filepath.Join("src", "rc")} {
			if st, err := os.Stat(filepath.Join(dir, c)); err == nil && st.IsDir() {
				return filepath.Join(dir, c)
			}
		}
		if partRoot(dir) {
			return ""
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

// gradleScripts are the files that make a directory a gradle project root (Kotlin and Groovy DSL).
var gradleScripts = []string{"build.gradle.kts", "build.gradle", "settings.gradle.kts", "settings.gradle"}

// partRoot reports a directory that is a gradle project root or a module part the kit scaffolds: one holding
// a gradle script, a module-include.xml or a part build script (<part>.gradle.kts, <part>.gradle), or named
// <module>-rt or <module>-ux.
func partRoot(dir string) bool {
	if b := filepath.Base(dir); strings.HasSuffix(b, "-rt") || strings.HasSuffix(b, "-ux") {
		return true
	}
	for _, f := range append([]string{"module-include.xml"}, gradleScripts...) {
		if _, err := os.Stat(filepath.Join(dir, f)); err == nil {
			return true
		}
	}
	for _, g := range []string{"*.gradle.kts", "*.gradle"} {
		if m, _ := filepath.Glob(filepath.Join(dir, g)); len(m) > 0 {
			return true
		}
	}
	return false
}

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
func cacheRows(path string, code []string, toks []javasrc.Token, decls []javasrc.Decl) []lint.Finding {
	var out []lint.Finding
	evidence, loaded := "", false
	for i, ln := range code {
		if !strings.Contains(ln, "Cache-Control") || !maxAge.MatchString(ln) {
			continue
		}
		if !loaded {
			evidence, loaded = rcEvidence(path, code), true
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
	return out
}

// rcEvidence is `rc/index.html:3 references js/app.js without ?v=` (plus ` (and N more)`) for the first
// unfingerprinted reference to this servlet's static paths in the rc/ directory above path, or "".
func rcEvidence(path string, code []string) string {
	dir := rcDirAbove(filepath.Dir(path))
	if dir == "" {
		return ""
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
	if err != nil || len(refs) == 0 {
		return ""
	}
	s := fmt.Sprintf("%s:%d references %s without ?v=", refs[0].File, refs[0].Line, refs[0].URL)
	if len(refs) > 1 {
		s += fmt.Sprintf(" (and %d more)", len(refs)-1)
	}
	return s
}

// rcDirAbove is the rc/ directory (rc/ or src/rc/) of the nearest directory at or above dir that has one,
// looking at most maxClimb levels up and not past a gradle project root; "" for none.
func rcDirAbove(dir string) string {
	for n := 0; n <= maxClimb; n++ {
		for _, c := range []string{"rc", filepath.Join("src", "rc")} {
			if st, err := os.Stat(filepath.Join(dir, c)); err == nil && st.IsDir() {
				return filepath.Join(dir, c)
			}
		}
		for _, g := range []string{"build.gradle.kts", "settings.gradle.kts"} {
			if _, err := os.Stat(filepath.Join(dir, g)); err == nil {
				return ""
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

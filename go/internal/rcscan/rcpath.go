package rcscan

import (
	"fmt"
	"strings"
)

// jsFileSize (module map modes only) reports a .js file of more than maxJSLines lines at its line 1, unless
// line 1 carries the `rc-scan: allow js-file-size` marker.
func (s *fileScan) jsFileSize() {
	if !s.o.mapped() || thirdParty(s.src.rc) || s.src.ext != "js" || len(s.src.lines) <= maxJSLines ||
		allowedOn(s.src.lines[0], "js-file-size") {
		return
	}
	s.emitAt(1, s.o.sevMap(), "js-file-size", fmt.Sprintf("%d lines (> %d) -- split by concern into services/, store,"+
		" components/ and pages/ files (types/frontend-standard.md § Component layout)", len(s.src.lines), maxJSLines))
}

// thirdParty is a file the module does not author and so does not split, given its rc-relative path (rcRel): under
// vendor/ (vendored JS is not even scanned; its HTML is) or ext/ (pruned by the walk), or a *.min.js (excluded by
// the walk). The module-map checks state it themselves so a change to the walk cannot turn them on third-party code.
func thirdParty(rc string) bool {
	return strings.HasPrefix(rc, "vendor/") || strings.HasPrefix(rc, "ext/") || strings.HasSuffix(rc, ".min.js")
}

// rcRel is a row path (relative to the artifact dir) made relative to the module's rc/ directory: the part after
// its first `rc` segment, or the whole path when it has none (the artifact dir argument is the rc dir itself).
// Only the path below the artifact dir is read, so an ancestor directory named rc/vendor or rc/js/services cannot
// exempt or skip the module (review of 64cda653, R4/R3).
func rcRel(rel string) string {
	if strings.HasPrefix(rel, "rc/") {
		return rel[len("rc/"):]
	}
	if i := strings.Index(rel, "/rc/"); i >= 0 {
		return rel[i+len("/rc/"):]
	}
	return rel
}

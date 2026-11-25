package rcscan

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/lint"
)

// staticMsg is the static-url-nover row text (issue #1035): a browser keeps a static asset it fetched until its
// cache entry expires, so an rc/ reference that does not change with the module keeps serving the old file
// after a -ux-only deploy. The row is advisory (WARN under every mode, --strict included).
const staticMsg = "static-url-nover: %s has no ?v= query -- a browser keeps the cached file after a module update; " +
	"append ?v=<module version> (types/dashboard.md § cache-bust)"

// versionParams are the query parameters that fingerprint a URL (the set lint-servlet reads as a fingerprint).
var versionParams = map[string]bool{"v": true, "ver": true, "version": true}

// relPrefixes are the directories of an rc bundle a relative reference names: the static assets (the station
// serves /module/ and /file/ the same way).
var relPrefixes = []string{"img/", "css/", "js/"}

var scheme = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9+.-]*:`)

// StaticNoVersion reports whether the reference u names a static asset with no fingerprint: a path under
// /module/ or /file/, or a relative img/, css/, js/ (with or without a leading ./), or one of those relative
// directories under an extra mount (/<servlet>/img/...), whose query has none of v, ver or version at any
// position. An absolute URL (any scheme, //host), an anchor, an empty value and a template (${...}, {{...}}) are
// no static reference. An HTML entity (&amp;) is read as the character it stands for before the query is split.
func StaticNoVersion(u string, mounts []string) bool {
	u = strings.NewReplacer("&amp;", "&", "&#38;", "&").Replace(strings.TrimSpace(u))
	if u == "" || u[0] == '#' || strings.HasPrefix(u, "//") || scheme.MatchString(u) ||
		strings.Contains(u, "${") || strings.Contains(u, "{{") {
		return false
	}
	u, _, _ = strings.Cut(u, "#")
	path, query, _ := strings.Cut(u, "?")
	for _, kv := range strings.Split(query, "&") {
		if k, _, _ := strings.Cut(kv, "="); versionParams[k] {
			return false
		}
	}
	path = strings.TrimPrefix(path, "./")
	if strings.HasPrefix(path, "/module/") || strings.HasPrefix(path, "/file/") || hasRelPrefix(path) {
		return true
	}
	for _, m := range mounts {
		if rest, ok := strings.CutPrefix(path, "/"+m+"/"); ok && hasRelPrefix(rest) {
			return true
		}
	}
	return false
}

func hasRelPrefix(path string) bool {
	for _, p := range relPrefixes {
		if strings.HasPrefix(path, p) {
			return true
		}
	}
	return false
}

// waiver is the `rc-scan: allow static-url-nover <reason>` marker, in a comment of the file's syntax (//, /* */
// or <!-- -->) on the reference's line: the reason is the text after the id up to the comment's closer.
var waiver = regexp.MustCompile(`(?://|/\*|<!--).*rc-scan: allow static-url-nover(?:[ \t]+(.*))?$`)

// waived reports a waiver with a reason on line; a marker with no reason waives nothing.
func waived(line string) bool {
	m := waiver.FindStringSubmatch(line)
	if m == nil {
		return false
	}
	reason, _, _ := strings.Cut(m[1], "*/")
	reason, _, _ = strings.Cut(reason, "-->")
	return strings.TrimSpace(reason) != ""
}

// staticRows are the static-url-nover rows of one file: each reference that is static, unfingerprinted and
// unwaived, once per line and URL. Third-party code (vendor/, ext/, *.min.js) is never read: the module cannot
// change its references.
func staticRows(src *source) []lint.Finding {
	if thirdParty(src.rc) {
		return nil
	}
	var rows []lint.Finding
	seen := map[string]bool{}
	for _, r := range fileRefs(src.ext, src.lines) {
		key := fmt.Sprintf("%d %s", r.Line, r.URL)
		if seen[key] || !StaticNoVersion(r.URL, nil) || waived(src.lines[r.Line-1]) {
			continue
		}
		seen[key] = true
		rows = append(rows, lint.Finding{Severity: lint.Warn, Path: src.rel, Line: r.Line, Msg: fmt.Sprintf(staticMsg, r.URL)})
	}
	return rows
}

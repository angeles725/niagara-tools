// Package uxversion is the lint-ux-version rule: every local resource a -ux module's rc/ tree loads carries
// `?v=<defaultModuleVersion>` (types/dashboard.md, "Cache-bust every static asset with the module version";
// types/frontend-standard.md §5 and §11). The servlet caches images for 30 days (max-age=2592000), so a
// suffix that differs from the module version keeps a kiosk on the old file after a -ux-only deploy.
//
//	stale-version  FAIL  a reference whose ?v= differs from the group build's defaultModuleVersion
//	no-version     WARN  a local reference with no ?v= at all (FAIL under --strict)
//
// Three shapes are not judged against defaultModuleVersion (subtree.go, jsver.go, report.go):
// a content-hashed name (main-18a30d6b12.js) gets no row; a sub-tree declared by `<!-- ux-version: <dir>/ = <ver> -->`
// is judged against <ver> (subtree-version FAIL); a ?v= built in JavaScript ("a.html?v=" + NAME) is resolved from a
// var, let or const of the same file, else a dynamic-version WARN.
//
// A RootRule: it walks the module root's *-ux/src/rc/ itself and reads the group version (the shared
// vendorversion.SourceVersion reader) from the nearest build.gradle.kts at, under or above the root.
package uxversion

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"regexp"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/htmltext"
	"github.com/angeles725/niagara-tools/go/internal/lint"
)

// ID is the lint name printed in every row.
const ID = "lint-ux-version"

// The check ids, as printed in a row's message and named by an allow marker.
const (
	checkStale   = "stale-version"
	checkMissing = "no-version"
)

// Spec registers the rule: `[--strict] <module-root>` (--strict anywhere), an empty root is an ERROR row.
func Spec() lint.Spec {
	return lint.Spec{
		Root:        rule{},
		Mode:        lint.Anywhere,
		Usage:       "usage: lint-ux-version.sh [--strict] <module-root>",
		StrictFails: true,
		EmptyError:  lint.EmptyRow{Msg: "no *-ux/src/rc *.html, *.js or *.css files found (wrong path or no -ux module)"},
	}
}

type rule struct{}

func (rule) ID() string              { return ID }
func (rule) Severity() lint.Severity { return lint.Warn }

// selection is the files read: the module's own rc/ pages, scripts and styles. Third-party code (rc/vendor,
// rc/ext, *.min.js) is never read; srcTest/ and every part that is not a -ux part are outside the path.
var selection = lint.Selection{
	Names:      []string{"*.html", "*.htm", "*.js", "*.css"},
	Except:     []string{"*.min.js"},
	Paths:      []string{"*-ux/src/rc/*"},
	PrunePaths: []string{"*/rc/vendor", "*/rc/ext"},
	Prune:      []string{"node_modules"},
}

// CheckRoot reads every selected file once and reports its stale and missing suffixes, file by file in walk
// order and by line within a file.
func (rule) CheckRoot(root string) ([]lint.Finding, error) {
	files, err := lint.Walk(root, selection)
	if err != nil {
		return nil, errors.New("cannot list every file under " + root + ": " + err.Error())
	}
	if len(files) == 0 {
		return nil, lint.ErrEmptyRoot
	}
	want, err := groupVersion(root)
	if err != nil {
		return nil, err
	}
	if want == "" {
		// Mutation: UXV-noversion -- comparing against an empty version passes every module that declares none
		return nil, errors.New(`no defaultModuleVersion("X.Y.Z") found in a build.gradle.kts under or above ` + root)
	}
	var srcs []source
	var rerr error
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			var pe *fs.PathError
			if errors.As(err, &pe) {
				err = pe.Err
			}
			rerr = fmt.Errorf("cannot scan %s: %s", f, err)
			break
		}
		srcs = append(srcs, source{f, strings.Split(htmltext.Decode(b), "\n")})
	}
	subs, out := declarations(srcs)
	for _, s := range srcs {
		out = append(out, scan(s.path, s.lines, want, subs)...)
	}
	return out, rerr
}

// scheme starts an absolute URL (http:, https:, data:, blob:, mailto:, javascript:, ...).
var scheme = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9+.-]*:`)

// verdict is the check id that reference ref fails against the module version want, and the version it
// carries when it is stale; "" when ref is fine or is not a local resource this lint reads.
func verdict(ref, want string) (id, got string) {
	// Mutation: UXV-external -- reading an external URL as local WARNs every CDN reference
	if ref == "" || strings.HasPrefix(ref, "#") || strings.HasPrefix(ref, "//") || scheme.MatchString(ref) {
		return "", ""
	}
	if strings.Contains(ref, "${") || strings.Contains(ref, "{{") { // a template: the value is not known here
		return "", ""
	}
	path, _, _ := strings.Cut(ref, "#")
	_, query, _ := strings.Cut(path, "?")
	for _, kv := range strings.Split(query, "&") {
		if v, ok := strings.CutPrefix(kv, "v="); ok {
			// an empty value is the version appended at run time ("a.js?v=" + VERSION): not decidable here
			if v == "" || v == want {
				return "", ""
			}
			// Mutation: UXV-stale -- not comparing the suffix with the module version passes a stale ?v=
			return checkStale, v
		}
	}
	// Mutation: UXV-missing -- dropping this row passes a local reference with no ?v= at all
	return checkMissing, ""
}

// waiverAt is the reason of an allow marker for check id on the line of r (or on the line its tag starts).
func waiverAt(lines []string, r ref, id string) string {
	for _, n := range []int{r.line, r.tag} {
		if n >= 1 && n <= len(lines) {
			if why := waiver(lines[n-1], id); why != "" {
				return why
			}
		}
	}
	return ""
}

// waiver is the reason of the allow marker `lint-ux-version: allow <id> <reason>` on line, "" when the line
// has none for id or gives no reason (a waiver records why). A comment closer ends the reason.
func waiver(line, id string) string {
	marker := "lint-ux-version: allow " + id
	i := strings.Index(line, marker)
	if i < 0 {
		return ""
	}
	rest := line[i+len(marker):]
	if rest != "" && rest[0] != ' ' && rest[0] != '\t' { // allow stale-versions is another word
		return ""
	}
	rest = strings.TrimSpace(rest)
	for _, closer := range []string{"-->", "*/"} {
		rest = strings.TrimSpace(strings.TrimSuffix(rest, closer))
	}
	return rest
}

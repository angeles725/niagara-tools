package rcscan

import (
	"os"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/lint"
)

// walkRC is the files a reader of one rc/ directory takes: its pages, scripts and styles, not the third-party
// code under vendor/ and ext/ nor a *.min.js.
var walkRC = lint.Selection{Names: []string{"*.html", "*.js", "*.css"}, Except: []string{"*.min.js"},
	Prune: []string{"vendor", "ext"}}

// RefFile is an Unfingerprinted reference: where it is (File is relative to the directory holding rc/, so it
// reads rc/index.html) and the URL as written.
type RefFile struct {
	File string
	Line int
	URL  string
}

// Unfingerprinted lists, in walk order, the static references under the rc/ directory rcDir that
// StaticNoVersion reports for the extra mounts and no waiver excuses: what lint-servlet cache-nofinger holds
// against a servlet that sets max-age>0 (issue #1035). A file it cannot read is an error.
func Unfingerprinted(rcDir string, mounts []string) ([]RefFile, error) {
	files, err := lint.Walk(rcDir, walkRC)
	if err != nil {
		return nil, err
	}
	var out []RefFile
	for _, p := range files {
		data, err := os.ReadFile(p)
		if err != nil {
			return nil, err
		}
		lines := records(data)
		for _, r := range fileRefs(p[strings.LastIndexByte(p, '.')+1:], lines) {
			if StaticNoVersion(r.URL, mounts) && !waived(lines[r.Line-1]) {
				out = append(out, RefFile{File: "rc/" + strings.TrimPrefix(p, strings.TrimSuffix(rcDir, "/")+"/"),
					Line: r.Line, URL: r.URL})
			}
		}
	}
	return out, nil
}

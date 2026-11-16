package modulemap

import (
	"path"
	"sort"
	"strconv"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/lint"
)

// mm4 checks every source file of the parts against the concerns that list it, in path order.
func (s *scan) mm4(rows []row, owners map[string][]int) {
	var srcs []string
	for _, f := range s.files {
		if isSource(f) {
			srcs = append(srcs, f)
		}
	}
	sort.Strings(srcs)
	for _, f := range srcs {
		switch o := owners[f]; {
		case len(o) == 0:
			s.add(s.strictSev(), s.abs(f), 0, "MM4: in no concern (list it in the Files of the concern that owns it)")
		case len(o) > 1:
			s.add(lint.Fail, s.abs(f), 0, "MM4: in "+strconv.Itoa(len(o))+" concerns ("+strings.Join(concerns(rows, o), ", ")+"); list it in one")
		}
	}
}

// isSource: a part's src/**/*.java, or a src/rc/**/*.{js,html,css} outside vendor/ and ext/ and not *.min.*.
func isSource(rel string) bool {
	seg := strings.Split(rel, "/")
	if len(seg) < 3 || seg[1] != "src" {
		return false
	}
	name := seg[len(seg)-1]
	if strings.HasSuffix(name, ".java") {
		return true
	}
	if len(seg) < 4 || seg[2] != "rc" || strings.Contains(name, ".min.") { // <part>/src/rc itself, a file, is no source
		return false
	}
	for _, d := range seg[3 : len(seg)-1] {
		if d == "vendor" || d == "ext" {
			return false
		}
	}
	switch path.Ext(name) {
	case ".js", ".html", ".css":
		return true
	}
	return false
}

// mm6 WARNs every srcTest Java file no row lists by its path, in path order.
func (s *scan) mm6(listed map[string]bool) {
	var tests []string
	for _, f := range s.files {
		if inSrcTest(f) && strings.HasSuffix(f, ".java") && !listed[f] {
			tests = append(tests, f)
		}
	}
	sort.Strings(tests)
	for _, f := range tests {
		s.add(lint.Warn, s.abs(f), 0, "MM6: test in no concern (list it in the Tests of the concern it covers)")
	}
}

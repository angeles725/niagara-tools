package modulemap

import (
	"path"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/lint"
)

// entries checks one cell of r (MM3) and returns how many of its entries name an existing file under a srcTest/
// directory (MM5 counts them for the Tests cell) and how many name an existing SPA test, a webtest/**/*.test.mjs
// file (MM5 counts those only for a row whose Files are under src/rc/). Exact entries go into listed; with owners
// set (the Files cell), every source file an entry lists records r's idx, once per row.
func (s *scan) entries(r row, cell []string, owners map[string][]int, listed map[string]bool) (tests, spa int) {
	mine := map[string]bool{}
	own := func(f string) {
		if owners != nil && !mine[f] {
			mine[f] = true
			owners[f] = append(owners[f], r.idx)
		}
	}
	for _, e := range cell {
		where := e + " (concern " + r.concern + ")"
		clean := path.Clean(e)
		if path.IsAbs(e) || clean == ".." || strings.HasPrefix(clean, "../") {
			s.add(lint.Fail, s.mapPath, r.line, "MM3: "+where+" is not a relative path inside the module root")
			continue
		}
		if strings.ContainsAny(e, "*?[") {
			n := 0
			for _, f := range s.files {
				if !strings.HasSuffix(f, ".java") && matchGlob(clean, f) {
					n++
					own(f)
					tests, spa = count(f, tests, spa)
				}
			}
			if n == 0 {
				s.add(lint.Fail, s.mapPath, r.line, "MM3: "+where+" matches no file (a glob never lists a .java file)")
			}
			continue
		}
		switch {
		case lint.IsFile(s.abs(clean)):
			listed[clean] = true
			own(clean)
			tests, spa = count(clean, tests, spa)
		case lint.IsDir(s.abs(clean)):
			s.add(lint.Fail, s.mapPath, r.line, "MM3: "+where+" is a directory: list its files, or use a src/rc glob")
		default:
			s.add(lint.Fail, s.mapPath, r.line, "MM3: "+where+" does not exist")
		}
	}
	return tests, spa
}

// inSrcTest: rel lies under a srcTest/ directory of a part.
func inSrcTest(rel string) bool { return strings.Contains("/"+rel, "/srcTest/") }

// count adds rel to the srcTest count or the SPA test count.
func count(rel string, tests, spa int) (int, int) {
	if inSrcTest(rel) {
		tests++
	}
	if inWebTest(rel) {
		spa++
	}
	return tests, spa
}

// inWebTest: rel is an SPA test, a *.test.mjs file under a webtest/ directory of a part.
func inWebTest(rel string) bool {
	return strings.HasSuffix(rel, ".test.mjs") && strings.Contains("/"+rel, "/webtest/")
}

// underRC: some entry of the Files cell lies under a src/rc/ directory (a row of SPA files).
func underRC(files []string) bool {
	for _, e := range files {
		if strings.Contains("/"+path.Clean(e), "/src/rc/") {
			return true
		}
	}
	return false
}

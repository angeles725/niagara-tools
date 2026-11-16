package modulemap

import (
	"path"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/lint"
)

// entries checks one cell of r (MM3) and returns how many of its entries name an existing file under a srcTest/
// directory (MM5 counts them for the Tests cell). Exact entries go into listed; with owners set (the Files
// cell), every source file an entry lists records r's idx, once per row.
func (s *scan) entries(r row, cell []string, owners map[string][]int, listed map[string]bool) int {
	tests := 0
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
					if inSrcTest(f) {
						tests++
					}
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
			if inSrcTest(clean) {
				tests++
			}
		case lint.IsDir(s.abs(clean)):
			s.add(lint.Fail, s.mapPath, r.line, "MM3: "+where+" is a directory: list its files, or use a src/rc glob")
		default:
			s.add(lint.Fail, s.mapPath, r.line, "MM3: "+where+" does not exist")
		}
	}
	return tests
}

// inSrcTest: rel lies under a srcTest/ directory of a part.
func inSrcTest(rel string) bool { return strings.Contains("/"+rel, "/srcTest/") }

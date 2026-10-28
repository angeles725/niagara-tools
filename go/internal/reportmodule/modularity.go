package reportmodule

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
	"github.com/angeles725/niagara-tools/go/internal/lint"
	"github.com/angeles725/niagara-tools/go/internal/lint/rules/modulemap"
	"github.com/angeles725/niagara-tools/go/internal/lint/rules/size"
)

// modularity is the per-module summary row (modularity-gaps Δ8), an INFO row the summary does not count:
// `(module)  INFO  modularity  mode=<m> concerns=<n> unlisted=<n> largest=<path> (<n> hand-written lines)`.
// The counts come from the module map (modulemap.Summarize: `-` with no valid map, `?` when the map or a part
// could not be read; lint-module-map's own rows report that fault); the largest class is the part source
// (`<part>/src/**/*.java`) with the most hand-written lines (size.HandLines), the first in path order on a tie,
// its path relative to the module root (`-` with none). The row only orients: it never changes the verdict.
func (r *report) modularity(a artifact) {
	sum, err := modulemap.Summarize(a.dir)
	concerns, unlisted := count(sum.Concerns), count(sum.Unlisted)
	if err != nil {
		concerns, unlisted = "?", "?"
	}
	largest := "-"
	if p, n := largestClass(a.dir); p != "" {
		largest = fmt.Sprintf("%s (%d hand-written lines)", p, n)
	}
	r.emit(a.name, "INFO", "modularity", fmt.Sprintf("mode=%s concerns=%s unlisted=%s largest=%s",
		modeName(sum.Mode), concerns, unlisted, largest))
}

func count(n int) string {
	if n < 0 {
		return "-"
	}
	return itoa(n)
}

// modeName is the mode as the map header spells it, "none" with no valid map.
func modeName(m lint.Modularity) string {
	switch m {
	case lint.ModularityStrict:
		return "strict"
	case lint.ModularityLegacy:
		return "legacy"
	}
	return "none"
}

// largestClass walks every -rt/-ux/-wb/-se part's src/ (dot directories and symlinked directories skipped, an
// unreadable entry skipped) and returns the *.java with the most hand-written lines, relative to root.
func largestClass(root string) (string, int) {
	var files []string
	entries, _ := os.ReadDir(root)
	for _, e := range entries {
		if !isPart(e.Name()) || !isRealDir(filepath.Join(root, e.Name())) {
			continue
		}
		src := filepath.Join(root, e.Name(), "src")
		_ = filepath.WalkDir(src, func(p string, d os.DirEntry, err error) error {
			switch {
			case err != nil:
				return nil
			case d.IsDir() && p != src && strings.HasPrefix(d.Name(), "."):
				return filepath.SkipDir
			case d.Type().IsRegular() && strings.HasSuffix(p, ".java"):
				files = append(files, p)
			}
			return nil
		})
	}
	sort.Strings(files)
	best, most := "", -1
	for _, p := range files {
		b, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		lines, err := javasrc.Lines(strings.NewReader(string(b)))
		if err != nil {
			continue
		}
		if n := size.HandLines(javasrc.NewFile(p, lines)); n > most {
			best, most = p, n
		}
	}
	if best == "" {
		return "", 0
	}
	rel, _ := filepath.Rel(root, best)
	return rel, most
}

// isPart: a module part directory name (<MOD>-rt, -ux, -wb or -se).
func isPart(n string) bool {
	for _, suf := range []string{"-rt", "-ux", "-wb", "-se"} {
		if strings.HasSuffix(n, suf) && len(n) > len(suf) {
			return true
		}
	}
	return false
}

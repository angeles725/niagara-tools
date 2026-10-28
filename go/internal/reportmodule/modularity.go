package reportmodule

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
	"github.com/angeles725/niagara-tools/go/internal/lint"
	"github.com/angeles725/niagara-tools/go/internal/lint/rules/modulemap"
	"github.com/angeles725/niagara-tools/go/internal/lint/rules/size"
)

// modularity is the per-module summary row (modularity-gaps Δ8), an INFO row the summary does not count:
// `(module)  INFO  modularity  mode=<m> concerns=<n> unlisted=<n> largest=<path> (<n> hand-written lines)`.
// The mode is lint.ModuleMode's, the source the per-file rules pick their severity from (the header's mode even
// when the map's table is not valid). The counts come from the module root's map (modulemap.Summarize: `-` with
// no valid map there). The largest class is the part source (`<part>/src/**/*.java`, the file set lint-module-map
// walks: modulemap.PartJava) with the most hand-written lines (size.HandLines), the first in path order on a tie,
// relative to the module root (`-` with none, no part included). With a map at the root, every value but the
// mode is `?` when the root has no part or the map or a part could not be read; with none, a part that could not
// be read makes the largest class `?`. lint-module-map's own rows report the fault. The row only orients: it never
// changes the verdict.
func (r *report) modularity(a artifact) {
	sum, err := modulemap.Summarize(a.dir)
	concerns, unlisted, largest := count(sum.Concerns), count(sum.Unlisted), "-"
	if err != nil {
		concerns, unlisted, largest = "?", "?", "?"
	} else if p, n, err := largestClass(a.dir); err != nil && !errors.Is(err, lint.ErrEmptyRoot) {
		largest = "?"
	} else if p != "" {
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

// largestClass returns the part Java source (modulemap.PartJava, in path order) with the most hand-written lines,
// relative to root; a file that cannot be read or tokenized is skipped. An error is PartJava's.
func largestClass(root string) (string, int, error) {
	files, err := modulemap.PartJava(root)
	if err != nil {
		return "", 0, err
	}
	best, most := "", -1
	for _, rel := range files {
		p := filepath.Join(root, rel)
		b, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		lines, err := javasrc.Lines(strings.NewReader(string(b)))
		if err != nil {
			continue
		}
		if n := size.HandLines(javasrc.NewFile(p, lines)); n > most {
			best, most = rel, n
		}
	}
	return best, max(most, 0), nil
}

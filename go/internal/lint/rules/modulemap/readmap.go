package modulemap

import (
	"errors"
	"os"
	"strconv"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/lint"
)

// Row is one concern row of a map as the module-map tool reads it (modularity-structure Δ1, Δ4).
type Row struct {
	Line                int // 1-based line of the row in MODULE-MAP.md
	Concern, Part, When string
	Files, Tests        []string // the cell entries as written (globs unexpanded)
}

// Map is a module root's parsed MODULE-MAP.md: its mode, its rows and its lines (for a writer that inserts a
// row after Rows' last line).
type Map struct {
	Mode  lint.Modularity
	Rows  []Row
	Lines []string
}

// ReadMap parses root's MODULE-MAP.md with the lint's own header and table reader. A missing or unreadable map is
// an error; an invalid header or table is an error naming its MM2 problem and line.
func ReadMap(root string) (Map, error) {
	p := strings.TrimRight(root, "/") + "/" + lint.MapFile
	b, err := os.ReadFile(p)
	if err != nil {
		return Map{}, errors.New("cannot read " + p + ": " + err.Error())
	}
	lines := strings.Split(string(b), "\n")
	mode, at, problem := lint.MapHeader(lines)
	if problem == "" {
		var rows []row
		rows, at, problem = table(lines)
		if problem == "" {
			m := Map{Mode: mode, Lines: lines}
			for _, r := range rows {
				m.Rows = append(m.Rows, Row{Line: r.line, Concern: r.concern, Part: r.part, When: r.when, Files: r.files, Tests: r.tests})
			}
			return m, nil
		}
	}
	return Map{}, errors.New(p + ":" + strconv.Itoa(at) + ": MM2: " + problem)
}

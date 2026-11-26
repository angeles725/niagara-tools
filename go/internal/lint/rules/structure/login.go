package structure

import (
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
	"github.com/angeles725/niagara-tools/go/internal/lint"
)

// Superclasses L18 looks for: the classes it reads are direct subclasses, as the dashboard scaffold emits them.
const (
	loginBase   = "BLoginTemplate"
	servletBase = "BWebServlet"
)

// loginRow is one BLoginTemplate subclass: the file relative to the module root and its 1-based line.
type loginRow struct {
	file string
	line int
}

// logins is the L18 reading of one profile: whether a class in it extends BWebServlet, and every subclass of
// BLoginTemplate. Comments and literals are blanked before the classes are read.
func (s *scan) logins(dir string) (servlet bool, rows []loginRow) {
	if !lint.IsDir(dir + "/src") {
		return false, nil
	}
	for _, f := range s.files(dir+"/src", false, "*.java") {
		raw := s.lines(f)
		file := javasrc.NewFile(f, raw)
		for _, c := range javasrc.Classes(javasrc.Tokens(file.Stripped())) {
			switch c.Extends {
			case servletBase:
				servlet = true
			case loginBase:
				rows = append(rows, loginRow{s.rel(f), c.Line + 1})
			}
		}
	}
	return servlet, rows
}

// loginPlacement is L18 over every profile: a class extending BLoginTemplate must stand in a dashboard -rt, the
// part that holds a BWebServlet itself or whose <module>-ux sibling does (types/dashboard.md § The module ships its
// own login). A login in any other part is the retired separate login module. A dashboard with a servlet and no
// login template is not flagged: the stock login stays a supported state (applyLogin false, an empty slot).
func (s *scan) loginPlacement(profiles []string) {
	type read struct {
		servlet bool
		rows    []loginRow
	}
	parts := map[string]read{}
	for _, p := range profiles {
		servlet, rows := s.logins(p)
		parts[p[strings.LastIndexByte(p, '/')+1:]] = read{servlet, rows}
	}
	for _, p := range profiles {
		name := p[strings.LastIndexByte(p, '/')+1:]
		r := parts[name]
		if len(r.rows) == 0 {
			continue
		}
		module, isRt := strings.CutSuffix(name, "-rt")
		if isRt && (r.servlet || parts[module+"-ux"].servlet) {
			continue
		}
		for _, row := range r.rows {
			s.out = append(s.out, lint.Finding{Path: row.file, Line: row.line, Msg: "L18: login outside the dashboard module (" + loginBase +
				" subclass in " + name + ", which is not a dashboard -rt: a dashboard ships its login in its own -rt, next to the -ux that serves the " +
				servletBase + "; templates/dashboard-login, types/dashboard.md § The module ships its own login)"})
		}
	}
}

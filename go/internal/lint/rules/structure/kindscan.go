package structure

import (
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/lint"
)

// declaredKind is the module's declared kind: the value of the first "type:" line of the header of the module
// root's MODULE-MAP.md (the lines before its first table row, outside ``` fences, as lint.MapHeader reads the
// modularity line; a trailing "# comment" is dropped). No map or no such line is "": no kind, no exemption.
func (s *scan) declaredKind() string {
	m := s.root + "/" + lint.MapFile
	if !lint.IsFile(m) {
		return ""
	}
	fenced := false
	for _, raw := range s.lines(m) {
		l := strings.TrimSpace(raw)
		switch {
		case strings.HasPrefix(l, "```"):
			fenced = !fenced
		case fenced:
		case strings.HasPrefix(l, "|"):
			return ""
		case strings.HasPrefix(l, kindKey):
			v, _, _ := strings.Cut(strings.TrimPrefix(l, kindKey), "#")
			return strings.TrimSpace(v)
		}
	}
	return ""
}

// rtPalette is L5 on one -rt part: a palette with at least one named entry. A module declared `type: library`
// may ship no palette (it is depended on, never dragged); a palette it does ship is judged like any other.
func (s *scan) rtPalette(rel, pal string, named bool) {
	switch {
	case !lint.IsFile(pal):
		if s.kind != kindLibrary {
			s.row(lint.Unset, rel, l5Missing)
		}
	case !named:
		s.row(lint.Unset, s.rel(pal), l5Empty)
	}
}

// emptyPart is L9 on one -wb or -ux part with no Java source and no named palette entry. The -ux part of a
// module declared `type: theme` passes when it holds a style sheet; undeclared, it FAILs as before, and the
// row says how to declare a theme.
func (s *scan) emptyPart(dir, name, rel string) {
	switch ux := strings.HasSuffix(name, "-ux"); {
	case !ux:
		s.row(lint.Unset, rel, l9Empty+")")
	case s.kind != kindTheme:
		s.row(lint.Unset, rel, l9Empty+l9ThemeHint)
	case len(s.files(dir, true, themeSheet)) == 0:
		s.row(lint.Unset, rel, l9Theme)
	}
}

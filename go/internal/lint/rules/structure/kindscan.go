package structure

import (
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/lint"
)

// declaredKind is the module's declared kind: the value of the first "type:" line of the header of the module
// root's MODULE-MAP.md, read by lint.ReadMapHeader as the modularity line is (the lines before its first table
// row, outside ``` fences; a trailing "# comment" is dropped). No map or no such line is "": no kind, no exemption.
func (s *scan) declaredKind() string {
	m := s.root + "/" + lint.MapFile
	if !lint.IsFile(m) {
		return ""
	}
	return lint.ReadMapHeader(s.lines(m)).Kind
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

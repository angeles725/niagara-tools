package structure

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/lint"
	"github.com/angeles725/niagara-tools/go/internal/xmlsrc"
)

// typeName reads the name of a module-include.xml <type> element.
var typeName = regexp.MustCompile(`<type[[:space:]][^>]*\bname="([^"]+)"`)

// declaredTypes is every type name the module's profiles declare in their module-include.xml (L17 reads it).
func (s *scan) declaredTypes(profiles []string) map[string]bool {
	out := map[string]bool{}
	for _, p := range profiles {
		inc := p + "/module-include.xml"
		if !lint.IsFile(inc) {
			continue
		}
		for _, ln := range s.lines(inc) {
			for _, m := range typeName.FindAllStringSubmatch(ln, -1) {
				out[m[1]] = true
			}
		}
	}
	return out
}

// palette is L17 over one module.palette, read through the shared XML scanner (comments are not elements): a t= that
// is a dotted Java class name FAILs ("Invalid typespec"), and so does <symbol>:B<Type> when the symbol resolves to
// THIS module (its latest m=, as the decoder keeps it, names own, the profile directory's module) and the module
// declares <Type> and not B<Type> ("Type not found"). A symbol of another module, or one the palette never maps, is
// not judged: vendor types can carry the B in their name (jsonToolkit declares a type named BFormatString). An m= that maps a symbol to a module
// other than its first m= WARNs (the decoder keeps the last mapping, so every later entry resolves in the new module;
// an identical repeat changes nothing and is clean). A palette that is not well-formed XML is one WARN, unchecked (none when it holds no "<p": L5 reports that).
// [ev: corpus B1418 §1418.5 e] [ev: corpus B1419 §1419.4 d] [ev: retro course-coverage Δ20]
func (s *scan) palette(pal, own string) {
	rel := s.rel(pal)
	doc, err := xmlsrc.Open(pal)
	if err != nil {
		if countText(s.lines(pal), "<p") == 0 {
			return // no <p> to check: L5 already reports a palette without entries
		}
		why := err.Error()
		if e, ok := err.(*xmlsrc.Error); ok {
			why = strconv.Itoa(e.Line) + ":" + strconv.Itoa(e.Col) + ": " + e.Msg
		}
		s.row(lint.Warn, rel, "L17: module.palette is not well-formed XML ("+why+") — palette type specs unchecked")
		return
	}
	type decl struct {
		module string
		line   int
	}
	first := map[string]decl{}
	current := map[string]string{} // symbol -> module of its latest m= (ValueDocDecoder keeps one per symbol)
	at := func(sev lint.Severity, line int, msg string) {
		s.out = append(s.out, lint.Finding{Severity: sev, Path: rel, Line: line, Msg: "L17: module.palette " + msg})
	}
	for _, e := range doc.Elements() {
		if e.Name != "p" {
			continue
		}
		if m, ok := e.Attr("m"); ok {
			if sym, mod, ok := strings.Cut(m, "="); ok {
				current[sym] = mod
				if d, seen := first[sym]; !seen {
					first[sym] = decl{mod, e.Line}
				} else if d.module != mod {
					at(lint.Warn, e.Line, `m="`+m+`" redeclares symbol `+sym+` (first `+sym+`=`+d.module+` at line `+
						strconv.Itoa(d.line)+`): every later t="`+sym+`:..." resolves in `+mod)
				}
			}
		}
		t, ok := e.Attr("t")
		if !ok {
			continue
		}
		sym, ty, hasSym := strings.Cut(t, ":")
		switch {
		case strings.Contains(t, "."):
			at(lint.Unset, e.Line, `t="`+t+`" is a Java class name, not <symbol>:<Type> (Workbench: "Invalid typespec")`)
		case !hasSym || s.types[ty]:
		case strings.HasPrefix(ty, "B") && s.types[ty[1:]] && current[sym] == own:
			at(lint.Unset, e.Line, `t="`+t+`" keeps the class's B prefix: the type is `+sym+`:`+ty[1:]+` (Workbench: "Type not found")`)
		}
	}
}

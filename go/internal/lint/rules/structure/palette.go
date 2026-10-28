package structure

import (
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
	"github.com/angeles725/niagara-tools/go/internal/lint"
	"github.com/angeles725/niagara-tools/go/internal/xmlsrc"
)

// gradleModuleName reads a gradle file's moduleName.set("X") or moduleName = "X" (lint-logger-name reads the same).
var gradleModuleName = regexp.MustCompile(`\bmoduleName\s*(?:\.set\(\s*|=\s*)"([^"]+)"`)

// ownModule is the module a profile directory belongs to, as lint-logger-name resolves it: the moduleName a gradle
// file (*.gradle.kts or *.gradle, in name order) in the profile directory declares, else one in the module root above
// it, else the directory name without its last "-<part>" suffix. A commented-out declaration is not read. The
// directory prefix need not be the module name (a renamed module keeps its old part directories).
// [ev: retro course-coverage Δ20]
func (s *scan) ownModule(dir string) string {
	for _, d := range []string{dir, s.root} {
		entries, err := os.ReadDir(d)
		if err != nil {
			continue
		}
		var names []string
		for _, e := range entries {
			if n := e.Name(); !e.IsDir() && (strings.HasSuffix(n, ".gradle.kts") || strings.HasSuffix(n, ".gradle")) {
				names = append(names, n)
			}
		}
		sort.Strings(names)
		for _, n := range names {
			// Mutation: L17-commented-modulename -- reading raw lines lets a commented-out old moduleName win
			code, _ := javasrc.Strip(s.lines(d + "/" + n)) // Kotlin and Groovy comments are Java's: // and /* */
			for _, ln := range code {
				if m := gradleModuleName.FindStringSubmatch(ln); m != nil {
					return m[1]
				}
			}
		}
	}
	return partModule(dir[strings.LastIndexByte(dir, '/')+1:])
}

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
// THIS module (its latest m=, as the decoder keeps it, names own, ignoring case; own is ownModule's) and the module
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
		case strings.HasPrefix(ty, "B") && s.types[ty[1:]] && strings.EqualFold(current[sym], own):
			at(lint.Unset, e.Line, `t="`+t+`" keeps the class's B prefix: the type is `+sym+`:`+ty[1:]+` (Workbench: "Type not found")`)
		}
	}
}

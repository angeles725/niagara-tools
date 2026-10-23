package structure

import (
	"errors"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/lint"
	"github.com/angeles725/niagara-tools/go/internal/strerror"
)

var (
	lexiconKey = regexp.MustCompile(`^[^#[:space:]][^=]*=`)
	twoPart    = regexp.MustCompile(`":[^:]+:[0-9]+\.[0-9]+"`)
	bajaPkg    = regexp.MustCompile(`^[[:space:]]*package[[:space:]]+javax\.baja\.`)
	niagaraAnn = regexp.MustCompile(`^[[:space:]]*@NiagaraType`)
	bajaImport = regexp.MustCompile(`^[[:space:]]*import[[:space:]]+javax\.baja\.`)
	btest      = regexp.MustCompile(`^[[:space:]]*import[[:space:]]+(javax\.baja\.test\.|com\.tridium\.btest\.)`)
	junit      = regexp.MustCompile(`^[[:space:]]*import[[:space:]]+org\.junit\.`)
	testWb     = regexp.MustCompile(`":test-wb"`)
	junitDep   = regexp.MustCompile(`"junit[:/][^"]*"`)
)

// profiles lists the root's profiles in name order: every sub-directory (symlinks followed, dot names
// excluded) named *-rt, *-ux, *-wb, *-se or *-doc, or holding build.gradle.kts, build.gradle or
// module-include.xml.
func (s *scan) profiles() []string {
	entries, err := os.ReadDir(s.root)
	if err != nil {
		s.fail(errors.New("cannot list " + s.root + ": " + strerror.Text(err)))
	}
	var out []string
	for _, e := range entries {
		name, p := e.Name(), s.root+"/"+e.Name()
		if strings.HasPrefix(name, ".") || !isDir(p) {
			continue
		}
		if hasProfileSuffix(name) || isFile(p+"/build.gradle.kts") || isFile(p+"/build.gradle") || isFile(p+"/module-include.xml") {
			out = append(out, p)
		}
	}
	return out
}

func hasProfileSuffix(name string) bool {
	for _, suf := range []string{"-rt", "-ux", "-wb", "-se", "-doc"} {
		if strings.HasSuffix(name, suf) {
			return true
		}
	}
	return false
}

// profile runs the per-profile checks L6, L4, L5, L7, L9, L1/L2, L3 and L11, in that row order.
func (s *scan) profile(dir string) {
	name, rel := dir[strings.LastIndexByte(dir, '/')+1:], s.rel(dir)
	inc, lex, pal, src := dir+"/module-include.xml", dir+"/module.lexicon", dir+"/module.palette", dir+"/src"
	hasInc := isFile(inc)
	if !hasInc {
		s.row(lint.Unset, rel, "L6: module-include.xml missing (author it; the gradle plugin generates META-INF/module.xml)")
	} else if meta := dir + "/META-INF/module.xml"; isFile(meta) {
		s.row(lint.Unset, s.rel(meta), "L6: hand-authored META-INF/module.xml found; author module-include.xml instead")
	}
	if hasInc {
		if types := countText(s.lines(inc), "<type "); types > 0 {
			n := strconv.Itoa(types)
			if !isFile(lex) {
				s.row(lint.Unset, rel, "L4: module.lexicon missing ("+n+" type(s) declared in module-include.xml)")
			} else if !grep(s.lines(lex), lexiconKey) {
				s.row(lint.Unset, s.rel(lex), "L4: lexicon empty ("+n+" type(s) declared, zero key=value entries)")
			}
		}
	}
	named := func() bool { return isFile(pal) && countText(s.lines(pal), "<p n=") > 0 } // grep -q '<p n='
	if strings.HasSuffix(name, "-rt") {
		if !isFile(pal) {
			s.row(lint.Unset, rel, "L5: module.palette missing for -rt profile")
		} else if !named() {
			s.row(lint.Unset, s.rel(pal), "L5: module.palette has no named component entries (empty palette — see B788)")
		}
	}
	gradle := s.files(dir, true, "*.gradle.kts")
	for _, g := range gradle {
		if grep(s.lines(g), twoPart) {
			s.row(lint.Unset, s.rel(g), "L7: dependency version is not a 3-part floor (use X.Y.Z e.g. 4.14.0, not X.Y e.g. 4.14)")
		}
	}
	var java []string
	if isDir(src) {
		java = s.files(src, false, "*.java")
	}
	if (strings.HasSuffix(name, "-wb") || strings.HasSuffix(name, "-ux")) && !named() && len(java) == 0 {
		s.row(lint.Unset, rel, "L9: empty skeleton artifact (0 Java classes, empty palette — delete or populate the profile)")
	}
	for _, f := range java {
		lines := s.lines(f)
		if grep(lines, bajaPkg) {
			s.row(lint.Unset, s.rel(f), "L1: package declares javax.baja.* (framework namespace; OEM modules use com.<vendor>.*)")
		}
		if n := count(lines, niagaraAnn); n > 1 {
			s.row(lint.Unset, s.rel(f), "L2: "+strconv.Itoa(n)+" @NiagaraType annotations in one file (one public type per file)")
		}
	}
	s.pureModel(src, java)
	s.mixedTests(dir, gradle)
}

// pureModel is L3: a WARN for every directory named model under src/ (parent before child, siblings in
// name order) holding a .java file that imports javax.baja.*.
func (s *scan) pureModel(src string, java []string) {
	warn := map[string]bool{}
	for _, f := range java {
		if !grep(s.lines(f), bajaImport) {
			continue
		}
		parts := strings.Split(strings.TrimPrefix(f, src+"/"), "/")
		for i := range parts[:len(parts)-1] {
			if parts[i] == "model" {
				warn[src+"/"+strings.Join(parts[:i+1], "/")] = true
			}
		}
	}
	dirs := make([]string, 0, len(warn))
	for d := range warn {
		dirs = append(dirs, d)
	}
	sort.Slice(dirs, func(i, j int) bool { // pre-order: compare component by component
		a, b := strings.Split(dirs[i], "/"), strings.Split(dirs[j], "/")
		for k := 0; k < len(a) && k < len(b); k++ {
			if a[k] != b[k] {
				return a[k] < b[k]
			}
		}
		return len(a) < len(b)
	})
	for _, d := range dirs {
		s.row(lint.Warn, s.rel(d), "L3: pure-model package contains javax.baja.* imports (advisory; isolate control logic from Baja for testability)")
	}
}

// mixedTests is L11: a srcTest/ mixing BTest and JUnit imports needs both a ":test-wb" and a junit
// declaration in the profile's *.gradle.kts files.
func (s *scan) mixedTests(dir string, gradle []string) {
	srcTest := dir + "/srcTest"
	if !isDir(srcTest) {
		return
	}
	var hasBTest, hasJUnit, hasTestWb, hasJUnitDep bool
	for _, f := range s.files(srcTest, false, "*.java") {
		lines := s.lines(f)
		hasBTest = hasBTest || grep(lines, btest)
		hasJUnit = hasJUnit || grep(lines, junit)
	}
	if !hasBTest || !hasJUnit {
		return
	}
	for _, g := range gradle {
		lines := s.lines(g)
		hasTestWb = hasTestWb || grep(lines, testWb)
		hasJUnitDep = hasJUnitDep || grep(lines, junitDep)
	}
	if !hasTestWb || !hasJUnitDep {
		s.row(lint.Unset, s.rel(dir), "L11: mixed BTest+JUnit srcTest without both :test-wb and junit gradle declarations")
	}
}

// countText is grep -c <fixed text>: the number of lines holding it.
func countText(lines []string, text string) int {
	n := 0
	for _, ln := range lines {
		if strings.Contains(ln, text) {
			n++
		}
	}
	return n
}

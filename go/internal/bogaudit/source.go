package bogaudit

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/htmltext"
	"github.com/angeles725/niagara-tools/go/internal/javasrc"
)

// srcSlot is one @NiagaraProperty: its flags expression and MIN facet (hasMin false: none).
type srcSlot struct {
	flags  string
	min    float64
	hasMin bool
}

// sources is what --source-dir declares, keyed by class name (the .java file name): the slots and the
// action flag letters of each class, its direct superclass, and every slot a .set("SLOT", ...) writes.
type sources struct {
	slots   map[string]*omap[srcSlot]
	actions map[string]*omap[string]
	extends map[string]string
	writes  map[string]bool
}

var (
	setCall      = regexp.MustCompile(`\.set\(\s*"([^"]+)"\s*,`)
	nameArg      = regexp.MustCompile(`name\s*=\s*"([^"]+)"`)
	flagsArg     = regexp.MustCompile(`flags\s*=\s*(Flags\.[A-Za-z_|]+|[A-Za-z_][A-Za-z0-9_.|]*)`)
	minDouble    = regexp.MustCompile(`BFacets\.MIN[^,)]*,\s*BDouble\.make\(\s*(-?[0-9.]+)[dDfFlL]?\s*\)`)
	minSeconds   = regexp.MustCompile(`BFacets\.MIN[^,)]*,\s*BRelTime\.makeSeconds\(\s*([0-9]+)[lL]?\s*\)`)
	actionFlag   = regexp.MustCompile(`flags\s*=\s*Flags\.(\w+)`)
	actionLetter = map[string]string{"HIDDEN": "h", "OPERATOR": "o", "READONLY": "r", "SUMMARY": "s", "TRANSIENT": "t"}
)

// scanSources walks dir as os.walk did: each directory's files in name order, then its sub-directories
// in name order, dot-directories skipped (D9b), symlinked directories not entered, unreadable ones and
// unreadable files passed over.
func scanSources(dir string) *sources {
	s := &sources{slots: map[string]*omap[srcSlot]{}, actions: map[string]*omap[string]{},
		extends: map[string]string{}, writes: map[string]bool{}}
	s.walk(dir)
	return s
}

func (s *sources) walk(dir string) {
	ents, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	var dirs []string
	for _, e := range ents {
		p := filepath.Join(dir, e.Name())
		if st, err := os.Stat(p); err == nil && st.IsDir() {
			if e.IsDir() && !strings.HasPrefix(e.Name(), ".") {
				dirs = append(dirs, p)
			}
			continue
		}
		if strings.HasSuffix(e.Name(), ".java") {
			if data, err := os.ReadFile(p); err == nil {
				s.file(strings.TrimSuffix(e.Name(), ".java"), htmltext.Decode(data))
			}
		}
	}
	for _, d := range dirs {
		s.walk(d)
	}
}

// file reads one class: its extends clause, its .set("SLOT", ...) writes, and each @NiagaraProperty or
// @NiagaraAction line joined with the next lines while its parentheses stay open.
func (s *sources) file(cls, content string) {
	if m := regexp.MustCompile(`\bclass\s+` + regexp.QuoteMeta(cls) + `\s+extends\s+(\w+)`).FindStringSubmatch(content); m != nil {
		s.extends[cls] = m[1]
	}
	for _, m := range setCall.FindAllStringSubmatch(content, -1) {
		s.writes[m[1]] = true
	}
	lines := strings.Split(content, "\n")
	for i := 0; i < len(lines); {
		ln := lines[i]
		if !strings.Contains(ln, "@NiagaraProperty") && !strings.Contains(ln, "@NiagaraAction") {
			i++
			continue
		}
		buf, depth, j := ln, javasrc.ParenDepth(ln), i+1
		for ; depth > 0 && j < len(lines); j++ {
			buf += " " + lines[j]
			depth += javasrc.ParenDepth(lines[j])
		}
		i = j
		nm := nameArg.FindStringSubmatch(buf)
		switch {
		case nm == nil:
		case strings.Contains(buf, "@NiagaraProperty"):
			s.property(cls, nm[1], buf)
		default:
			letter := "o" // an action with no flags is operator-visible
			if fm := actionFlag.FindStringSubmatch(buf); fm != nil {
				w := strings.ToUpper(fm[1])
				if letter = actionLetter[w]; letter == "" {
					letter = strings.ToLower(w[:1])
				}
			}
			classMap(s.actions, cls).set(nm[1], letter)
		}
	}
}

func (s *sources) property(cls, name, buf string) {
	var sl srcSlot
	if fm := flagsArg.FindStringSubmatch(buf); fm != nil {
		sl.flags = fm[1]
	}
	if m := minDouble.FindStringSubmatch(buf); m != nil {
		sl.min, sl.hasMin = pyFloat(m[1])
	}
	if m := minSeconds.FindStringSubmatch(buf); m != nil && !sl.hasMin {
		if v, ok := pyFloat(m[1]); ok {
			sl.min, sl.hasMin = v*1000, true
		}
	}
	classMap(s.slots, cls).set(name, sl)
}

func classMap[V any](m map[string]*omap[V], cls string) *omap[V] {
	if m[cls] == nil {
		m[cls] = &omap[V]{}
	}
	return m[cls]
}

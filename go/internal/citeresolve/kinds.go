package citeresolve

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/citeparse"
)

// registerKinds adds the resolvers of the course and bitácora pointers.
func (e *Engine) registerKinds() {
	e.by[citeparse.Course] = courseResolver{e}
	e.by[citeparse.Bitacora] = bitacoraResolver{e}
}

func (e *Engine) course() string {
	return filepath.Join(e.roots.Research, "sources", "n4-dev-course", "course")
}

func (e *Engine) courseText() string {
	return filepath.Join(e.help(), "versions", e.roots.Version, "course-text")
}

type courseResolver struct{ e *Engine }

// Resolve finds the longest prefix of the target's words that names a file or directory of the course
// tree or of its course-text (also as <prefix>.txt); the words after it are unchecked. The pages of a PDF
// are checked in its course-text twin, whose pages end in form feeds.
func (r courseResolver) Resolve(p citeparse.Pointer, _ *Card) Result {
	words := strings.Fields(p.Target)
	for n := len(words); n > 0; n-- {
		rel := strings.Join(words[:n], " ")
		f, err := first(rel, r.e.course(), r.e.courseText())
		if err == nil && f == "" {
			f, err = first(rel+".txt", r.e.courseText())
		}
		if err != nil {
			return toolError(err)
		}
		if f == "" {
			continue
		}
		q := p
		if n < len(words) {
			q.Lines = nil // the range was written after the last word, which is not part of the path
		}
		var res Result
		if strings.HasSuffix(f, ".pdf") {
			res = Result{Status: Resolved, File: f, Detail: "pdf " + r.e.short(f)}
		} else {
			res = r.e.hit(q, f)
		}
		if res.Status == Resolved && len(p.Pages) > 0 {
			res = r.pages(p, f, res)
		}
		if n < len(words) {
			res.Detail += " [unchecked: " + strings.Join(words[n:], " ") + "]"
		}
		return res
	}
	return Result{Status: MissingFile, Detail: "no " + p.Target + " under the course tree or its " + r.e.roots.Version + " course-text"}
}

// pages checks every page range of p against the course-text twin of f.
func (r courseResolver) pages(p citeparse.Pointer, f string, res Result) Result {
	twin := filepath.Join(r.e.courseText(), strings.TrimSuffix(filepath.Base(f), filepath.Ext(f))+".txt")
	if k, err := kind(twin); err != nil {
		return toolError(err)
	} else if k != "file" {
		res.Detail += " [pages not checked: no course-text twin]"
		return res
	}
	lines, err := r.e.fs.lines(twin)
	if err != nil {
		return toolError(err)
	}
	pages := strings.Split(strings.Join(lines, "\n"), "\f")
	if strings.TrimSpace(pages[len(pages)-1]) == "" {
		pages = pages[:len(pages)-1]
	}
	var ex []string
	for _, rg := range p.Pages {
		if rg.From < 1 || rg.To < rg.From || rg.To > len(pages) {
			return Result{Status: LineOutOfRange, File: twin, Detail: fmt.Sprintf("pages %d-%d, %s has %d", rg.From, rg.To, filepath.Base(twin), len(pages))}
		}
		ex = append(ex, "p"+fmt.Sprint(rg.From)+": "+excerpt(firstText(pages[rg.From-1])))
	}
	res.Detail = strings.Join(ex, " | ") + " [read " + r.e.short(twin) + "]"
	return res
}

// firstText is the first line of a page that is not blank or only its page number.
func firstText(page string) string {
	for _, l := range strings.Split(page, "\n") {
		if t := strings.TrimSpace(l); t != "" && strings.Trim(t, "0123456789") != "" {
			return l
		}
	}
	return ""
}

type bitacoraResolver struct{ e *Engine }

// Resolve finds the client bitacora/<date>-<slug>.md (or <slug>.md), a checkout and its worktrees being
// one file, and checks each section as a numbered or titled heading.
func (r bitacoraResolver) Resolve(p citeparse.Pointer, _ *Card) Result {
	t := r.e.clientTree()
	idx, err := r.e.fs.index(t.root, t.skip, r.e.treeRoots())
	if err != nil {
		return toolError(err)
	}
	var names []string
	for base := range idx {
		if base == p.Target+".md" || strings.HasSuffix(base, "-"+p.Target+".md") {
			names = append(names, base)
		}
	}
	sort.Strings(names)
	g := grouper{at: map[string]int{}}
	for _, base := range names {
		for _, f := range idx[base] {
			if filepath.Base(filepath.Dir(f)) == "bitacora" {
				g.add(t, f)
			}
		}
	}
	return r.e.pick(p, g.groups(), "bitacora/*-"+p.Target+".md",
		Result{Status: MissingFile, Detail: "no bitacora/<date>-" + p.Target + ".md under " + t.root})
}

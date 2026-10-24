package citeresolve

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/citeparse"
)

// registerFiles adds the resolvers of the pointers that name a file in a tree.
func (e *Engine) registerFiles() {
	e.by[citeparse.Path] = pathResolver{e}
	e.by[citeparse.Code] = codeResolver{e}
	e.by[citeparse.Devguide] = guideResolver{e, "devguide-clean"}
	e.by[citeparse.Guide] = guideResolver{e, "guides-clean"}
	e.by[citeparse.Help] = helpResolver{e}
	e.by[citeparse.Mem] = memResolver{e}
	e.by[citeparse.Vineflower] = vineResolver{e}
	e.by[citeparse.Javap] = jarResolver{e}
	e.by[citeparse.Jar] = jarResolver{e}
	e.registerKinds()
}

func (e *Engine) help() string      { return filepath.Join(e.roots.Research, "niagara-help") }
func (e *Engine) organized() string { return filepath.Join(e.roots.Research, "organized") }

// first returns the first of bases/rel that exists ("" when none).
func first(rel string, bases ...string) (string, error) {
	for _, b := range bases {
		p := filepath.Join(b, rel)
		if k, err := kind(p); err != nil {
			return "", err
		} else if k != "" {
			return p, nil
		}
	}
	return "", nil
}

// hit checks the locators of p against the file (or directory) at path.
func (e *Engine) hit(p citeparse.Pointer, path string) Result {
	if k, err := kind(path); err != nil {
		return toolError(err)
	} else if k == "dir" {
		if len(p.Lines)+len(p.Tokens)+len(p.Sections) > 0 {
			return Result{Status: MissingFile, File: path, Detail: path + " is a directory"}
		}
		return Result{Status: Resolved, File: path, Detail: "directory"}
	}
	lines, err := e.fs.lines(path)
	if err != nil {
		return toolError(err)
	}
	res, ok := checkLines(p, lines)
	if !ok {
		return res
	}
	if drift, ok := checkAnchor(p, path, lines); !ok {
		return drift
	}
	var ex []string
	if res.Detail != "" {
		ex = append(ex, res.Detail)
	}
	for _, t := range p.Tokens {
		i := find(lines, func(l string) bool { return strings.Contains(l, t) })
		if i < 0 {
			return Result{Status: MissingSection, File: path, Detail: "token " + t + " not in " + filepath.Base(path)}
		}
		ex = append(ex, excerpt(lines[i]))
	}
	for _, s := range p.Sections {
		i := find(lines, headingMatcher(s))
		if i < 0 {
			return Result{Status: MissingSection, File: path, Detail: "no heading §" + s + " in " + filepath.Base(path)}
		}
		ex = append(ex, excerpt(lines[i]))
	}
	if len(ex) == 0 && len(lines) > 0 {
		if ex = []string{firstHeading(lines)}; ex[0] == "" {
			ex[0] = excerpt(lines[0])
		}
	}
	return Result{Status: Resolved, File: path, Detail: strings.Join(ex, " | ")}
}

// headingMatcher matches a kit heading by number (`## 2. Software`, `### 2.2 Go`, `## 1 · How`) or by
// title (a heading or the bold lead of a bullet): the title's words and the heading's words (each cut at its first `—`, `(`, `[`, `:` or backtick,
// numbering dropped) agree until one of them ends, so a title that runs on into prose still matches.
func headingMatcher(s string) func(string) bool {
	tw := titleWords(s)
	return func(l string) bool {
		h, ok := headingText(l)
		if numeric.MatchString(s) {
			return ok && numbered(h, s)
		}
		if !ok {
			if h, ok = boldBullet(l); !ok {
				return false
			}
		}
		hw := titleWords(h)
		n := 0
		for n < len(tw) && n < len(hw) && tw[n] == hw[n] {
			n++
		}
		return n > 0 && (n == len(tw) || n == len(hw))
	}
}

var boldLead = regexp.MustCompile(`^\s*[-*]\s+\*\*([^*]+)\*\*`)

// boldBullet is the bold lead of a `- **Title:** text` bullet, which kit cards cite like a heading.
func boldBullet(l string) (string, bool) {
	m := boldLead.FindStringSubmatch(l)
	if m == nil {
		return "", false
	}
	return m[1], true
}

var titleCut = regexp.MustCompile("\\s[—–]\\s|\\(|\\[|:\\s|`")

// titleWords lowercases the words of a title up to its first cut, numbering and punctuation dropped. A
// title that opens with a code span (`module-include.xml` vs …) keeps that span's text as its first word.
func titleWords(s string) []string {
	s = unwrapLeadingCode(s)
	if loc := titleCut.FindStringIndex(s); loc != nil {
		s = s[:loc[0]]
	}
	var out []string
	for _, w := range strings.Fields(strings.ToLower(s)) {
		if w = strings.Trim(w, "\"'*,.;:·"); w != "" && !(len(out) == 0 && numeric.MatchString(w)) {
			out = append(out, w)
		}
	}
	return out
}

// unwrapLeadingCode drops the backticks of a code span that opens s, so the backtick cut of titleWords
// ends the title after it, not before it.
func unwrapLeadingCode(s string) string {
	rest, ok := strings.CutPrefix(strings.TrimSpace(s), "`")
	if !ok {
		return s
	}
	if code, after, closed := strings.Cut(rest, "`"); closed {
		return code + after
	}
	return s
}

type pathResolver struct{ e *Engine }

// Resolve looks for the path under the card's directory, the kit, the repository, the research tree,
// niagara-help (its versioned copy first, so `source/…` reads the cited version), organized/, the module
// trees, the course tree, then the guide and devguide trees (versioned first), in that order; `~/rest`
// under the home root; `KIT/rest` under the kit; `…/rest` under an earlier hit of the card, else in the
// trees and the course tree; `dir/…/rest` below dir. A path found under none of them (a bare file name, a
// partial package path) is searched in the trees.
func (r pathResolver) Resolve(p citeparse.Pointer, c *Card) Result {
	if rest, ok := strings.CutPrefix(p.Target, "…/"); ok {
		for i := len(c.files) - 1; i >= 0; i-- {
			for d := filepath.Dir(c.files[i]); d != "/" && d != "."; d = filepath.Dir(d) {
				if f, err := first(rest, d); err != nil {
					return toolError(err)
				} else if f != "" {
					return r.e.hit(p, f)
				}
			}
		}
		return r.e.search(p, rest, "ellipsis prefix: no earlier hit on this card holds "+rest+", nor a searched tree", r.e.course())
	}
	if strings.Contains(p.Target, "/…/") {
		return r.e.infix(p, c)
	}
	ro := r.e.roots
	ver := filepath.Join(r.e.help(), "versions", ro.Version)
	target, bases := p.Target, []string{c.Dir, ro.Kit, ro.Repo, ro.Research, ver, r.e.help(), r.e.organized(), ro.Modules, r.e.course(),
		filepath.Join(ver, "guides-clean"), filepath.Join(ver, "devguide-clean"),
		filepath.Join(r.e.help(), "guides-clean"), filepath.Join(r.e.help(), "devguide-clean")}
	target = r.e.homePath(target)
	if rest, ok := strings.CutPrefix(strings.TrimPrefix(target, "$"), "KIT/"); ok {
		target, bases = rest, []string{ro.Kit}
	}
	if filepath.IsAbs(target) {
		bases = []string{"/"}
	}
	f, err := first(target, bases...)
	switch {
	case err != nil:
		return toolError(err)
	case f != "":
		return r.e.hit(p, f)
	case filepath.IsAbs(target):
		return Result{Status: MissingFile, Detail: target + " does not exist"}
	}
	return r.e.search(p, target, target+" not under the card, kit, repo, research or modules roots, nor a searched tree")
}

// homePath expands a leading `~/` to the home root, so the path resolves by existence like any absolute
// path; a kit path (`KIT/…`) or an unknown home leaves it as written.
func (e *Engine) homePath(target string) string {
	if rest, ok := strings.CutPrefix(target, "~/"); ok && e.roots.Home != "" {
		return filepath.Join(e.roots.Home, rest)
	}
	return target
}

// search resolves p by the files named, or ending in, target in the trees (repository, organized/,
// versioned source, clients) and below each extra directory.
func (e *Engine) search(p citeparse.Pointer, target, missing string, extra ...string) Result {
	groups, err := e.lookup(target, nil, extra...)
	if err != nil {
		return toolError(err)
	}
	return e.pick(p, groups, target, Result{Status: MissingFile, Detail: missing})
}

// organizedDirs are the organized/ copies of a module part, organized/<mod>/<part>/ (also
// organized/<part>/<part>/), and organized/<name>/ for a module named without its part.
func (e *Engine) organizedDirs(name string) []string {
	mod := name
	if i := strings.LastIndexByte(name, '-'); i > 0 {
		mod = name[:i]
	}
	return []string{filepath.Join(e.organized(), mod, name), filepath.Join(e.organized(), name, name),
		filepath.Join(e.organized(), name)}
}

// moduleDirs are the organized/ copies of a module part and niagara-help/versions/<version>/source/<part>/.
func (e *Engine) moduleDirs(name string) []string {
	return append(e.organizedDirs(name), filepath.Join(e.source(), name))
}

// inTrees finds file under dirs: joined to each dir when it has a directory, else by base name anywhere
// below the dirs.
func (e *Engine) inTrees(file string, dirs []string) (string, error) {
	if strings.Contains(file, "/") {
		return first(file, dirs...)
	}
	for _, d := range dirs {
		idx, err := e.fs.index(d, baseSkip, nil)
		if err != nil {
			return "", err
		}
		if hits := idx[file]; len(hits) > 0 {
			return hits[0], nil
		}
	}
	return "", nil
}

// index maps each base name below dir to its paths, in walk order, entering no directory named in skip and
// no other tree root (cached; a missing dir is empty).
func (f *files) index(dir string, skip, roots map[string]bool) (map[string][]string, error) {
	if f.idx == nil {
		f.idx = map[string]map[string][]string{}
	}
	if m, ok := f.idx[dir]; ok {
		return m, nil
	}
	m := map[string][]string{}
	if k, err := kind(dir); err != nil || k != "dir" {
		f.idx[dir] = m
		return m, err
	}
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && path != dir && (skip[d.Name()] || roots[path]) {
			return filepath.SkipDir
		}
		if !d.IsDir() {
			m[d.Name()] = append(m[d.Name()], path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("cannot list every file under %s: %w", dir, err)
	}
	f.idx[dir] = m
	return m, nil
}

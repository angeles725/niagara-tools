package citeresolve

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/citeparse"
)

// copies are the sub-directories of an organized/<mod>/<part>/ module copy that hold source files.
var copies = []string{"vineflower", "decompiled", "extracted", ""}

// registerFiles adds the resolvers of the pointers that name a file in a tree.
func (e *Engine) registerFiles() {
	e.by[citeparse.Path] = pathResolver{e}
	e.by[citeparse.Code] = codeResolver{e}
	e.by[citeparse.Devguide] = guideResolver{e, "devguide-clean"}
	e.by[citeparse.Guide] = guideResolver{e, "guides-clean"}
	e.by[citeparse.Help] = helpResolver{e}
	e.by[citeparse.Mem] = memResolver{e}
	e.by[citeparse.Vineflower] = vineResolver{e}
	e.by[citeparse.Javap] = javapResolver{e}
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

// headingMatcher matches a kit heading by number (`## 2. Software`, `### 2.2 Go`) or by title text.
func headingMatcher(s string) func(string) bool {
	low := strings.ToLower(s)
	return func(l string) bool {
		h, ok := headingText(l)
		if !ok {
			return false
		}
		if numeric.MatchString(s) {
			return numbered(h, s)
		}
		core := strings.ToLower(strings.TrimLeft(h, "0123456789.) "))
		return strings.Contains(strings.ToLower(h), low) || (len(core) >= 4 && strings.HasPrefix(low, core))
	}
}

type pathResolver struct{ e *Engine }

// Resolve looks for the path under the card's directory, the kit, the repository, the research tree,
// niagara-help and the module trees, in that order; `…/rest` under an earlier hit of the card.
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
		return Result{Status: MissingFile, Detail: "ellipsis prefix: no earlier hit on this card holds " + rest}
	}
	ro := r.e.roots
	f, err := first(p.Target, c.Dir, ro.Kit, ro.Repo, ro.Research, r.e.help(), ro.Modules)
	switch {
	case err != nil:
		return toolError(err)
	case f != "":
		return r.e.hit(p, f)
	case !strings.Contains(p.Target, "/"):
		return Result{Status: Unparsed, Detail: "bare file name: not in the card directory or a root, no tree named to search"}
	}
	return Result{Status: MissingFile, Detail: p.Target + " not under the card, kit, repo, research or modules roots"}
}

// moduleDirs are the copies of a module part: organized/<mod>/<part>/ (also organized/<part>/<part>/)
// and niagara-help/versions/<version>/source/<part>/.
func (e *Engine) moduleDirs(part string) []string {
	mod := part
	if i := strings.LastIndexByte(part, '-'); i > 0 {
		mod = part[:i]
	}
	return []string{filepath.Join(e.organized(), mod, part), filepath.Join(e.organized(), part, part),
		filepath.Join(e.help(), "versions", e.roots.Version, "source", part)}
}

// inTrees finds file under dirs: joined to each dir's copies when it has a directory, else by base name
// anywhere below the dirs.
func (e *Engine) inTrees(file string, dirs []string, subs []string) (string, error) {
	if strings.Contains(file, "/") {
		for _, d := range dirs {
			for _, s := range subs {
				if f, err := first(file, filepath.Join(d, s)); err != nil || f != "" {
					return f, err
				}
			}
		}
		return "", nil
	}
	for _, d := range dirs {
		idx, err := e.fs.index(d)
		if err != nil {
			return "", err
		}
		if hits := idx[file]; len(hits) > 0 {
			return hits[0], nil
		}
	}
	return "", nil
}

// index maps each base name below dir to its paths, in walk order (cached; a missing dir is empty).
func (f *files) index(dir string) (map[string][]string, error) {
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
		if d.IsDir() && (d.Name() == ".git" || d.Name() == "pipeline") {
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

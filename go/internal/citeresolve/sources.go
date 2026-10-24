package citeresolve

import (
	"path/filepath"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/citeparse"
)

type codeResolver struct{ e *Engine }

// Resolve finds File in the copies of module part Target; with no part, as a root-relative path and
// then in every decompile copy and versioned source tree.
func (r codeResolver) Resolve(p citeparse.Pointer, c *Card) Result {
	q := p
	q.Target = p.File
	if p.Target != "" {
		f, err := r.e.inTrees(p.File, r.e.moduleDirs(p.Target), copies)
		switch {
		case err != nil:
			return toolError(err)
		case f == "":
			return Result{Status: MissingFile, Detail: "no " + p.File + " in the copies of " + p.Target}
		}
		return r.e.hit(q, f)
	}
	res := pathResolver{r.e}.Resolve(q, c)
	if res.Status != MissingFile || strings.HasPrefix(p.File, "…/") {
		return res
	}
	dirs, err := r.e.sourceDirs()
	if err != nil {
		return toolError(err)
	}
	f, err := r.e.inTrees(p.File, dirs, copies)
	switch {
	case err != nil:
		return toolError(err)
	case f == "":
		return Result{Status: MissingFile, Detail: p.File + " not under a root, organized/ copy or " + r.e.roots.Version + " source"}
	}
	return r.e.hit(q, f)
}

// sourceDirs lists every organized/<mod>/<part>/ and niagara-help/versions/<version>/source/<part>/.
func (e *Engine) sourceDirs() ([]string, error) {
	var out []string
	mods, err := e.fs.names(e.organized())
	if err != nil {
		return nil, err
	}
	for _, m := range mods {
		parts, err := e.fs.names(filepath.Join(e.organized(), m))
		if err != nil {
			return nil, err
		}
		for _, p := range parts {
			out = append(out, filepath.Join(e.organized(), m, p))
		}
	}
	src := filepath.Join(e.help(), "versions", e.roots.Version, "source")
	parts, err := e.fs.names(src)
	if err != nil {
		return nil, err
	}
	for _, p := range parts {
		out = append(out, filepath.Join(src, p))
	}
	return out, nil
}

type guideResolver struct {
	e   *Engine
	dir string // devguide-clean or guides-clean
}

// Resolve prefers niagara-help/versions/<version>/<dir>; a file only in the unversioned <dir> (4.14)
// resolves with a version-mismatch note.
func (r guideResolver) Resolve(p citeparse.Pointer, _ *Card) Result {
	ver := filepath.Join(r.e.help(), "versions", r.e.roots.Version, r.dir)
	old := filepath.Join(r.e.help(), r.dir)
	for _, d := range []string{ver, old} {
		f, err := r.e.inTrees(p.Target, []string{d}, []string{""})
		if err != nil {
			return toolError(err)
		}
		if f == "" {
			continue
		}
		res := r.e.hit(p, f)
		if d == old {
			res.Detail += " [version mismatch: only in the unversioned " + r.dir + " (4.14), not " + r.e.roots.Version + "]"
		}
		return res
	}
	return Result{Status: MissingFile, Detail: "no " + p.Target + " under " + r.dir + " (" + r.e.roots.Version + " or unversioned)"}
}

type helpResolver struct{ e *Engine }

// Resolve finds niagara-help/versions/<Version>/<Target>.
func (r helpResolver) Resolve(p citeparse.Pointer, _ *Card) Result {
	f, err := first(p.Target, filepath.Join(r.e.help(), "versions", p.Version))
	switch {
	case err != nil:
		return toolError(err)
	case f == "":
		return Result{Status: MissingFile, Detail: "no niagara-help/versions/" + p.Version + "/" + p.Target}
	}
	return r.e.hit(p, f)
}

type memResolver struct{ e *Engine }

// Resolve finds the memory files whose name starts with Target.
func (r memResolver) Resolve(p citeparse.Pointer, _ *Card) Result {
	names, err := r.e.fs.names(r.e.roots.Memory)
	if err != nil {
		return toolError(err)
	}
	var hits []string
	for _, n := range names {
		if strings.HasPrefix(n, p.Target) && strings.HasSuffix(n, ".md") {
			hits = append(hits, n)
		}
	}
	if len(hits) == 0 {
		return Result{Status: MissingFile, Detail: "no memory file starts with " + p.Target}
	}
	return Result{Status: Resolved, File: filepath.Join(r.e.roots.Memory, hits[0]), Detail: strings.Join(hits, ", ")}
}

type vineResolver struct{ e *Engine }

// Resolve finds File in the vineflower/ copy of module part Target; the copy's version is not checked.
func (r vineResolver) Resolve(p citeparse.Pointer, _ *Card) Result {
	f, err := r.e.inTrees(p.File, r.e.moduleDirs(p.Target)[:2], []string{"vineflower"})
	switch {
	case err != nil:
		return toolError(err)
	case f == "":
		return Result{Status: MissingFile, Detail: "no " + p.File + " in the vineflower copy of " + p.Target}
	}
	q := p
	q.Target = p.File
	res := r.e.hit(q, f)
	if p.Version != "" {
		res.Detail += " [copy version not checked against " + p.Version + "]"
	}
	return res
}

type javapResolver struct{ e *Engine }

// Resolve finds the jar in organized/; what javap printed is not reproduced.
func (r javapResolver) Resolve(p citeparse.Pointer, _ *Card) Result {
	jar := filepath.Base(p.Target)
	f, err := first(jar, r.e.moduleDirs(strings.TrimSuffix(jar, ".jar"))[:2]...)
	switch {
	case err != nil:
		return toolError(err)
	case f == "":
		return Result{Status: MissingFile, Detail: "no " + jar + " under organized/"}
	}
	return Result{Status: Resolved, File: f, Detail: "jar " + f + " [javap output not checked]"}
}

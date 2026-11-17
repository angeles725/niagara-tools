package citeresolve

import (
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/citeparse"
)

// generatedPaths are the repo paths a build or a toolbelt run writes and git does not track, so a doctrine
// mention of one resolves on any machine (a clean clone has none of them). A "/"-ending entry is a prefix.
var generatedPaths = []string{"toolbelt/INDEX.md", "go/bin/"}

type repoPathResolver struct{ e *Engine }

// Resolve finds a backticked toolbelt/, scripts/, go/ or tests/ path of prose under the kit or the
// repository root, by that exact path: never by base name elsewhere, so a deleted file is MISSING_FILE
// until the doctrine that names it is reworded (retro 2026-11-13-remove-dead-bash-lint-libs Δ1).
func (r repoPathResolver) Resolve(p citeparse.Pointer, _ *Card) Result {
	for _, g := range generatedPaths {
		if p.Target == g || strings.HasSuffix(g, "/") && strings.HasPrefix(p.Target, g) {
			return Result{Status: Resolved, Detail: "generated, not committed"}
		}
	}
	f, err := first(p.Target, r.e.roots.Kit, r.e.roots.Repo)
	switch {
	case err != nil:
		return toolError(err)
	case f == "":
		return Result{Status: MissingFile, Detail: "no " + p.Target + " under the kit or the repository"}
	}
	return r.e.hit(p, f)
}

// outsidePath is the path an [ev: …] pointer names when it is a home (`~/`) or absolute path: such evidence
// resolves on one machine and not another, so it is OUTSIDE_REPO before any lookup (Δ2 of the same retro).
// A [deferred: …] tag and prose keep resolving by existence.
func outsidePath(p citeparse.Pointer) (string, bool) {
	if !p.Evidence {
		return "", false
	}
	path := p.Target
	switch {
	case p.Kind == citeparse.Code && p.Target == "":
		path = p.File
	case p.Kind != citeparse.Path && p.Kind != citeparse.Archive:
		return "", false
	}
	return path, strings.HasPrefix(path, "/") || strings.HasPrefix(path, "~/")
}

// outside is the OUTSIDE_REPO result of path.
func outside(path string) Result {
	return Result{Status: OutsideRepo, Detail: path + " is outside the repository: cite a repo-relative path, or record it as live evidence (`[ev: live read of …]`)"}
}

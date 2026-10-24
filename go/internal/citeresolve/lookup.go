package citeresolve

import (
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/citeparse"
)

// maxCandidates is how many files an AMBIGUOUS detail lists before it counts the rest.
const maxCandidates = 4

// tree is one root a bare or partial path is searched in: indexed by base name in one walk per run, with
// the logical name of each file in it, so the copies of one file group together.
type tree struct {
	root string
	skip map[string]bool // directory names the walk does not enter
	key  func(rel []string) string
}

var (
	baseSkip    = map[string]bool{".git": true, "pipeline": true}
	repoSkip    = map[string]bool{".git": true, "testdata": true}
	clientSkip  = map[string]bool{".git": true, "build": true, ".gradle": true, "node_modules": true, "_salvage": true}
	copyDirs    = map[string]bool{"vineflower": true, "decompiled": true, "extracted": true}
	copyOrder   = []string{"vineflower", "decompiled", "extracted"} // the copyDirs in search order
	worktreeDir = regexp.MustCompile(`^(.+)-worktrees$`)
)

// trees are the roots searched by base name: the repository holding the kit, organized/, the versioned
// niagara-help source and the client trees. Never the whole research tree, never / or /mnt/c.
func (e *Engine) trees() []tree {
	return []tree{
		{e.roots.Repo, repoSkip, func(r []string) string { return "repo:" + strings.Join(r, "/") }},
		{e.organized(), baseSkip, organizedKey},
		{e.source(), baseSkip, func(r []string) string { return "src:" + strings.Join(r, "/") }},
		e.clientTree(),
	}
}

func (e *Engine) clientTree() tree { return tree{e.roots.Clients, clientSkip, clientKey} }

func (e *Engine) source() string {
	return filepath.Join(e.help(), "versions", e.roots.Version, "source")
}

// docSourcePart is the organized/ part of the docSource module, whose copies hold <part>/<rest> trees.
const docSourcePart = "docSource-doc"

// organizedKey names organized/<mod>/<part>/<copy>/<rest> as <part>/<rest>, the name the versioned
// source gives the same file; organized/docSource/docSource-doc/<copy>/<part>/<rest> names it too.
func organizedKey(r []string) string {
	if len(r) >= 5 && r[1] == docSourcePart && copyDirs[r[2]] {
		return "src:" + strings.Join(r[3:], "/")
	}
	if len(r) >= 4 && copyDirs[r[2]] {
		return "src:" + r[1] + "/" + strings.Join(r[3:], "/")
	}
	return "org:" + strings.Join(r, "/")
}

// clientKey names a file of a <checkout>-worktrees/<tree>/ copy as the same file of <checkout>.
func clientKey(r []string) string {
	for i := 0; i+1 < len(r); i++ {
		if m := worktreeDir.FindStringSubmatch(r[i]); m != nil {
			r = append(append(append([]string(nil), r[:i]...), m[1]), r[i+2:]...)
			break
		}
	}
	return "client:" + strings.Join(r, "/")
}

// copyRank orders the copies of one file: the module part's vineflower, decompiled and extracted copies,
// then the docSource copies, then the versioned source.
func copyRank(p string) int {
	if strings.Contains(p, "/"+docSourcePart+"/") {
		return len(copyDirs)
	}
	for i, c := range copyOrder {
		if strings.Contains(p, "/"+c+"/") {
			return i
		}
	}
	return len(copyDirs) + 1
}

// group is the copies of one logical file, in search order.
type group struct {
	key    string
	copies []string
}

// lookup finds the files named target, or whose path ends in /target, in the trees (keeping those keep
// accepts; nil keeps all) and below each extra directory, grouped by logical file.
func (e *Engine) lookup(target string, keep func(string) bool, extra ...string) ([]group, error) {
	all := e.trees()
	for _, d := range extra {
		d := d
		all = append(all, tree{d, baseSkip, func(r []string) string { return "dir:" + filepath.Join(d, filepath.Join(r...)) }})
	}
	g := grouper{at: map[string]int{}}
	for _, t := range all {
		idx, err := e.fs.index(t.root, t.skip, e.treeRoots())
		if err != nil {
			return nil, err
		}
		for _, p := range idx[filepath.Base(target)] {
			if strings.HasSuffix(p, "/"+target) && (keep == nil || keep(p)) {
				g.add(t, p)
			}
		}
	}
	return g.groups(), nil
}

// grouper collects files into groups by the logical key their tree gives them.
type grouper struct {
	out []group
	at  map[string]int
}

func (g *grouper) add(t tree, p string) {
	rel, _ := filepath.Rel(t.root, p)
	k := t.key(strings.Split(filepath.ToSlash(rel), "/"))
	if i, ok := g.at[k]; ok {
		g.out[i].copies = append(g.out[i].copies, p)
		return
	}
	g.at[k] = len(g.out)
	g.out = append(g.out, group{k, []string{p}})
}

// groups returns the groups, each with its copies in copyRank order.
func (g *grouper) groups() []group {
	for _, x := range g.out {
		sort.SliceStable(x.copies, func(a, b int) bool { return copyRank(x.copies[a]) < copyRank(x.copies[b]) })
	}
	return g.out
}

// treeRoots are the tree roots and the research root, which no other tree's walk enters (a fixture
// repository holds them all).
func (e *Engine) treeRoots() map[string]bool {
	m := map[string]bool{e.roots.Research: true}
	for _, t := range e.trees() {
		m[t.root] = true
	}
	return m
}

// covered reports whether dir is a tree root or below one, so the trees' indexes already list it.
func (e *Engine) covered(dir string) bool {
	for _, t := range e.trees() {
		if dir == t.root || strings.HasPrefix(dir, t.root+"/") {
			return true
		}
	}
	return false
}

// pick resolves p against what lookup found: one logical file is checked copy by copy; several are
// AMBIGUOUS, listed, never a guess; none is missing.
func (e *Engine) pick(p citeparse.Pointer, groups []group, what string, missing Result) Result {
	switch len(groups) {
	case 0:
		return missing
	case 1:
		return e.hitCopies(p, groups[0].copies)
	}
	var names []string
	for i, g := range groups {
		if i == maxCandidates {
			names = append(names, fmt.Sprintf("+%d more", len(groups)-i))
			break
		}
		names = append(names, e.short(g.copies[0]))
	}
	return Result{Status: Ambiguous, Detail: fmt.Sprintf("%d files match %s: %s", len(groups), what, strings.Join(names, ", "))}
}

// hitCopies checks p against each copy of one file and returns the first that holds every locator,
// naming the copy read; when none does, the first copy's failure, naming the copies checked.
func (e *Engine) hitCopies(p citeparse.Pointer, copies []string) Result {
	var first Result
	for i, c := range copies {
		r := e.hit(p, c)
		if r.Status == Resolved || r.Status == ToolError {
			r.Detail = strings.TrimSpace(r.Detail + " [read " + e.short(c) + "]")
			return r
		}
		if i == 0 {
			first = r
		}
	}
	if len(copies) > 1 {
		names := make([]string, len(copies))
		for i, c := range copies {
			names[i] = e.short(c)
		}
		first.Detail += fmt.Sprintf(" [%d copies checked: %s]", len(copies), strings.Join(names, ", "))
	}
	return first
}

// short names path relative to the research tree, the parent of the client trees, or the repository.
func (e *Engine) short(path string) string {
	for _, base := range []string{e.roots.Research, filepath.Dir(e.roots.Clients), e.roots.Repo} {
		if rel, err := filepath.Rel(base, path); err == nil && !strings.HasPrefix(rel, "..") {
			return rel
		}
	}
	return path
}

// inModule keeps the files below dirs, and the client files below a directory named module or module-<x>.
func (e *Engine) inModule(module string, dirs []string) func(string) bool {
	return func(p string) bool {
		for _, d := range dirs {
			if strings.HasPrefix(p, d+"/") {
				return true
			}
		}
		rel, err := filepath.Rel(e.roots.Clients, p)
		if err != nil || strings.HasPrefix(rel, "..") {
			return false
		}
		for _, seg := range strings.Split(filepath.Dir(rel), "/") {
			if seg == module || strings.HasPrefix(seg, module+"-") {
				return true
			}
		}
		return false
	}
}

package cisteps

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/gitcmd"
)

// changeSet is where the changed files come from: a commit range (the bytes of its end commit, what
// a push sends) or, with no range, the work tree against where HEAD left origin/main (untracked
// files included: what a developer is about to commit).
type changeSet struct {
	end   string // the range's end commit; "" reads the work tree
	paths []string
	errTo io.Writer
}

// listChanges runs git in the process directory; deleted files are left out (nothing to check).
func listChanges(rng string, errTo io.Writer) (changeSet, error) {
	g := gitcmd.Command{Stderr: errTo}
	if rng != "" {
		from, to, ok := strings.Cut(rng, "..")
		if !ok || from == "" || to == "" || strings.HasPrefix(to, ".") || strings.Contains(to, "..") {
			return changeSet{}, fmt.Errorf("--range %q is not <from>..<to>", rng)
		}
		res := g.Run("diff", "--name-only", "--no-renames", "--diff-filter=d", "-z", from, to)
		if !res.OK() {
			return changeSet{}, fmt.Errorf("git diff %s failed", rng)
		}
		return changeSet{end: to, paths: split0(res.Stdout), errTo: errTo}, nil
	}
	base := g.Run("merge-base", "origin/main", "HEAD")
	if !base.OK() {
		return changeSet{}, errors.New("cannot find where HEAD left origin/main (git fetch origin first)")
	}
	diff := g.Run("diff", "--name-only", "--no-renames", "--diff-filter=d", "-z", strings.TrimSpace(base.Stdout))
	others := g.Run("ls-files", "-z", "--others", "--exclude-standard")
	if !diff.OK() || !others.OK() {
		return changeSet{}, errors.New("git diff / ls-files against origin/main failed")
	}
	seen := map[string]bool{}
	var paths []string
	for _, p := range append(split0(diff.Stdout), split0(others.Stdout)...) {
		if !seen[p] {
			seen[p] = true
			paths = append(paths, p)
		}
	}
	sort.Strings(paths)
	return changeSet{paths: paths, errTo: errTo}, nil
}

func split0(out string) []string {
	var paths []string
	for _, p := range strings.Split(out, "\x00") {
		if p != "" {
			paths = append(paths, p)
		}
	}
	return paths
}

// allFiles lists every file of the change's end state: the range's end commit, or the work tree
// (tracked files that still exist plus untracked ones).
func (c changeSet) allFiles() ([]string, error) {
	g := gitcmd.Command{Stderr: c.errTo}
	if c.end != "" {
		res := g.Run("ls-tree", "-r", "-z", "--name-only", c.end)
		if !res.OK() {
			return nil, fmt.Errorf("git ls-tree %s failed", c.end)
		}
		return split0(res.Stdout), nil
	}
	tracked := g.Run("ls-files", "-z")
	others := g.Run("ls-files", "-z", "--others", "--exclude-standard")
	if !tracked.OK() || !others.OK() {
		return nil, errors.New("git ls-files failed")
	}
	var files []string
	for _, p := range append(split0(tracked.Stdout), split0(others.Stdout)...) {
		if st, err := os.Lstat(p); err == nil && st.Mode().IsRegular() {
			files = append(files, p)
		}
	}
	sort.Strings(files)
	return files, nil
}

// materialize writes each path's bytes under tree at the same relative path, so the checks read
// exactly the change and name its files as the repository does.
func (c changeSet) materialize(tree string, paths []string) error {
	for _, p := range paths {
		var data []byte
		if c.end != "" {
			res := gitcmd.Command{Stderr: c.errTo}.Run("show", c.end+":"+p)
			if !res.OK() {
				return fmt.Errorf("git show %s:%s failed", c.end, p)
			}
			data = []byte(res.Stdout)
		} else {
			b, err := os.ReadFile(p)
			if err != nil {
				return err
			}
			data = b
		}
		dst := filepath.Join(tree, filepath.FromSlash(p))
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(dst, data, 0o644); err != nil {
			return err
		}
	}
	return nil
}

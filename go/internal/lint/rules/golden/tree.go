package golden

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// materialize writes tree under root: parents as 0755 directories, then the explicit modes, deepest first,
// so a locked directory is locked after its contents are written.
func materialize(root string, tree map[string]entry) error {
	paths := make([]string, 0, len(tree))
	for p := range tree {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	for _, p := range paths {
		e, full := tree[p], filepath.Join(root, p)
		err := os.MkdirAll(filepath.Dir(full), 0o755)
		if err == nil {
			switch e.kind {
			case "dir":
				err = os.MkdirAll(full, 0o755)
			case "symlink":
				err = os.Symlink(e.body, full)
			default:
				err = os.WriteFile(full, []byte(e.body), 0o644)
			}
		}
		if err != nil {
			return err
		}
	}
	for i := len(paths) - 1; i >= 0; i-- {
		if e := tree[paths[i]]; e.kind != "symlink" {
			if err := os.Chmod(filepath.Join(root, paths[i]), e.mode); err != nil {
				return err
			}
		}
	}
	return nil
}

// snapshot is every entry under root by slash path, spelled "<kind> <mode> <content|target>", except the
// skip entries. It unlocks what it reads (u+r, and u+rx on a directory), so a snapshot dir stays removable.
func snapshot(root string, skip []string) (map[string]string, error) {
	skipped := map[string]bool{}
	for _, s := range skip {
		skipped[s] = true
	}
	out := map[string]string{}
	var walk func(rel string) error
	walk = func(rel string) error {
		d := filepath.Join(root, rel)
		if fi, err := os.Lstat(d); err != nil {
			return err
		} else if fi.Mode().Perm()&0o700 != 0o700 {
			if err := os.Chmod(d, fi.Mode().Perm()|0o700); err != nil {
				return err
			}
		}
		names, err := os.ReadDir(d)
		if err != nil {
			return err
		}
		for _, n := range names {
			p := filepath.ToSlash(filepath.Join(rel, n.Name()))
			if skipped[p] || skipped[p+"/"] {
				continue
			}
			full := filepath.Join(root, p)
			fi, err := os.Lstat(full)
			if err != nil {
				return err
			}
			mode := fi.Mode().Perm()
			switch {
			case fi.Mode()&os.ModeSymlink != 0:
				target, err := os.Readlink(full)
				if err != nil {
					return err
				}
				out[p] = "symlink -> " + target
			case fi.IsDir():
				out[p] = fmt.Sprintf("dir %04o", mode)
				if err := walk(p); err != nil {
					return err
				}
			default:
				if mode&0o400 == 0 {
					if err := os.Chmod(full, mode|0o400); err != nil {
						return err
					}
				}
				b, err := os.ReadFile(full)
				if err != nil {
					return err
				}
				out[p] = fmt.Sprintf("file %04o %q", mode, b)
			}
		}
		return nil
	}
	return out, walk("")
}

// treeProblems lists, in path order, every entry missing from, extra in or changed in the tree left.
func treeProblems(unsub *strings.Replacer, left, want map[string]string) []string {
	paths := []string{}
	for p := range left {
		paths = append(paths, p)
	}
	for p := range want {
		if _, ok := left[p]; !ok {
			paths = append(paths, p)
		}
	}
	sort.Strings(paths)
	var out []string
	for _, p := range paths {
		got, inLeft := left[p]
		w, inWant := want[p]
		switch {
		case !inWant:
			out = append(out, fmt.Sprintf("extra %s after the run: %s", p, unsub.Replace(got)))
		case !inLeft:
			out = append(out, fmt.Sprintf("missing %s after the run: want %s", p, unsub.Replace(w)))
		case got != w:
			out = append(out, fmt.Sprintf("changed %s after the run: %s, want %s", p, unsub.Replace(got), unsub.Replace(w)))
		}
	}
	return out
}

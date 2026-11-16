package modulemap

import (
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/lint"
)

// partDir is the root's single sub-directory named *-<part>.
func partDir(root, part string) (string, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return "", fail(3, "module-map: cannot list %s: %v", root, err)
	}
	var found []string
	for _, e := range entries {
		if n := e.Name(); strings.HasSuffix(n, "-"+part) && len(n) > len(part)+1 && lint.IsDir(filepath.Join(root, n)) {
			found = append(found, n)
		}
	}
	if len(found) != 1 {
		return "", fail(3, "module-map: want one *-%s part under %s, found %d", part, root, len(found))
	}
	return found[0], nil
}

// roleDir is the part's role folder for a new concern, relative to <part>/src: the shallowest Java package
// directory itself when it is the role folder (a ux module's route/), else its default role child.
func roleDir(src, role string) (string, error) {
	best, depth := "", -1
	err := filepath.WalkDir(src, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, p)
		if d.IsDir() && (rel == "rc" || strings.HasPrefix(d.Name(), ".")) {
			return filepath.SkipDir
		}
		if !d.IsDir() && strings.HasSuffix(p, ".java") {
			dir := filepath.ToSlash(filepath.Dir(rel))
			if n := strings.Count(dir, "/"); depth < 0 || n < depth || (n == depth && dir < best) {
				best, depth = dir, n
			}
		}
		return nil
	})
	if err != nil {
		return "", fail(3, "module-map: cannot walk %s: %v", src, err)
	}
	if depth < 0 {
		return "", fail(3, "module-map: no Java source under %s to place the concern beside", src)
	}
	if best == "." {
		return "", fail(3, "module-map: the shallowest Java source under %s is in the default package; move it into a package before adding a concern beside it", src)
	}
	if path.Base(best) == role {
		return best, nil
	}
	return best + "/" + role, nil
}

package modulemap

import (
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	mmrule "github.com/angeles725/niagara-tools/go/internal/lint/rules/modulemap"
)

// expand resolves a Tests cell to the files it names, relative to root: an exact entry as written, a glob by
// matching every regular file under root (dot directories skipped), in path order. An entry that is absolute or
// leaves root is refused (exit 3), as lint-module-map MM3 refuses it.
func expand(root string, cell []string) ([]string, error) {
	var all []string
	walked := false
	var out []string
	for _, e := range cell {
		// Mutation: MMT-contain -- expanding an entry outside the root runs (or lists) files the map does not own
		clean := path.Clean(e)
		if path.IsAbs(e) || clean == ".." || strings.HasPrefix(clean, "../") {
			return nil, fail(3, "module-map: Tests entry %s is not a relative path inside the module root (lint-module-map MM3)", e)
		}
		if !strings.ContainsAny(e, "*?[") {
			out = append(out, clean)
			continue
		}
		if !walked {
			walked = true
			err := filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if d.IsDir() && p != root && strings.HasPrefix(d.Name(), ".") {
					return filepath.SkipDir
				}
				if d.Type().IsRegular() {
					rel, _ := filepath.Rel(root, p)
					all = append(all, filepath.ToSlash(rel))
				}
				return nil
			})
			if err != nil {
				return nil, fail(3, "module-map: cannot walk %s: %v", root, err)
			}
			sort.Strings(all)
		}
		for _, f := range all {
			if mmrule.MatchGlob(e, f) {
				out = append(out, f)
			}
		}
	}
	return out, nil
}

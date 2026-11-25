package uxversion

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/vendorversion"
)

// groupVersion is the module version: the first uncommented defaultModuleVersion("X") (vendorversion.
// SourceVersion) of build.gradle.kts in root, then in each directory directly under it, then in each ancestor
// up to the gradle root (a directory holding settings.gradle.kts) or the repository root (.git). The group
// build file sits at the gradle root, which is the module root of build.sh (<repo>) or one level above the
// module directory report-module reads (<repo>/<MOD>). "" when none is found.
func groupVersion(root string) string {
	read := func(dir string) string {
		b, err := os.ReadFile(filepath.Join(dir, "build.gradle.kts"))
		if err != nil {
			return ""
		}
		return vendorversion.SourceVersion(string(b))
	}
	if v := read(root); v != "" {
		return v
	}
	if entries, err := os.ReadDir(root); err == nil {
		for _, e := range entries {
			if n := e.Name(); e.IsDir() && !strings.HasPrefix(n, ".") && !strings.HasSuffix(n, "-worktrees") {
				if v := read(filepath.Join(root, n)); v != "" {
					return v
				}
			}
		}
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return ""
	}
	for dir := abs; !holds(dir, "settings.gradle.kts") && !holds(dir, ".git"); {
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
		if v := read(dir); v != "" {
			return v
		}
	}
	return ""
}

// holds reports whether dir has an entry named name.
func holds(dir, name string) bool {
	_, err := os.Lstat(filepath.Join(dir, name))
	return err == nil
}

package uxversion

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/vendorversion"
)

// groupVersion is the module version: the first uncommented defaultModuleVersion("X") (vendorversion.
// SourceVersion) of build.gradle.kts in root, then in each directory directly under it, then in each ancestor
// up to the gradle root (a directory holding settings.gradle.kts) or the repository root (.git). The group
// build file sits at the gradle root, which is the module root of build.sh (<repo>) or one level above the
// module directory report-module reads (<repo>/<MOD>). Children that declare different versions are an error
// (the group is not decidable, never the first one); the same version in several is fine. "" when none is found.
func groupVersion(root string) (string, error) {
	read := func(dir string) string {
		b, err := os.ReadFile(filepath.Join(dir, "build.gradle.kts"))
		if err != nil {
			return ""
		}
		return vendorversion.SourceVersion(string(b))
	}
	if v := read(root); v != "" {
		return v, nil
	}
	first, firstDir := "", ""
	if entries, err := os.ReadDir(root); err == nil {
		for _, e := range entries {
			n := e.Name()
			if !e.IsDir() || strings.HasPrefix(n, ".") || strings.HasSuffix(n, "-worktrees") {
				continue
			}
			dir := strings.TrimSuffix(root, "/") + "/" + n
			v := read(dir)
			switch {
			case v == "":
			case first == "":
				first, firstDir = v, dir
			case v != first:
				// Mutation: UXV-ambiguous -- taking the first child's version judges a module against another group's
				return "", fmt.Errorf("ambiguous defaultModuleVersion: %s in %s, %s in %s", first, firstDir, v, dir)
			}
		}
	}
	if first != "" {
		return first, nil
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return "", nil
	}
	for dir := abs; !holds(dir, "settings.gradle.kts") && !holds(dir, ".git"); {
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", nil
		}
		dir = parent
		if v := read(dir); v != "" {
			return v, nil
		}
	}
	return "", nil
}

// holds reports whether dir has an entry named name.
func holds(dir, name string) bool {
	_, err := os.Lstat(filepath.Join(dir, name))
	return err == nil
}

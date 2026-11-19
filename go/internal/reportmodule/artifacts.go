package reportmodule

import (
	"fmt"
	"path/filepath"
	"strings"
)

// The artifact set of a module root, shared by the lint run, --ready and build.sh (--schema-baseline-ref and
// its INCOMPLETE cross-reference): one definition, so a docs/ folder beside the parts is never walked as an
// artifact by one tool and skipped by another. [ev: retro client-2.3.0-big-upgrade K4]

// IsModuleArtifact says whether dir, a directory of the module root, is one of its artifacts: it holds a
// module-include.xml, or its name is a module part (<MOD>-rt/-ux/-wb/-se/-doc) and it holds its own
// <name>.gradle.kts or a build.gradle[.kts]. A docs/ directory or a gradle directory beside the parts (buildSrc/, a
// shared conventions build) is not one.
func IsModuleArtifact(dir string) bool {
	if isFile(dir + "/module-include.xml") {
		return true
	}
	name := filepath.Base(dir)
	if !isPartName(name) {
		return false
	}
	for _, f := range []string{name + ".gradle.kts", "build.gradle.kts", "build.gradle"} {
		if isFile(dir + "/" + f) {
			return true
		}
	}
	return false
}

// isModuleArtifact is IsModuleArtifact of a discovered artifact.
func isModuleArtifact(a artifact) bool { return IsModuleArtifact(a.dir) }

// isPartName says whether name ends with a Niagara module part suffix.
func isPartName(name string) bool {
	for _, p := range []string{"-rt", "-ux", "-wb", "-se", "-doc"} {
		if strings.HasSuffix(name, p) && len(name) > len(p) {
			return true
		}
	}
	return false
}

// SplitArtifacts parts the walked directories of a module root (in their order) into its artifacts and the
// other directories. When none of them reads as an artifact, all of them are kept as artifacts and none is
// other: the walk stays what it was (every directory) rather than dropping a whole module the predicate cannot
// recognize.
func SplitArtifacts(dirs []string) (artifacts, other []string) {
	for _, d := range dirs {
		if IsModuleArtifact(d) {
			artifacts = append(artifacts, d)
		} else {
			other = append(other, d)
		}
	}
	if len(artifacts) == 0 {
		return dirs, nil
	}
	return artifacts, other
}

// NotArtifactDetail is the reason a directory beside the module parts is not walked as an artifact.
const NotArtifactDetail = "not a module artifact (no module-include.xml, not a module part with its build file)"

// lintArtifacts is the lint run's artifact set out of the discovered directories: SplitArtifacts, each other
// directory one uncounted INFO line on out.
func lintArtifacts(r *report, found []artifact) []artifact {
	dirs := make([]string, len(found))
	for i, a := range found {
		dirs[i] = a.dir
	}
	keep, other := SplitArtifacts(dirs)
	for _, d := range other {
		fmt.Fprintf(r.out, "%s  INFO  artifact  %s: skipped\n", filepath.Base(d), NotArtifactDetail)
	}
	kept := map[string]bool{}
	for _, d := range keep {
		kept[d] = true
	}
	var out []artifact
	for _, a := range found {
		if kept[a.dir] {
			out = append(out, a)
		}
	}
	return out
}

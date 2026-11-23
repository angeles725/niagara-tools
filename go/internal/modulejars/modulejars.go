// Package modulejars is the one reader of an installation's module list: the jars of <niagara_home>/modules.
// The mirror of an installation (mirrorhome), the jar-lock check (preflight) and the template lint
// (lint/rules/ntplmanifest) each listed that directory on their own, with their own rule for dot files and
// links; they share this listing and keep only the predicate that is theirs. Stdlib only, a leaf package.
package modulejars

import (
	"os"
	"strings"
)

// Dir is the directory of an installation that holds one <module>.jar per module part it ships.
const Dir = "modules"

// Suffix ends the name of a module jar.
const Suffix = ".jar"

// Names are the entries of dir whose name ends in .jar and does not start with a dot, in name order. A link, a
// dangling link and a directory of that name count: use Regular for the jars that can be read. When dir cannot
// be listed the error is os.ReadDir's, with the names it did list (none for a missing dir): a list that could
// not be read is never the same as an empty one.
func Names(dir string) ([]string, error) {
	held, err := os.ReadDir(dir)
	var names []string
	for _, e := range held {
		if name := e.Name(); strings.HasSuffix(name, Suffix) && !strings.HasPrefix(name, ".") {
			names = append(names, name)
		}
	}
	return names, err
}

// Regular are the Names that are a regular file, a link followed (test -f): a dangling link and a directory
// named x.jar ship nothing.
func Regular(dir string) ([]string, error) {
	names, err := Names(dir)
	var files []string
	for _, name := range names {
		if st, statErr := os.Stat(dir + "/" + name); statErr == nil && st.Mode().IsRegular() {
			// Mutation: NTPL-jar-regular -- counting a jar by its name alone ships a module from a dangling link or a directory
			files = append(files, name)
		}
	}
	return files, err
}

package puretest

import (
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/findwalk"
	"github.com/angeles725/niagara-tools/go/internal/lint"
)

// findJar is `find -H <root> -name <name>`: every path of an entry called name under root, in name order
// (find took directory order), none when there is no such entry. A root that is a symlink is followed, as
// find -H does, so a relocated or shared Gradle cache is read like a directory.
func findJar(root, name string) []string {
	var found []string
	findwalk.Walk(root, true, findwalk.NoLimit, func(e findwalk.Entry) bool {
		if e.Name == name {
			found = append(found, e.Path)
		}
		return true
	}, nil)
	return found
}

// sourceRel is the source location of <pkg.TestClass> below a source root: the file path without the
// extension (the packages as a path, then the class name). A class in the default package sits directly
// under the root.
func sourceRel(fqcn string) string {
	i := strings.LastIndex(fqcn, ".")
	if i < 0 {
		return fqcn
	}
	return strings.ReplaceAll(fqcn[:i], ".", "/") + "/" + fqcn[i+1:]
}

// sourceRoot is the first test root under the module dir rt that holds the test's source file rel + ".java",
// "" when none does.
func sourceRoot(rt, rel string) string {
	for _, root := range testRoots {
		if lint.IsFile(rt + "/" + root + "/" + rel + ".java") {
			return rt + "/" + root
		}
	}
	return ""
}

package puretest

import (
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/findwalk"
	"github.com/angeles725/niagara-tools/go/internal/lint"
)

// findJar is `find <root> -name <name>`, first hit: the path of an entry called name under root, "" when
// there is none. A root that is a symlink is not followed, as find does not. Where find took the first
// hit in directory order, this takes the first in name order.
func findJar(root, name string) string {
	found := ""
	findwalk.Walk(root, false, findwalk.NoLimit, func(e findwalk.Entry) bool {
		if found == "" && e.Name == name {
			found = e.Path
		}
		return found == ""
	}, nil)
	return found
}

// splitClass is the source location of <pkg.TestClass> below a source root: the packages as a path and
// the class name. A class without a package is looked up in a directory of its own name, as the bash
// tool's ${fqcn%.*} left it.
func splitClass(fqcn string) (pkgPath, simple string) {
	pkg := fqcn
	if i := strings.LastIndex(fqcn, "."); i >= 0 {
		pkg, simple = fqcn[:i], fqcn[i+1:]
	} else {
		simple = fqcn
	}
	return strings.ReplaceAll(pkg, ".", "/"), simple
}

// sourceRoot is the first test root under the module dir rt that holds the test's source file, "" when
// none does.
func sourceRoot(rt, pkgPath, simple string) string {
	for _, root := range testRoots {
		if lint.IsFile(rt + "/" + root + "/" + pkgPath + "/" + simple + ".java") {
			return rt + "/" + root
		}
	}
	return ""
}

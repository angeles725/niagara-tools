package puretest

import (
	"os"
	"regexp"
	"strings"
)

// JUnit loads a class by the name its source's package line gives, not by the path the source sits under: a
// name built from the path (test.com.x.FooTest for srcTest/test/com/x/FooTest.java) compiles, then JUnit
// reports only initializationError(JUnitCommandLineParseResult) (issue #1115).

const (
	msgNoClass    = "class %s not found; the package line of %s says %s"
	noPackageText = "(none)" // what the message says for a source without a package line
)

// packageLine is the package declaration, on a line of its own start.
var packageLine = regexp.MustCompile(`(?m)^[ \t]*package[ \t]+([A-Za-z0-9_.$]+)[ \t]*;`)

// packageOf is the package a source file declares, "" for the default package or a file it cannot read.
func packageOf(file string) string {
	src, err := os.ReadFile(file)
	if err != nil {
		return ""
	}
	if m := packageLine.FindSubmatch(src); m != nil {
		return string(m[1])
	}
	return ""
}

// classOf is the name JUnit loads for the test source file: its package line and the file's own name.
func classOf(file string) string {
	name := file[strings.LastIndex(file, "/")+1:]
	name = strings.TrimSuffix(name, ".java")
	if pkg := packageOf(file); pkg != "" {
		return pkg + "." + name
	}
	return name
}

// checkPackage is exit 3 when class is not the name the package line of file gives.
func (r *run) checkPackage(file, class string) int {
	if classOf(file) == class {
		return ExitOK
	}
	pkg := packageOf(file)
	if pkg == "" {
		pkg = noPackageText
	}
	return r.die(ExitEnv, msgNoClass, class, file, pkg)
}

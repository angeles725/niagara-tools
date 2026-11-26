package puretest

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/lint"
)

// run is one invocation: where its messages and its children's output go.
type run struct{ s cli.Stdio }

// Main implements `run-pure-test <module-rt-dir> <pkg.TestClass>` and `run-pure-test --all <module-rt-dir>`.
func Main(args []string, s cli.Stdio) int {
	r := &run{s}
	if len(args) == 2 && args[0] == flagAll {
		return r.all(args[1])
	}
	if len(args) != 2 {
		return r.die(ExitUsage, msgUsage)
	}
	t, code := r.tools(args[0])
	if code != ExitOK {
		return code
	}
	class := args[1]
	rel := sourceRel(class)
	root := sourceRoot(t.rt, rel)
	if root == "" {
		return r.die(ExitEnv, msgNoSource, t.rt, rel)
	}
	file := root + "/" + rel + ".java"
	if code := r.checkPackage(file, class); code != ExitOK {
		return code
	}
	return r.build(t, sourcePath(t.rt, []string{root}, s.Env(kitJavaEnv)), []string{file}, class)
}

// toolset is what a run needs from its host: the module part (absolute) and the jars it compiles and runs with.
type toolset struct {
	rt                string
	junits, hamcrests []string
}

// tools checks the module dir, javac and the two jars, and makes the part dir absolute: the compile and the
// JVM run in different directories, so a relative one named another place for each (issue #1115).
func (r *run) tools(rt string) (toolset, int) {
	if !lint.IsDir(rt) {
		return toolset{}, r.die(ExitEnv, msgNoModuleDir, rt)
	}
	if !onPath(javac) {
		return toolset{}, r.die(ExitEnv, msgNoJavac)
	}
	cache := r.s.Env("HOME") + "/" + gradleDir
	t := toolset{rt, findJar(cache, junitJar), findJar(cache, hamcrestJar)}
	if abs, err := filepath.Abs(rt); err == nil {
		t.rt = abs
	}
	if len(t.junits) == 0 {
		return t, r.die(ExitEnv, msgNoJUnit)
	}
	if len(t.hamcrests) == 0 {
		return t, r.die(ExitEnv, msgNoHamcrest)
	}
	return t, ExitOK
}

// build compiles the test sources (and the adapters) into a temporary directory and runs JUnitCore on classes.
func (r *run) build(t toolset, sourcepath string, sources []string, classes ...string) int {
	home := r.s.Env(homeEnv)
	if code := r.checkHome(home); code != ExitOK {
		return code
	}
	r.noticeCopies(junitJar, t.junits)
	r.noticeCopies(hamcrestJar, t.hamcrests)
	// The classes go to a temporary directory: the module tree may be another session's working tree.
	tmp, err := os.MkdirTemp("", tempPattern)
	if err != nil {
		return r.die(exitNoChdir, msgNoTemp, err)
	}
	defer os.RemoveAll(tmp)
	defer removeOnSignal(tmp)()
	if r.compile(tmp, t.junits[0], sourcepath, sources) != 0 {
		// Exit 1 is kept for a JUnit failure: a test that does not compile proves nothing.
		return r.die(ExitEnv, msgNoCompile, strings.Join(classes, " "))
	}
	if code := r.compileAdapters(t.rt, tmp, home); code != ExitOK {
		return code
	}
	// From the module dir, so a test that reads src/… by a relative path finds it wherever the tool was called.
	args := append([]string{"-cp", tmp + ":" + t.junits[0] + ":" + t.hamcrests[0], runner}, classes...)
	return r.exec(t.rt, java, args...)
}

// sourcePath is the module's pure sources, its test roots and, when the launcher names it, the kit's Java
// helpers (toolbelt/java: n4kit.series), so a module test uses them without copying them in.
func sourcePath(rt string, roots []string, kitJava string) string {
	path := rt + "/" + mainSources
	for _, root := range roots {
		path += ":" + root
	}
	if kitJava != "" {
		path += ":" + kitJava
	}
	return path
}

// compile runs javac on the test source; -sourcepath brings in the pure classes it references, compiled
// together so package-private members stay reachable.
func (r *run) compile(out, junit, sourcepath string, sources []string) int {
	args := append([]string{}, compileFlags...)
	args = append(args, "-cp", junit, "-sourcepath", sourcepath, "-d", out)
	args = append(args, sources...)
	return r.exec("", javac, args...)
}

// noticeCopies says which copy of a jar is used when the cache holds more than one (the first in name
// order, so the choice is the same on every host).
func (r *run) noticeCopies(name string, found []string) {
	if len(found) > 1 {
		fmt.Fprintf(r.s.Err, prefix+msgSeveralJars+"\n", name, len(found), found[0])
	}
}

// die prints a message on stderr and returns code.
func (r *run) die(code int, format string, a ...any) int {
	fmt.Fprintf(r.s.Err, prefix+format+"\n", a...)
	return code
}

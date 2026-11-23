package puretest

import (
	"fmt"
	"os"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/lint"
)

// run is one invocation: where its messages and its children's output go.
type run struct{ s cli.Stdio }

// Main implements `run-pure-test <module-rt-dir> <pkg.TestClass>`.
func Main(args []string, s cli.Stdio) int {
	r := &run{s}
	if len(args) != 2 {
		return r.die(ExitUsage, msgUsage)
	}
	rt, class := args[0], args[1]
	if !lint.IsDir(rt) {
		return r.die(ExitEnv, msgNoModuleDir, rt)
	}
	if !onPath(javac) {
		return r.die(ExitEnv, msgNoJavac)
	}
	cache := s.Env("HOME") + "/" + gradleDir
	junits, hamcrests := findJar(cache, junitJar), findJar(cache, hamcrestJar)
	if len(junits) == 0 {
		return r.die(ExitEnv, msgNoJUnit)
	}
	if len(hamcrests) == 0 {
		return r.die(ExitEnv, msgNoHamcrest)
	}
	junit, hamcrest := junits[0], hamcrests[0]
	rel := sourceRel(class)
	root := sourceRoot(rt, rel)
	if root == "" {
		return r.die(ExitEnv, msgNoSource, rt, rel)
	}
	r.noticeCopies(junitJar, junits)
	r.noticeCopies(hamcrestJar, hamcrests)
	// The classes go to a temporary directory: the module tree may be another session's working tree.
	tmp, err := os.MkdirTemp("", tempPattern)
	if err != nil {
		return r.die(exitNoChdir, msgNoTemp, err)
	}
	defer os.RemoveAll(tmp)
	defer removeOnSignal(tmp)()
	if r.compile(tmp, junit, sourcePath(rt, root, s.Env(kitJavaEnv)), root+"/"+rel+".java") != 0 {
		// Exit 1 is kept for a JUnit failure: a test that does not compile proves nothing.
		return r.die(ExitEnv, msgNoCompile, class)
	}
	// From the module dir, so a test that reads src/… by a relative path finds it wherever the tool was called.
	return r.exec(rt, java, "-cp", tmp+":"+junit+":"+hamcrest, runner, class)
}

// sourcePath is the module's pure sources, its test root and, when the launcher names it, the kit's Java
// helpers (toolbelt/java: n4kit.series), so a module test uses them without copying them in.
func sourcePath(rt, root, kitJava string) string {
	path := rt + "/" + mainSources + ":" + root
	if kitJava != "" {
		path += ":" + kitJava
	}
	return path
}

// compile runs javac on the test source; -sourcepath brings in the pure classes it references, compiled
// together so package-private members stay reachable.
func (r *run) compile(out, junit, sourcepath, source string) int {
	args := append([]string{}, compileFlags...)
	args = append(args, "-cp", junit, "-sourcepath", sourcepath, "-d", out, source)
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

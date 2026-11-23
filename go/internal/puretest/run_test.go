package puretest

import (
	"os"
	"strings"
	"testing"
)

// TestUsage: exactly two arguments, else exit 2.
func TestUsage(t *testing.T) {
	f := newFixture(t)
	for _, args := range [][]string{nil, {f.rt}, {f.rt, testClass, "extra"}} {
		f.stops("usage", ExitUsage, "usage: run-pure-test.sh <module-rt-dir> <pkg.TestClass>", args...)
	}
}

// TestEnvironment: each missing piece is exit 3 with its own message, before anything is compiled.
func TestEnvironment(t *testing.T) {
	f := newFixture(t)
	f.stops("no module dir", ExitEnv, "module-rt-dir not found: "+f.dir+"/nope", f.dir+"/nope", testClass)
	f.stops("the module dir is a file", ExitEnv, "module-rt-dir not found: "+f.junit, f.junit, testClass)
	f.stops("no test source", ExitEnv, "test source not found: "+f.rt+"/srcTest[/test]/com/example/demo/NopeTest.java", f.rt, "com.example.demo.NopeTest")
	f.stops("a class without a package is looked up in the test root itself", ExitEnv, "test source not found: "+f.rt+"/srcTest[/test]/Bar.java", f.rt, "Bar")

	os.Remove(f.hamcrest)
	f.stops("no hamcrest", ExitEnv, "hamcrest-core-1.3.jar not in ~/.gradle — run one gradle build to fetch it", f.rt, testClass)
	os.Remove(f.junit)
	f.stops("no junit", ExitEnv, "junit-4.13.2.jar not in ~/.gradle — run one './gradlew :<mod>-rt:compileTestJava' to fetch it", f.rt, testClass)

	f = newFixture(t)
	os.Remove(f.dir + "/bin/javac")
	f.stops("no javac", ExitEnv, "javac not on PATH (need a JDK 8)", f.rt, testClass)
}

// TestPass: the test is compiled into a temporary directory with the pure sources on the source path, run
// with JUnitCore from the module dir (a source-structural test reads src/… from there), and exit 0 is
// the tests' own; the temporary directory is gone afterwards and the module tree is unchanged.
func TestPass(t *testing.T) {
	f := newFixture(t)
	t.Chdir(f.dir + "/tmp") // any directory but the module's
	before, _ := os.ReadDir(f.rt)
	if code := f.run(f.rt, testClass); code != ExitOK || f.out != "javac stdout\njava stdout\n" || f.err != "javac stderr\njava stderr\n" {
		t.Errorf("exit %d stdout %q stderr %q", code, f.out, f.err)
	}
	tmp := f.tempDir()
	wantJavac := f.dir + "/tmp <-source> <8> <-target> <8> <-nowarn> <-cp> <" + f.junit + "> <-sourcepath> <" + f.rt + "/src:" + f.rt + "/srcTest/test> <-d> <" +
		tmp + "> <" + f.rt + "/srcTest/test/com/example/demo/AdderTest.java>\n"
	if got := f.call("javac"); got != wantJavac {
		t.Errorf("javac call\n%s\nwant\n%s", got, wantJavac)
	}
	wantJava := f.rt + " <-cp> <" + tmp + ":" + f.junit + ":" + f.hamcrest + "> <org.junit.runner.JUnitCore> <" + testClass + ">\n"
	if got := f.call("java"); got != wantJava {
		t.Errorf("java call\n%s\nwant\n%s", got, wantJava)
	}
	if !strings.HasPrefix(tmp, f.dir+"/tmp/") {
		t.Errorf("compile output %q is not under TMPDIR", tmp)
	}
	if _, err := os.Stat(tmp); err == nil {
		t.Errorf("the temporary directory %s was left behind", tmp)
	}
	if after, _ := os.ReadDir(f.rt); len(after) != len(before) {
		t.Errorf("the module dir changed: %v", after)
	}
}

// TestAClassWithoutAPackageRuns: a test in the default package sits directly under the test root, and is the
// source javac gets (retro go-toolbelt-small-port Δ3).
func TestAClassWithoutAPackageRuns(t *testing.T) {
	f := newFixture(t)
	f.write("rt/srcTest/test/PlainTest.java", "")
	if code := f.run(f.rt, "PlainTest"); code != ExitOK {
		t.Fatalf("exit %d stderr %q", code, f.err)
	}
	if got := f.call("javac"); !strings.HasSuffix(got, "<"+f.rt+"/srcTest/test/PlainTest.java>\n") {
		t.Errorf("javac call %s", got)
	}
	if got := f.call("java"); !strings.HasSuffix(got, "<org.junit.runner.JUnitCore> <PlainTest>\n") {
		t.Errorf("java call %s", got)
	}
}

// TestASymlinkedGradleCacheIsFollowed: ~/.gradle that is a link to the real cache (a shared or relocated cache)
// supplies both jars, as a directory would; nothing is said about it.
func TestASymlinkedGradleCacheIsFollowed(t *testing.T) {
	f := newFixture(t)
	if err := os.Rename(f.dir+"/home/.gradle", f.dir+"/cache"); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(f.dir+"/cache", f.dir+"/home/.gradle"); err != nil {
		t.Fatal(err)
	}
	if code := f.run(f.rt, testClass); code != ExitOK || f.err != "javac stderr\njava stderr\n" {
		t.Fatalf("exit %d stderr %q", code, f.err)
	}
	if got := f.call("java"); !strings.Contains(got, ":"+f.dir+"/home/.gradle/caches/a/junit-4.13.2.jar:") {
		t.Errorf("java call %s", got)
	}
}

// TestSeveralCopiesOfAJarNameTheOneUsed: the first copy in name order is used and a NOTICE on stderr names it
// and the count, for each jar that has more than one; a jar with one copy gets no notice.
func TestSeveralCopiesOfAJarNameTheOneUsed(t *testing.T) {
	f := newFixture(t)
	f.write("home/.gradle/caches/b/junit-4.13.2.jar", "")
	f.write("home/.gradle/caches/c/junit-4.13.2.jar", "")
	f.write("home/.gradle/wrapper/a/hamcrest-core-1.3.jar", "")
	if code := f.run(f.rt, testClass); code != ExitOK {
		t.Fatalf("exit %d stderr %q", code, f.err)
	}
	cache := f.dir + "/home/.gradle"
	want := "run-pure-test: NOTICE junit-4.13.2.jar found 3 times under ~/.gradle, using " + cache + "/caches/a/junit-4.13.2.jar\n" +
		"run-pure-test: NOTICE hamcrest-core-1.3.jar found 2 times under ~/.gradle, using " + cache + "/wrapper/a/hamcrest-core-1.3.jar\n" +
		"javac stderr\njava stderr\n"
	if f.err != want {
		t.Errorf("stderr %q, want %q", f.err, want)
	}
	if got := f.call("java"); !strings.Contains(got, "<"+f.tempDir()+":"+cache+"/caches/a/junit-4.13.2.jar:"+cache+"/wrapper/a/hamcrest-core-1.3.jar>") {
		t.Errorf("java call %s", got)
	}
}

// TestSourceRoot: srcTest/test/ wins over srcTest/; srcTest/ is used when only it holds the test.
func TestSourceRoot(t *testing.T) {
	f := newFixture(t)
	f.write("rt/srcTest/com/example/demo/AdderTest.java", "")
	f.run(f.rt, testClass)
	if got := f.call("javac"); !strings.Contains(got, "<"+f.rt+"/src:"+f.rt+"/srcTest/test> ") {
		t.Errorf("both roots: javac call %s", got)
	}
	f = newFixture(t)
	f.write("rt/srcTest/com/example/demo/AdderTest.java", "")
	os.RemoveAll(f.rt + "/srcTest/test")
	f.run(f.rt, testClass)
	if got := f.call("javac"); !strings.HasSuffix(got, "<-sourcepath> <"+f.rt+"/src:"+f.rt+"/srcTest> <-d> <"+f.tempDir()+"> <"+f.rt+"/srcTest/com/example/demo/AdderTest.java>\n") {
		t.Errorf("srcTest only: javac call %s", got)
	}
}

// TestKitJavaOnSourcePath (O12): the kit's Java helper root the launcher names joins the javac source path
// after the module's roots; unset, the source path is the module's alone (TestPass, TestSourceRoot).
func TestKitJavaOnSourcePath(t *testing.T) {
	f := newFixture(t)
	f.kitJava = f.dir + "/kit/java"
	f.run(f.rt, testClass)
	if got := f.call("javac"); !strings.Contains(got, "<-sourcepath> <"+f.rt+"/src:"+f.rt+"/srcTest/test:"+f.dir+"/kit/java> ") {
		t.Errorf("javac call %s", got)
	}
}

// TestFailingTestIsExitOne: the runner's status is the tool's: 1 is a failed test, never swallowed.
func TestFailingTestIsExitOne(t *testing.T) {
	for _, code := range []int{1, 7} {
		f := newFixture(t)
		f.tool("java", code)
		if got := f.run(f.rt, testClass); got != code || f.out != "javac stdout\njava stdout\n" {
			t.Errorf("java exits %d: exit %d stdout %q", code, got, f.out)
		}
		if _, err := os.Stat(f.tempDir()); err == nil {
			t.Errorf("java exits %d: the temporary directory was left behind", code)
		}
	}
}

// TestCompileErrorIsNotARed (P7): a test that does not compile is exit 3, never the exit 1 of a failed
// test, and it is not run.
func TestCompileErrorIsNotARed(t *testing.T) {
	f := newFixture(t)
	f.tool("javac", 1)
	want := "javac stderr\nrun-pure-test: " + testClass + " does not compile (javac failed above) — fix the test or the pure sources; a compile error is not a RED\n"
	if code := f.run(f.rt, testClass); code != ExitEnv || f.err != want {
		t.Errorf("exit %d stderr %q, want 3 and %q", code, f.err, want)
	}
	if f.call("java") != "" {
		t.Errorf("java ran: %s", f.call("java"))
	}
	if _, err := os.Stat(f.tempDir()); err == nil {
		t.Error("the temporary directory was left behind")
	}
}

// TestNoJava: a JDK without java on PATH ends as a shell does for a missing command, 127.
func TestNoJava(t *testing.T) {
	f := newFixture(t)
	os.Remove(f.dir + "/bin/java")
	if code := f.run(f.rt, testClass); code != exitNotFound || f.err != "javac stderr\nrun-pure-test: java: command not found\n" {
		t.Errorf("exit %d stderr %q", code, f.err)
	}
}

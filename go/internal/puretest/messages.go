// Package puretest runs one pure-logic JUnit 4 test of a module part outside gradle, as
// toolbelt/run-pure-test.sh does: the test and the pure sources it references are compiled into a
// temporary directory (nothing is written in the module tree) and run with JUnitCore from the part
// directory. The header of that script has the why.
package puretest

// Exit codes of run-pure-test.sh (unchanged from the bash tool; its header lists them).
const (
	ExitOK     = 0 // the tests passed
	ExitFailed = 1 // a test failed: the RED a test-first step looks for
	ExitUsage  = 2 // usage
	ExitEnv    = 3 // environment: no module dir, no JDK, no JUnit in the gradle cache, no test source, or it does not compile
)

// What bash returns for a command it cannot run, or that a signal killed.
const (
	exitNoChdir       = 1
	exitNotExecutable = 126
	exitNotFound      = 127
	signalBase        = 128
)

// The tools and jars a run needs. JUnit and Hamcrest live in the gradle cache or distribution under
// $HOME/.gradle, not in the repository or the Niagara install.
const (
	javac       = "javac"
	java        = "java"
	gradleDir   = ".gradle"
	junitJar    = "junit-4.13.2.jar"
	hamcrestJar = "hamcrest-core-1.3.jar"
	runner      = "org.junit.runner.JUnitCore"
	mainSources = "src"                    // under the module part: the pure sources the test references
	tempPattern = "tmp.*"                  // the compile output directory, removed when the run ends
	kitJavaEnv  = "RUN_PURE_TEST_KIT_JAVA" // set by run-pure-test.sh: the kit's Java helper source root
	homeEnv     = "NIAGARA_HOME"           // the mirror (or install) whose modules/ jars the adapters compile against
)

// compileFlags make javac emit Java 8 classes without the warnings a newer JDK adds about it.
var compileFlags = []string{"-source", "8", "-target", "8", "-nowarn"}

// testRoots are the test source roots under the module part, tried in order: the package sits under an
// extra test/ segment in the kit's fixtures, or directly under srcTest/.
var testRoots = []string{"srcTest/test", "srcTest"}

// Messages (fmt formats); each is printed on stderr after "run-pure-test: ".
const (
	prefix         = "run-pure-test: "
	msgUsage       = "usage: run-pure-test.sh <module-rt-dir> <pkg.TestClass>"
	msgNoModuleDir = "module-rt-dir not found: %s"
	msgNoJavac     = "javac not on PATH (need a JDK 8)"
	msgNoJUnit     = junitJar + " not in ~/.gradle — run one './gradlew :<mod>-rt:compileTestJava' to fetch it"
	msgNoHamcrest  = hamcrestJar + " not in ~/.gradle — run one gradle build to fetch it"
	msgNoSource    = "test source not found: %s/srcTest[/test]/%s.java"
	msgSeveralJars = "NOTICE %s found %d times under ~/.gradle, using %s"
	msgNoTemp      = "cannot make a temporary directory: %v"
	msgNoCompile   = "%s does not compile (javac failed above) — fix the test or the pure sources; a compile error is not a RED"
	msgNoCommand   = "%s: command not found"
	msgCannotRun   = "%s: %s"
)

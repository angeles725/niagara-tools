package buildmodule

import (
	"os"
	"testing"
	"time"
)

// A signal sent to build.sh's pid alone does not reach gradle (retro p1-build-deploy-safety Δ4): the handler
// passes it on to the running gradle and waits for it to end before it puts modules/ back, so a gradle that
// would install its jar after the restore never gets the chance. The fake gradle stops on the signal; left
// running, it installs the new jar two seconds later.
func TestASignalReachesTheRunningGradleBeforeTheRestore(t *testing.T) {
	f := newFixture(t)
	f.signs()
	f.jar(f.deployed("rt"), "1.0.0", "old")
	f.jar(f.built("rt"), "1.0.1", "new")
	before := f.read(f.deployed("rt"))
	code := -1
	exitProcess = func(c int) { code = c }
	t.Cleanup(func() { exitProcess = os.Exit })
	f.does("gradlew", "trap 'exit 1' INT TERM HUP\nkill -TERM $PPID\n"+
		"for _ in $(seq 40); do sleep 0.05; done\n"+
		`cp "`+f.built("rt")+`" "`+f.deployed("rt")+`"`+"\n")
	f.build()
	if code != 143 {
		t.Errorf("exit %d, want 143\nstderr: %s", code, f.err)
	}
	if f.read(f.deployed("rt")) != before {
		t.Errorf("modules/Foo-rt.jar is the jar gradle installed after the restore, not the pre-build one\nstderr: %s", f.err)
	}
}

// A gradle that ignores the passed-on signal is killed once gradleStopWait has passed, still before the
// restore, so it cannot install its jar afterwards either.
func TestAGradleThatIgnoresTheSignalIsKilledBeforeTheRestore(t *testing.T) {
	f := newFixture(t)
	f.signs()
	f.jar(f.deployed("rt"), "1.0.0", "old")
	f.jar(f.built("rt"), "1.0.1", "new")
	before := f.read(f.deployed("rt"))
	code := -1
	exitProcess = func(c int) { code = c }
	wait := gradleStopWait
	gradleStopWait = 300 * time.Millisecond
	t.Cleanup(func() { exitProcess, gradleStopWait = os.Exit, wait })
	f.does("gradlew", "trap '' INT TERM HUP\nkill -TERM $PPID\n"+
		"for _ in $(seq 40); do sleep 0.05; done\n"+
		`cp "`+f.built("rt")+`" "`+f.deployed("rt")+`"`+"\n")
	start := time.Now()
	f.build()
	if took := time.Since(start); took > 1500*time.Millisecond { // left running, the fake gradle takes 2 s
		t.Errorf("the run took %v: gradle was waited for, not killed", took)
	}
	if code != 143 {
		t.Errorf("exit %d, want 143\nstderr: %s", code, f.err)
	}
	if f.read(f.deployed("rt")) != before {
		t.Errorf("modules/Foo-rt.jar is the jar the unkilled gradle installed\nstderr: %s", f.err)
	}
}

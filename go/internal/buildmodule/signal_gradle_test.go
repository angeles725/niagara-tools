package buildmodule

import (
	"errors"
	"os"
	"os/exec"
	"strings"
	"syscall"
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
	for _, text := range []string{
		"build.sh: interrupted (SIGTERM) before the gate passed; putting modules/ back as it was before the build",
		"  Restored the previously-deployed Foo-rt.jar into " + f.home + "/modules.",
	} {
		if strings.Count(f.err, text) != 1 {
			t.Errorf("stderr should carry %q once:\n%s", text, f.err)
		}
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

// Once a signal arrived, no gradle starts: one that would start after the handler looked finds the run
// stopping, so it cannot install a jar after the restore either.
func TestNoGradleStartsOnceASignalArrived(t *testing.T) {
	b := &build{}
	b.stopGradle(syscall.SIGTERM)
	marker := t.TempDir() + "/ran"
	if err := b.runTracked(exec.Command("touch", marker)); !errors.Is(err, errInterrupted) {
		t.Errorf("runTracked after a signal: %v, want errInterrupted", err)
	}
	if _, err := os.Stat(marker); err == nil {
		t.Error("gradle was started after the signal")
	}
}

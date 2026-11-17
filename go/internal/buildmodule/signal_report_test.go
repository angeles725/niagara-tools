package buildmodule

import (
	"os"
	"testing"
	"time"
)

// A signal while report-module runs comes after the gate passed: the jar in modules/ is a verified one, so the
// handler leaves it in place (only the window from the baseline snapshot to a passing gate is undone).
func TestASignalAfterTheGatePassedKeepsTheVerifiedJar(t *testing.T) {
	f := newFixture(t)
	f.signs()
	f.jar(f.deployed("rt"), "1.0.0", "old")
	f.jar(f.built("rt"), "1.0.1", "new")
	code := -1
	exitProcess = func(c int) { code = c }
	t.Cleanup(func() { exitProcess = os.Exit })
	f.does("gradlew", `cp "`+f.built("rt")+`" "`+f.deployed("rt")+`"`+"\n")
	f.does("report-module.sh", "kill -TERM $PPID\nsleep 0.5\n")
	f.build()
	installed := f.read(f.built("rt"))
	if code != 143 {
		t.Errorf("exit %d, want 143\nstderr: %s", code, f.err)
	}
	if f.read(f.deployed("rt")) != installed {
		t.Errorf("modules/Foo-rt.jar was rolled back although the gate had passed\nstderr: %s", f.err)
	}
}

// A gradle that ends while a child of its own still holds the output open is waited for, as before #795; the
// bound on waiting belongs to the signal handler only (verifier probe P9 of #795: WaitDelay failed this build).
func TestAChildHoldingGradlesOutputDoesNotFailTheBuild(t *testing.T) {
	f := newFixture(t)
	f.signs()
	f.jar(f.built("rt"), "1.0.1", "new")
	wait := gradleStopWait
	gradleStopWait = 200 * time.Millisecond
	t.Cleanup(func() { gradleStopWait = wait })
	f.does("gradlew", "(sleep 0.8) &\n")
	if code := f.build(); code != 0 {
		t.Errorf("exit %d, want 0\nstdout: %s\nstderr: %s", code, f.out, f.err)
	}
}

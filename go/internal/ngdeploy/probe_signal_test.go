package ngdeploy

import (
	"os"
	"os/signal"
	"strings"
	"syscall"
	"testing"
)

// A signal held by the probe is honoured however the probe goes on: one that arrives while it looks at the
// last jar, which the station does not have yet (nothing to move, no check after a move), still ends the run
// with 128 + the signal before the build, instead of being dropped when the probe stops holding signals
// (review T15 R3/R4 probe-signal-dropped).
func TestASignalWhileTheProbeLooksAtAJarItDoesNotMoveEndsTheRun(t *testing.T) {
	for _, sig := range []syscall.Signal{syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP} {
		f := newFixture(t)
		upgrade(t, f)
		if err := os.Remove(f.deployedJar("ux")); err != nil {
			t.Fatal(err)
		}
		own := make(chan os.Signal, 1)
		signal.Notify(own, sig) // keeps the signal from ending the test binary
		probeRegular = func(path string) (bool, error) {
			if strings.HasSuffix(path, "/demo-ux.jar") {
				if err := syscall.Kill(os.Getpid(), sig); err != nil {
					t.Fatal(err)
				}
				<-own
			}
			return regular(path)
		}
		code := f.run("--no-backup")
		probeRegular = regular
		signal.Stop(own)
		want := "[ng-deploy] ERROR interrupted (" + signalNames[sig] + ") before the copy; nothing was copied to the station\n"
		if code != signalBase+int(sig) || !strings.HasSuffix(f.err, want) || strings.Contains(f.calls(), "gradlew") {
			t.Errorf("%v: exit %d, stderr %q, calls %q", sig, code, f.err, f.calls())
		}
	}
}

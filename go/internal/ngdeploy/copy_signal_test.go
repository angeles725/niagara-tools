package ngdeploy

import (
	"os"
	"os/signal"
	"strings"
	"syscall"
	"testing"
)

// signalDuring delivers sig to this process while the rename onto a path ending in suffix is held, and
// lets the rename go on only when the signal has reached every channel that asked for it: the test asks
// for it too, and signal.Stop on its own channel waits for the delivery in progress to end.
func signalDuring(t *testing.T, sig syscall.Signal, suffix string) {
	t.Helper()
	own := make(chan os.Signal, 1)
	signal.Notify(own, sig) // also keeps the signal from ending the test when nothing else catches it
	t.Cleanup(func() { rename, link = os.Rename, os.Link; signal.Stop(own) })
	sent := false
	during := func(really func(string, string) error) func(string, string) error {
		return func(from, to string) error {
			err := really(from, to)
			if !sent && strings.HasSuffix(to, suffix) {
				sent = true
				if e := syscall.Kill(os.Getpid(), sig); e != nil {
					t.Fatal(e)
				}
				<-own
				signal.Stop(own)
				signal.Notify(own, sig)
			}
			return err
		}
	}
	rename, link = during(os.Rename), during(os.Link) // a jar is moved aside with link
}

// A SIGINT or SIGTERM during the copy is honoured at the next step: what the copy did is undone from the
// jars aside, the staged jars are removed, and the run ends with 128 + the signal, wherever it arrived.
func TestASignalDuringTheCopyUndoesItAndExitsWithTheSignalCode(t *testing.T) {
	for _, tc := range []struct {
		sig     syscall.Signal
		name    string
		during  string // the rename that is in progress when the signal arrives
		outcome string
	}{
		{syscall.SIGTERM, "SIGTERM", "/demo-rt.jar" + asideSuffix, copyUnchanged},
		{syscall.SIGINT, "SIGINT", "/demo-ux.jar" + asideSuffix, copyUnchanged},
		{syscall.SIGTERM, "SIGTERM", "/demo-rt.jar", copyUndone},
		{syscall.SIGINT, "SIGINT", "/demo-ux.jar", copyUndone},
	} {
		f := newFixture(t)
		upgrade(t, f)
		signalDuring(t, tc.sig, tc.during)
		code := f.run("--no-backup")
		want := warnNoBackup + "[ng-deploy] ERROR copy interrupted (" + tc.name + "); " + tc.outcome
		if code != 128+int(tc.sig) || f.err != want || strings.Contains(f.out, "copy ok") || f.exists(f.work+"/.last-deploy-sha") {
			t.Errorf("%s during the rename onto %s: exit %d, stdout:\n%sstderr %q", tc.name, tc.during, code, f.out, f.err)
		}
		f.station("old-rt", "old-ux", 0o640)
	}
}

package buildmodule

import (
	"os"
	"os/signal"
	"syscall"
)

// exitProcess ends the run from the signal handler; a test replaces it.
var exitProcess = os.Exit

// signalNames are the signals the handler catches, as its messages name them.
var signalNames = map[syscall.Signal]string{syscall.SIGINT: "SIGINT", syscall.SIGTERM: "SIGTERM", syscall.SIGHUP: "SIGHUP"}

// cleanupOnSignal watches for SIGINT, SIGTERM and SIGHUP. When one arrives while modules/ may hold a jar that
// has not passed the gate (from the baseline snapshot until the chain passed), the pre-build jars are put back
// first, as a failing gate does (build-mod Δ6); then the temporary directory, which holds the snapshot, is
// removed and the run ends as bash does for that signal. The returned function stops watching and waits for
// a handler in progress. A gradle that ignores the signal (it reaches only this process, as a kill of its
// pid does) can still install its jar after the restore; a terminal's Ctrl-C reaches both.
func (b *build) cleanupOnSignal() (stop func()) {
	ch, done := make(chan os.Signal, 1), make(chan struct{})
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	go func() {
		defer close(done)
		if sig, ok := <-ch; ok {
			b.undoUnverified(sig.(syscall.Signal))
			b.cleanup()
			exitProcess(signalBase + int(sig.(syscall.Signal)))
		}
	}()
	return func() { signal.Stop(ch); close(ch); <-done }
}

// undoUnverified puts modules/ back as it was before the build when the run is interrupted between the
// baseline snapshot and a passing chain; outside that window it does nothing.
func (b *build) undoUnverified(sig syscall.Signal) {
	if !b.unverified.Load() {
		return
	}
	if b.opt.skipDrift {
		b.warn(msgInterruptedNoSnapshot, signalNames[sig], b.home)
		return
	}
	b.warn(msgInterrupted, signalNames[sig])
	for _, p := range b.profiles {
		b.restoreBaseline(p, restoredPlain)
		b.removeFirstDeploy(p)
	}
}

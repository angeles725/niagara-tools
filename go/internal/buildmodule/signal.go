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
// first, as a failing gate does (retro go-build-module-port Δ6); then the temporary directory, which holds
// the snapshot, is removed and the run ends as bash does for that signal. modulesMu keeps a restore or a
// cleanup of the chain from running at the same time (command.go). The returned function stops watching and waits for
// a handler in progress. Before the restore the signal is passed on to a running gradle, which the handler
// waits for (and kills after gradleStopWait), and no gradle starts after it, so a kill of build.sh's pid alone
// does not let the gradle the run started install its jar after the restore (retro p1-build-deploy-safety Δ4).
func (b *build) cleanupOnSignal() (stop func()) {
	ch, done := make(chan os.Signal, 1), make(chan struct{})
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	go func() {
		defer close(done)
		if sig, ok := <-ch; ok {
			b.modulesMu.Lock() // held to the end: the chain's own restore or cleanup waits, then the run exits
			// gradle ends without modulesMu, so it is stopped with the lock held: the chain, once gradle ended,
			// cannot restore or clean up before the handler does.
			b.stopGradle(sig.(syscall.Signal))
			b.undoUnverified(sig.(syscall.Signal))
			b.cleanupHeld()
			exitProcess(signalBase + int(sig.(syscall.Signal)))
			b.modulesMu.Unlock() // reached only when a test replaced exitProcess
		}
	}()
	return func() { signal.Stop(ch); close(ch); <-done }
}

// undoUnverified puts modules/ back as it was before the build when the run is interrupted between the
// baseline snapshot and a passing chain; outside that window it does nothing. The caller holds modulesMu.
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
		b.restoreBaselineHeld(p, restoredPlain)
		b.removeFirstDeployHeld(p)
	}
}

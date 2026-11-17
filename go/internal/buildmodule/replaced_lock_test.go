package buildmodule

import (
	"os"
	"strings"
	"testing"
	"time"
)

// Retro p1-build-deploy-safety Δ11: keepReplaced reads modules/ under modulesMu, so it waits while the signal
// handler (or the chain's own restore) holds it, and never keeps a copy of a jar the restore is about to
// replace (verifier probe P11b of #795).
func TestKeepReplacedWaitsForTheRestoreLock(t *testing.T) {
	b := &build{}
	b.modulesMu.Lock()
	done := make(chan struct{})
	go func() { b.keepReplaced(); close(done) }()
	select {
	case <-done:
		t.Fatal("keepReplaced ran while modulesMu was held")
	case <-time.After(200 * time.Millisecond):
	}
	b.modulesMu.Unlock()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("keepReplaced did not run after modulesMu was released")
	}
}

// On a mirror home, an interrupt whose gradle ends while the handler restores keeps no .mirror-replaced copy
// and prints no "replaced" row: the restored modules/ jar is the deployed one (verifier probe P11b of #795).
func TestAnInterruptedMirrorBuildKeepsNoReplacedCopy(t *testing.T) {
	f, oldSum := mirrorBuild(t)
	code := -1
	exitProcess = func(c int) { code = c }
	t.Cleanup(func() { exitProcess = os.Exit })
	f.does("gradlew", `cp "`+f.built("rt")+`" "`+f.deployed("rt")+`"`+"\nkill -TERM $PPID\n")
	f.run(f.root, "Foo", f.home)
	if code != 143 {
		t.Errorf("exit %d, want 143\nstderr: %s", code, f.err)
	}
	if kept := f.home + "/.mirror-replaced/Foo-rt.jar." + oldSum; exists(kept) {
		t.Errorf("a replaced copy was kept for a jar the restore put back: %s", kept)
	}
	if strings.Contains(f.out, "replaced") {
		t.Errorf("stdout carries a replaced row:\n%s", f.out)
	}
}

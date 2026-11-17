package buildmodule

import (
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

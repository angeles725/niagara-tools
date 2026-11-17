package buildmodule

import (
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"
)

// gradleStopWait is how long the signal handler waits for gradle to end after passing the signal on, before
// it kills it; a test may shorten it.
var gradleStopWait = 10 * time.Second

// gradleRun is the gradle the chain is running, so the signal handler can stop it before it puts modules/
// back (retro p1-build-deploy-safety Δ4).
type gradleRun struct {
	mu   sync.Mutex
	proc *os.Process
	done chan struct{} // closed when gradle has ended
}

// runTracked starts cmd, records it for the signal handler while it runs, and waits for it.
func (b *build) runTracked(cmd *exec.Cmd) error {
	if err := cmd.Start(); err != nil {
		return err
	}
	done := make(chan struct{})
	b.gradle.mu.Lock()
	b.gradle.proc, b.gradle.done = cmd.Process, done
	b.gradle.mu.Unlock()
	err := cmd.Wait()
	b.gradle.mu.Lock()
	b.gradle.proc, b.gradle.done = nil, nil
	b.gradle.mu.Unlock()
	close(done)
	return err
}

// stopGradle passes sig on to a running gradle and waits for it to end; one that is still running after
// gradleStopWait is killed. A signal sent to build.sh's pid alone does not reach gradle, which could
// otherwise install its jar after the restore. With no gradle running it does nothing.
func (b *build) stopGradle(sig syscall.Signal) {
	b.gradle.mu.Lock()
	proc, done := b.gradle.proc, b.gradle.done
	b.gradle.mu.Unlock()
	if proc == nil {
		return
	}
	_ = proc.Signal(sig) // a gradle that already ended is not an error here
	select {
	case <-done:
	case <-time.After(gradleStopWait):
		_ = proc.Kill()
		<-done
	}
}

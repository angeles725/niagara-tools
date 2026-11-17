package buildmodule

import (
	"errors"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"
)

// gradleStopWait is how long the signal handler waits for gradle to end after passing the signal on, before
// it kills it, and again after the kill; a test may shorten it.
var gradleStopWait = 10 * time.Second

// errInterrupted is runTracked's answer when a signal arrived before gradle could start: it is not started.
var errInterrupted = errors.New("interrupted before gradle started")

// gradleRun is the gradle the chain is running, so the signal handler can stop it before it puts modules/
// back (retro p1-build-deploy-safety Δ4). mu makes starting gradle and recording it one step, so a signal
// either finds gradle recorded or stops it from starting.
type gradleRun struct {
	mu       sync.Mutex
	proc     *os.Process
	done     chan struct{} // closed when gradle has ended
	stopping bool          // a signal arrived: no gradle starts any more
}

// runTracked starts cmd, records it for the signal handler while it runs, and waits for it. The wait has no
// WaitDelay: a child gradle leaves holding the output open is waited for, as before; only the signal handler's
// wait is bounded (stopGradle).
func (b *build) runTracked(cmd *exec.Cmd) error {
	b.gradle.mu.Lock()
	if b.gradle.stopping {
		b.gradle.mu.Unlock()
		return errInterrupted
	}
	if err := cmd.Start(); err != nil {
		b.gradle.mu.Unlock()
		return err
	}
	done := make(chan struct{})
	b.gradle.proc, b.gradle.done = cmd.Process, done
	b.gradle.mu.Unlock()
	err := cmd.Wait()
	b.gradle.mu.Lock()
	b.gradle.proc, b.gradle.done = nil, nil
	b.gradle.mu.Unlock()
	close(done)
	return err
}

// stopGradle keeps any later gradle from starting, passes sig on to a running one and waits for it to end;
// one still running after gradleStopWait is killed, and the wait after the kill is bounded too. The signal
// reaches the gradle the run started (the wrapper's client); a gradle daemon, used outside a mirror home,
// cancels its build on its own. With no gradle running it does nothing.
func (b *build) stopGradle(sig syscall.Signal) {
	b.gradle.mu.Lock()
	b.gradle.stopping = true
	proc, done := b.gradle.proc, b.gradle.done
	b.gradle.mu.Unlock()
	if proc == nil {
		return
	}
	_ = proc.Signal(sig) // a gradle that already ended is not an error here
	select {
	case <-done:
		return
	case <-time.After(gradleStopWait):
	}
	_ = proc.Kill()
	select {
	case <-done:
	case <-time.After(gradleStopWait):
	}
}

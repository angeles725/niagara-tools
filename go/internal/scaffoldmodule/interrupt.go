package scaffoldmodule

import (
	"errors"
	"os"
	"os/signal"
	"syscall"
)

// interrupts holds SIGINT, SIGTERM and SIGHUP from the claim of the destination until its fate is settled,
// so that a run stopped mid-copy discards its partial tree (discard, as a write fault does) instead of
// leaving one the next run refuses with "destination already exists".
type interrupts struct {
	arrive chan os.Signal
	got    syscall.Signal
}

// signalNames are the names the interrupted line gives the held signals.
var signalNames = map[syscall.Signal]string{syscall.SIGINT: "SIGINT", syscall.SIGTERM: "SIGTERM", syscall.SIGHUP: "SIGHUP"}

// emitting is the watch of the emit under way, which writeFile asks before each file; nil outside an emit.
var emitting *interrupts

// errInterrupted ends the copy at the first write after a held signal.
var errInterrupted = errors.New("interrupted")

func holdSignals() *interrupts {
	w := &interrupts{arrive: make(chan os.Signal, 1)}
	signal.Notify(w.arrive, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	return w
}

func (w *interrupts) stop() { signal.Stop(w.arrive) }

// caught reports whether a held signal arrived since the watch started, and keeps which.
func (w *interrupts) caught() bool {
	if w == nil {
		return false
	}
	if w.got == 0 {
		select {
		case sig := <-w.arrive:
			w.got = sig.(syscall.Signal)
		default:
		}
	}
	return w.got != 0
}

// interrupted is the failure of an emit a held signal stopped: 128 + the signal, as a shell reports it.
func (w *interrupts) interrupted() *failure {
	return failf(128+int(w.got), interruptedLine, signalNames[w.got])
}

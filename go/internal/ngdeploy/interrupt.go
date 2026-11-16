package ngdeploy

import (
	"os"
	"os/signal"
	"syscall"
)

// interrupts watches for SIGINT and SIGTERM: the copy has a watch of its own while it runs, and the deploy
// one from the move aside in <NIAGARA_HOME>/modules on (home.go). Before that the deploy keeps the default:
// the signal ends it, and nothing was touched yet.
type interrupts struct {
	arrive chan os.Signal
	got    syscall.Signal // the first one caught; 0 for none
}

// watchInterrupts starts the watch; stop ends it.
func watchInterrupts() *interrupts {
	w := &interrupts{arrive: make(chan os.Signal, 1)}
	signal.Notify(w.arrive, syscall.SIGINT, syscall.SIGTERM)
	return w
}

func (w *interrupts) stop() { signal.Stop(w.arrive) }

// caught reports whether a signal arrived since the watch started. The copy asks between two steps: a
// step is one rename, so the signal waits for no more than that (or for the cp of one jar).
func (w *interrupts) caught() bool {
	if w.got == 0 {
		select {
		case sig := <-w.arrive:
			w.got, _ = sig.(syscall.Signal)
		default:
		}
	}
	return w.got != 0
}

// name is SIGINT or SIGTERM; exitCode is what a shell reports for a command the signal ended.
func (w *interrupts) name() string  { return signalNames[w.got] }
func (w *interrupts) exitCode() int { return signalBase + int(w.got) }

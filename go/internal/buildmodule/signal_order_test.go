package buildmodule

import (
	"os"
	"sync/atomic"
	"syscall"
	"testing"
	"time"
)

// Retro p1-build-deploy-safety Δ7: a signal that arrives while the chain itself holds modulesMu (inside
// restoreBaseline, removeFirstDeploy or cleanup) waits for it: the handler touches neither modules/ nor the
// temporary directory, and the run does not end, until the chain's step has finished.
func TestASignalWaitsForTheChainsOwnRestore(t *testing.T) {
	temp := t.TempDir() + "/snapshot"
	if err := os.Mkdir(temp, 0o700); err != nil {
		t.Fatal(err)
	}
	var exited atomic.Int32
	exitProcess = func(c int) { exited.Store(int32(c)) }
	t.Cleanup(func() { exitProcess = os.Exit })

	b := &build{temps: []string{temp}}
	b.modulesMu.Lock() // the chain is inside its own restore
	stop := b.cleanupOnSignal()
	if err := syscall.Kill(os.Getpid(), syscall.SIGHUP); err != nil {
		t.Fatal(err)
	}
	time.Sleep(200 * time.Millisecond)
	if c := exited.Load(); c != 0 {
		t.Fatalf("the run ended (exit %d) while the chain held modulesMu", c)
	}
	if _, err := os.Stat(temp); err != nil {
		t.Fatalf("the snapshot was removed while the chain held modulesMu: %v", err)
	}
	b.modulesMu.Unlock() // the chain's step ends
	for i := 0; i < 100 && exited.Load() == 0; i++ {
		time.Sleep(20 * time.Millisecond)
	}
	stop()
	if c := int(exited.Load()); c != signalBase+int(syscall.SIGHUP) {
		t.Errorf("exit %d after the chain's step, want %d", c, signalBase+int(syscall.SIGHUP))
	}
	if _, err := os.Stat(temp); !os.IsNotExist(err) {
		t.Errorf("the snapshot is still there after the handler ran: %v", err)
	}
}

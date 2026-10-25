package cli

import (
	"bytes"
	"context"
	"io"
	"os"
	"syscall"
	"testing"
	"time"

	kitcli "github.com/angeles725/niagara-tools/go/internal/cli"
)

// The first signal only cancels (the shutdown runs); a second one forces the exit, 128+signo.
func TestASecondSignalForcesTheExitTheFirstOnlyCancels(t *testing.T) {
	t.Parallel()
	sigs, exits := make(chan os.Signal, 2), make(chan int, 2)
	ctx := watchSignals(sigs, func(code int) { exits <- code })
	sigs <- os.Interrupt
	select {
	case <-ctx.Done():
	case <-time.After(5 * time.Second):
		t.Fatal("the first signal did not cancel")
	}
	select {
	case code := <-exits:
		t.Fatalf("the first signal exited %d: the session close must run first", code)
	case <-time.After(50 * time.Millisecond):
	}
	sigs <- syscall.SIGTERM
	select {
	case code := <-exits:
		if code != 128+int(syscall.SIGTERM) {
			t.Errorf("forced exit %d, want %d", code, 128+int(syscall.SIGTERM))
		}
	case <-time.After(5 * time.Second):
		t.Fatal("a second signal did not force the exit")
	}
}

// An interrupted server exits 0 and prints nothing, with its stdin still open.
func TestAnInterruptedRunExitsZeroWithNoOutput(t *testing.T) {
	t.Parallel()
	interrupt, cancel := context.WithCancel(context.Background())
	cancel()
	in, w := io.Pipe()
	t.Cleanup(func() { w.Close() })
	var out, errOut bytes.Buffer
	if code := RunContext(interrupt, nil, in, kitcli.Stdio{Out: &out, Err: &errOut}); code != 0 ||
		out.Len() != 0 || errOut.Len() != 0 {
		t.Errorf("exit %d, out %q, err %q: want 0 and no output", code, out.String(), errOut.String())
	}
}

package scaffoldmodule

import (
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"testing"
)

// signalAtWrite sends sig to this process at the n-th file write of a run, waits until it is delivered,
// then lets the write go on; the seam is restored when the test ends. A channel of the test's own keeps a
// signal nothing else catches from ending the test binary.
func signalAtWrite(t *testing.T, n int, sig syscall.Signal) {
	t.Helper()
	own := make(chan os.Signal, 1)
	signal.Notify(own, sig)
	real, calls := createFile, 0
	createFile = func(path string, data []byte, mode os.FileMode) error {
		if calls++; calls == n {
			if err := syscall.Kill(os.Getpid(), sig); err != nil {
				t.Fatal(err)
			}
			<-own
		}
		return real(path, data, mode)
	}
	t.Cleanup(func() { createFile = real; signal.Stop(own) })
}

// S1: a SIGINT, SIGTERM or SIGHUP during the copy discards the partial tree as a write fault does, with
// 128 + the signal, so the next run is not refused with "destination already exists".
func TestASignalDuringTheCopyRemovesThePartialTree(t *testing.T) {
	for sig, name := range map[syscall.Signal]string{syscall.SIGINT: "SIGINT", syscall.SIGTERM: "SIGTERM", syscall.SIGHUP: "SIGHUP"} {
		t.Run(name, func(t *testing.T) {
			tb := newKit(t)
			out := filepath.Join(t.TempDir(), "out")
			dest := out + "/Foo"
			signalAtWrite(t, 3, sig)
			r := run(t, tb, "Foo", out)
			want := "scaffold-module: interrupted (" + name + ")\nscaffold-module: removed the partial " + dest + "\n"
			if r.code != 128+int(sig) || r.out != "" || r.err != want {
				t.Errorf("code=%d out=%q\n err %q\nwant %q", r.code, r.out, r.err, want)
			}
			if _, err := os.Lstat(dest); !os.IsNotExist(err) {
				t.Errorf("the partial tree stays: %v (err %v)", keys(tree(t, dest)), err)
			}
			createFile = os.WriteFile
			if r := run(t, tb, "Foo", out); r.code != 0 {
				t.Errorf("rerun: code=%d err=%q", r.code, r.err)
			}
		})
	}
}

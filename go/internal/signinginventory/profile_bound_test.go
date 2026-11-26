package signinginventory

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

// A FIFO or an oversized file named like a profile is one WARN row: the run neither hangs on it nor reads it
// whole. [ev: issue #1160, verify advisory]
func TestAProfileThatIsNotASmallRegularFileIsAWarnRow(t *testing.T) {
	home := t.TempDir()
	dir := filepath.Join(home, ".tridium/security")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	fifo := filepath.Join(dir, "fifo.signing.xml")
	if err := syscall.Mkfifo(fifo, 0o600); err != nil {
		t.Skipf("mkfifo: %v", err)
	}
	big := write(t, dir, "big.signing.xml", strings.Repeat("x", maxProfileBytes+1))
	done := make(chan struct{})
	var code int
	var out, errs string
	go func() {
		code, out, errs = inv(t, map[string]string{"HOME": home})
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(20 * time.Second):
		t.Fatal("the run hung on a FIFO profile")
	}
	if code != 0 || errs != "" {
		t.Fatalf("exit %d, stderr %q\n%s", code, errs, out)
	}
	has(t, out, "WARN", "profile", fifo, "not a regular file of at most 1 MiB")
	has(t, out, "WARN", "profile", big, "not a regular file of at most 1 MiB")
}

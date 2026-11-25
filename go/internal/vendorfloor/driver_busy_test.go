package vendorfloor

import (
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// An executable another descriptor still holds open for writing fails exec with ETXTBSY until it is
// closed: start retries it instead of reporting "cannot run" (CI flake of
// TestNodeThatEndsWithoutAResultIsAnEnvFault: "text file busy").
func TestStartRetriesATextFileBusyExec(t *testing.T) {
	p := filepath.Join(t.TempDir(), "node")
	if err := os.WriteFile(p, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	w, err := os.OpenFile(p, os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	go func() { time.Sleep(40 * time.Millisecond); w.Close() }()
	cmd, stdout, err := start(p, nil, io.Discard)
	if err != nil {
		t.Fatalf("a busy executable must be retried until it can run: %v", err)
	}
	_, _ = io.Copy(io.Discard, stdout)
	if err := cmd.Wait(); err != nil {
		t.Fatal(err)
	}
}

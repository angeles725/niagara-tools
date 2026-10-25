//go:build unix

package niagaratest

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

// TestExecRunnerKillsTheGroup: a timeout kills the started process and the child it forked, not only the
// direct child (a gradle JVM, a WSL interop process).
func TestExecRunnerKillsTheGroup(t *testing.T) {
	t.Parallel()
	pidFile := filepath.Join(t.TempDir(), "pid")
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	start := time.Now()
	_, err := execRunner{}.Run(ctx, Cmd{Name: "sh", Args: []string{"-c", `sleep 30 & echo $! > "$0"; wait`, pidFile}})
	if !errors.Is(err, context.DeadlineExceeded) || time.Since(start) > 5*time.Second {
		t.Fatalf("err %v after %s", err, time.Since(start))
	}
	b, err := os.ReadFile(pidFile)
	if err != nil {
		t.Fatal(err)
	}
	pid, _ := strconv.Atoi(strings.TrimSpace(string(b)))
	for i := 0; syscall.Kill(pid, 0) == nil; i++ {
		if i == 50 {
			t.Fatalf("the forked child %d outlived the timeout", pid)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func TestExecRunnerExitAndEnv(t *testing.T) {
	t.Parallel()
	var out bytes.Buffer
	code, err := execRunner{}.Run(context.Background(), Cmd{Name: "sh", Args: []string{"-c", `printf %s "$RNT_X"; exit 7`},
		Env: []string{"RNT_X=seen"}, Stdout: &out})
	if code != 7 || err != nil || out.String() != "seen" {
		t.Errorf("exit %d, err %v, stdout %q", code, err, out.String())
	}
}

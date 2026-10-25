//go:build unix

package niagaratest

import (
	"os/exec"
	"syscall"
)

// ownGroup starts cmd in a process group of its own and makes cancellation kill that whole group, so a
// gradle JVM or a WSL interop child cannot outlive a timeout.
func ownGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error { return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) }
}

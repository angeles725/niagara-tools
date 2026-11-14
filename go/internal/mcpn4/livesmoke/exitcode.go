package livesmoke

import (
	"os"
	"strconv"
	"syscall"
)

// exitCode is the exit status as Python's Popen.returncode prints it: the code, or minus the
// signal that killed the process.
func exitCode(ps *os.ProcessState) string {
	if ps == nil {
		return "None"
	}
	if ws, ok := ps.Sys().(syscall.WaitStatus); ok && ws.Signaled() {
		return strconv.Itoa(-int(ws.Signal()))
	}
	return strconv.Itoa(ps.ExitCode())
}

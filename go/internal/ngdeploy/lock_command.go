package ngdeploy

import (
	"os"
	"path/filepath"
	"strings"
)

// ownCommand is the program name of this run as ps -p <pid> -o comm= prints it: the comm of the process,
// or, where /proc has none, the name the run was started by.
func ownCommand() string {
	if comm, ok := commandOf("self"); ok {
		return comm
	}
	return filepath.Base(os.Args[0])
}

// commandOf reads the program name of a process of this host (pid, or "self"); ok is false when no such
// process runs or /proc cannot say.
func commandOf(pid string) (string, bool) {
	comm, err := os.ReadFile("/proc/" + pid + "/comm")
	if err != nil {
		return "", false
	}
	return strings.TrimSuffix(string(comm), "\n"), true
}

// recycled says, for a lock of this host whose pid runs another program than the one the lock records,
// which program that is: the run that made the lock is gone and its pid went to that program.
func recycled(pid, host, command string) (string, bool) {
	if here, _ := os.Hostname(); host != here {
		return "", false
	}
	running, ok := commandOf(pid)
	return running, ok && running != command
}

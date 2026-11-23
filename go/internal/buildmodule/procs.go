package buildmodule

import (
	"os"
	"strconv"
	"strings"
	"syscall"
)

// What gradle started. The signal the handler passes on reaches gradle's own pid only; a single-use daemon
// it forked is its child, so it is found from /proc, ended with gradle, and cannot install a jar after the
// restore (retro p1-build-deploy-safety Δ10). The other way, starting gradle in its own process group and
// signalling the group, takes it out of the terminal's foreground group (a rich console writing terminal
// modes from a background group is stopped by SIGTTOU), so a Ctrl-C would no longer reach it the way it does
// today. Without /proc (not Linux) nothing is found and nothing more is ended.

// procRef is one process: its pid and its start time (field 22 of /proc/<pid>/stat), so a pid that has been
// handed to another process since is never mistaken for it.
type procRef struct {
	pid   int
	start string
}

// procStat is the parent pid, start time and state of a process, ok false when it is gone or unreadable.
func procStat(pid int) (ppid int, start string, state byte, ok bool) {
	b, err := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/stat")
	if err != nil {
		return 0, "", 0, false
	}
	// "pid (comm) state ppid …": comm may hold spaces and parentheses, so the fields start after the last ')'
	text := string(b)
	i := strings.LastIndexByte(text, ')')
	if i < 0 {
		return 0, "", 0, false
	}
	f := strings.Fields(text[i+1:]) // f[0] is field 3 (state), f[1] the parent pid, f[19] field 22 (start time)
	if len(f) < 20 {
		return 0, "", 0, false
	}
	ppid, err = strconv.Atoi(f[1])
	if err != nil {
		return 0, "", 0, false
	}
	return ppid, f[19], f[0][0], true
}

// descendants are the live processes below pid: its children, their children and so on.
func descendants(root int) []procRef {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil
	}
	children := map[int][]procRef{}
	for _, e := range entries {
		pid, err := strconv.Atoi(e.Name())
		if err != nil {
			continue
		}
		if ppid, start, state, ok := procStat(pid); ok && state != 'Z' {
			children[ppid] = append(children[ppid], procRef{pid, start})
		}
	}
	var out []procRef
	for queue := []int{root}; len(queue) > 0; queue = queue[1:] {
		for _, c := range children[queue[0]] {
			out = append(out, c)
			queue = append(queue, c.pid)
		}
	}
	return out
}

// alive reports whether the process p still runs under its pid.
func alive(p procRef) bool {
	_, start, state, ok := procStat(p.pid)
	return ok && start == p.start && state != 'Z'
}

// killAll ends the processes of kin that still run, and what each of them has started since: the ones whose
// parent ended are now children of init, so the tree is read again from each survivor.
func killAll(kin []procRef) {
	var all []procRef
	for _, p := range kin {
		if alive(p) {
			all = append(all, p)
			all = append(all, descendants(p.pid)...)
		}
	}
	for _, p := range all {
		if alive(p) {
			_ = syscall.Kill(p.pid, syscall.SIGKILL)
		}
	}
}

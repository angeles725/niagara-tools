package niagaratest

import (
	"fmt"
	"strings"
)

// service is a running niagarad service: its process and the niagara_home its PathName names ("" when it
// names none).
type service struct{ name, home string }

// services reads the `SVC <pid> <name> <PathName>` rows of inUseScript (tab-separated).
func services(rows []string) map[string]service {
	svcs := map[string]service{}
	for _, r := range rows {
		if f := strings.SplitN(r, "\t", 4); len(f) == 4 && f[0] == "SVC" {
			svcs[f[1]] = service{f[2], serviceHome(f[3])}
		}
	}
	return svcs
}

// serviceHome is the niagara_home of a service PathName, `"X:\home\bin\niagarad.exe" args` or unquoted.
func serviceHome(pathName string) string {
	exe := strings.TrimSpace(pathName)
	if rest, ok := strings.CutPrefix(exe, `"`); ok {
		exe, _, _ = strings.Cut(rest, `"`)
	} else if i := strings.Index(strings.ToLower(exe), ".exe"); i >= 0 {
		exe = exe[:i+4]
	}
	const tail = `\bin\niagarad.exe`
	if !strings.HasSuffix(strings.ToLower(exe), tail) || !drivePath.MatchString(exe) {
		return ""
	}
	return exe[:len(exe)-len(tail)]
}

// attribute judges an `UNREAD <pid> <ppid> <image>` row: a process whose path and command line a non-elevated
// probe cannot read (a service's station.exe). It belongs to the install of the niagarad service it is, or
// that started it. refusal is "" when it runs from another install; pass is then the row naming the source.
func attribute(r string, svcs map[string]service, install string) (pass, refusal string) {
	f := strings.Split(r, "\t")
	if len(f) != 4 {
		return "", "cannot verify that no station uses the install: " + r
	}
	pid, ppid, image := f[1], f[2], f[3]
	src, s := pid, svcs[pid]
	if s.home == "" {
		src, s = ppid, svcs[ppid]
	}
	if s.home == "" {
		return "", fmt.Sprintf("cannot attribute running station process %s to an install", pid)
	}
	via := fmt.Sprintf("%s %s (service %s, pid %s)", image, pid, s.name, src)
	if within(install, s.home) || within(s.home, install) {
		return "", "a running process uses the install: " + via
	}
	return fmt.Sprintf("preflight: %s %s belongs to %s (service %s, pid %s)", image, pid, s.home, s.name, src), ""
}

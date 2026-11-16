package ngdeploy

import (
	"fmt"
	"os"
	"regexp"
	"time"
)

// lockHolder reads the content a run writes into its lock (lockContent); a lock an earlier version wrote
// has no command line.
var lockHolder = regexp.MustCompile(`\Apid (\d+)\nhost (\S+)\nstarted (\S+)\n(?:command ([^\n]+)\n)?\z`)

// lockedBy says who holds a lock and how to tell that its run is gone: from its content, or, when that is
// not what a run writes, from the age of the file alone. hint is the line that follows when the pid of the
// lock runs another program on this host (recycled), else "".
func lockedBy(path string) (holder, gone, hint string) {
	content, _ := os.ReadFile(path)
	if m := lockHolder.FindStringSubmatch(string(content)); m != nil {
		if started, err := time.Parse(lockTime, m[3]); err == nil {
			pid, host, command := m[1], m[2], m[4]
			if command == "" {
				command = ownCommand() // an earlier version of this program
			}
			if running, ok := recycled(pid, host, command); ok {
				hint = fmt.Sprintf(hintNotNgDeploy, pid, running)
			}
			return fmt.Sprintf(holderKnown, pid, host, m[3], age(started)), fmt.Sprintf(goneKnown, pid, host, command), hint
		}
	}
	made := time.Now()
	if st, err := os.Lstat(path); err == nil {
		made = st.ModTime()
	}
	return fmt.Sprintf(holderUnknown, age(made)), goneUnknown, ""
}

package ngdeploy

import (
	"fmt"
	"os"
	"regexp"
	"time"
)

// lockHolder reads the content a run writes into its lock (lockContent).
var lockHolder = regexp.MustCompile(`\Apid (\d+)\nhost (\S+)\nstarted (\S+)\n\z`)

// lockedBy says who holds a lock and how to tell that its run is gone: from its content, or, when that is
// not what a run writes, from the age of the file alone.
func lockedBy(path string) (holder, gone string) {
	content, _ := os.ReadFile(path)
	if m := lockHolder.FindStringSubmatch(string(content)); m != nil {
		if started, err := time.Parse(lockTime, m[3]); err == nil {
			return fmt.Sprintf(holderKnown, m[1], m[2], m[3], age(started)), fmt.Sprintf(goneKnown, m[1], m[2], m[1])
		}
	}
	made := time.Now()
	if st, err := os.Lstat(path); err == nil {
		made = st.ModTime()
	}
	return fmt.Sprintf(holderUnknown, age(made)), goneUnknown
}

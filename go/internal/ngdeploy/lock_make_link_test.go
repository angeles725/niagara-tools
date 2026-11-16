package ngdeploy

import (
	"os"
	"testing"
)

// D5, without a race: the lock gets its name by one hard link, made when the lock is not there yet and the
// file it links holds the whole content. A lock made, then written, never calls the link and fails here.
func TestTheLockGetsItsNameByOneLinkOfAWholeFile(t *testing.T) {
	defer func() { hardLink = os.Link }()
	dir, links := t.TempDir(), 0
	hardLink = func(from, to string) error {
		links++
		content, err := os.ReadFile(from)
		if _, there := os.Lstat(to); there == nil || to != lockFile(dir) || err != nil || !ownLock.Match(content) {
			t.Errorf("at the link: lock there %v, to %q, temporary file %q (%v)", there == nil, to, content, err)
		}
		return os.Link(from, to)
	}
	code, errs, d := lockIn(dir)
	defer d.unlock()
	content, _ := os.ReadFile(lockFile(dir))
	if code != proceed || errs != "" || links != 1 || !ownLock.Match(content) {
		t.Errorf("exit %d, stderr %q, %d links, lock %q", code, errs, links, content)
	}
}

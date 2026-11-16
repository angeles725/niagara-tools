package ngdeploy

import (
	"os"
	"os/exec"
	"strings"
	"syscall"
	"testing"
	"time"
)

// The sweep removes old temporary lock files only: a lock itself, however old, is another run's and is
// refused and kept; a directory or symbolic link with a temporary lock file's name was not made by a run
// and is left (verifier A4).
func TestTheSweepLeavesAnOldLockAndWhatIsNotARegularFile(t *testing.T) {
	dir := t.TempDir()
	lock, asDir, asLink := lockFile(dir), lockFile(dir)+".x.tmp", lockFile(dir)+".y.tmp"
	if err := os.WriteFile(lock, []byte("pid 4194000\nhost elsewhere\nstarted 2020-01-02T03:04:05Z\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(asDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(dir+"/elsewhere", asLink); err != nil {
		t.Fatal(err)
	}
	long := time.Now().Add(-2 * time.Hour)
	for _, p := range []string{lock, asDir} {
		if err := os.Chtimes(p, long, long); err != nil {
			t.Fatal(err)
		}
	}
	if out, err := exec.Command("/usr/bin/touch", "-h", "-d", "2 hours ago", asLink).CombinedOutput(); err != nil { // the link's own time
		t.Fatalf("%v: %s", err, out)
	}
	code, errs, _ := lockIn(dir)
	if code != ExitEnv || strings.Contains(errs, "NOTICE") {
		t.Errorf("exit %d, stderr %q", code, errs)
	}
	for _, p := range []string{lock, asDir, asLink} {
		if _, err := os.Lstat(p); err != nil {
			t.Errorf("%s was removed", p)
		}
	}
}

// A link that fails because the run's own temporary file is gone (another run swept it) is not a file
// system without hard links: the lock is taken in two steps with no "takes no hard link" NOTICE (verifier A3).
func TestALinkWhoseTemporaryFileIsGoneSaysNoHardLinkNothing(t *testing.T) {
	failLink(t, syscall.ENOENT)
	dir := t.TempDir()
	code, errs, d := lockIn(dir)
	defer d.unlock()
	content, _ := os.ReadFile(lockFile(dir))
	if code != proceed || errs != "" || !ownLock.Match(content) {
		t.Errorf("exit %d, stderr %q, lock %q", code, errs, content)
	}
}

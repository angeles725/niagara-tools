package ngdeploy

import (
	"os"
	"strings"
	"syscall"
	"testing"
	"time"
)

// failLink makes every link of a lock fail with errno, as a file system without hard links does.
func failLink(t *testing.T, errno syscall.Errno) {
	t.Cleanup(func() { hardLink = os.Link })
	hardLink = func(from, to string) error { return &os.LinkError{Op: "link", Old: from, New: to, Err: errno} }
}

// T18: the "takes no hard link" NOTICE comes once per run, and only when a lock was taken: two locks in
// two steps say it once, a lock refused in two steps says only the refusal.
func TestTheNoHardLinkNoticeComesOncePerRunAndOnlyOnSuccess(t *testing.T) {
	failLink(t, syscall.EPERM)
	f := newFixture(t)
	withHome(t, f, true)
	f.run("--no-backup")
	if n := strings.Count(f.err, "takes no hard link"); n != 1 {
		t.Errorf("two locks in two steps: %d NOTICEs, want 1, stderr %q", n, f.err)
	}
	dir := t.TempDir()
	if err := os.WriteFile(lockFile(dir), []byte("pid 4194000\nhost elsewhere\nstarted 2020-01-02T03:04:05Z\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	code, errs, _ := lockIn(dir)
	if code != ExitEnv || strings.Count(errs, "\n") != 1 || !strings.HasPrefix(errs, "[ng-deploy] ERROR another ng-deploy run holds ") {
		t.Errorf("a refused lock: exit %d, stderr %q", code, errs)
	}
}

// T18: a link that fails for another reason than "the lock is there" (EMLINK and the like) takes the lock in
// two steps too, with the NOTICE naming the reason, instead of failing the run.
func TestALinkErrorOtherThanExistFallsBackToTwoSteps(t *testing.T) {
	failLink(t, syscall.EMLINK)
	dir := t.TempDir()
	code, errs, d := lockIn(dir)
	defer d.unlock()
	content, _ := os.ReadFile(lockFile(dir))
	if code != proceed || !strings.Contains(errs, "NOTICE: "+dir+" takes no hard link (Too many links)") || !ownLock.Match(content) {
		t.Errorf("exit %d, stderr %q, lock %q", code, errs, content)
	}
}

// T18: a temporary lock file a killed run left (older than a minute) is removed with a NOTICE when the next
// run takes the lock; a younger one, which a run taking its lock now may still use, is left.
func TestAStaleTemporaryLockFileIsSweptAndAYoungOneLeft(t *testing.T) {
	dir := t.TempDir()
	old, young := lockFile(dir)+".111.tmp", lockFile(dir)+".222.tmp"
	for _, p := range []string{old, young} {
		if err := os.WriteFile(p, []byte("pid 1\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	long := time.Now().Add(-2 * time.Minute)
	if err := os.Chtimes(old, long, long); err != nil {
		t.Fatal(err)
	}
	code, errs, d := lockIn(dir)
	defer d.unlock()
	want := "[ng-deploy] NOTICE: " + old + ", the temporary lock file of a run that was killed while it took its lock, was removed\n"
	_, oldErr := os.Lstat(old)
	_, youngErr := os.Lstat(young)
	if code != proceed || errs != want || oldErr == nil || youngErr != nil {
		t.Errorf("exit %d, stderr %q, old left %v, young left %v", code, errs, oldErr == nil, youngErr == nil)
	}
}

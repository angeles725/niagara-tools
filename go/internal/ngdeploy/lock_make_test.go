package ngdeploy

import (
	"bytes"
	"io"
	"os"
	"regexp"
	"strings"
	"syscall"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

// lockIn takes the lock of dir as a run of its own and returns how it ended, its stderr and the run.
func lockIn(dir string) (int, string, *deploy) {
	var errs bytes.Buffer
	d := &deploy{s: cli.Stdio{Out: io.Discard, Err: &errs}}
	code := d.lock(dir)
	if d.signals != nil {
		d.signals.stop()
	}
	return code, errs.String(), d
}

// The content of a lock: what lockContent makes of this run.
var ownLock = regexp.MustCompile(`\Apid \d+\nhost \S+\nstarted \d{4}-\d\d-\d\dT\d\d:\d\d:\d\dZ\n\z`)

// D5: on a file system that takes no hard link the lock is made in two steps, and the run says so: one
// NOTICE, a whole lock, no temporary file left.
func TestALockWhereHardLinksAreUnavailableIsMadeInTwoStepsAndSaysSo(t *testing.T) {
	defer func() { hardLink = os.Link }()
	hardLink = func(from, to string) error { return &os.LinkError{Op: "link", Old: from, New: to, Err: syscall.EPERM} }
	dir := t.TempDir()
	code, errs, d := lockIn(dir)
	defer d.unlock()
	want := "[ng-deploy] NOTICE: " + dir + " takes no hard link (Operation not permitted): the run lock is made, then written; " +
		"a run that starts in the same instant can find it empty and say that it does not say which run made it\n"
	content, _ := os.ReadFile(lockFile(dir))
	entries, _ := os.ReadDir(dir)
	if code != proceed || errs != want || !ownLock.Match(content) || len(entries) != 1 {
		t.Errorf("exit %d, stderr %q, lock %q, directory %v", code, errs, content, entries)
	}
}

// A lock in a directory this run may not write is still refused, and such a directory with no lock gets
// none (the run cannot change it), as before the temporary file.
func TestALockInADirectoryThisRunMayNotWriteIsStillRefused(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root writes every directory")
	}
	held, free := t.TempDir(), t.TempDir()
	if err := os.WriteFile(lockFile(held), []byte("pid 4194000\nhost elsewhere\nstarted 2020-01-02T03:04:05Z\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, dir := range []string{held, free} {
		if err := os.Chmod(dir, 0o555); err != nil {
			t.Fatal(err)
		}
		defer os.Chmod(dir, 0o755)
	}
	code, errs, _ := lockIn(held)
	if code != ExitEnv || !strings.HasPrefix(errs, "[ng-deploy] ERROR another ng-deploy run holds "+lockFile(held)+" (pid 4194000 ") {
		t.Errorf("a held lock: exit %d, stderr %q", code, errs)
	}
	code, errs, d := lockIn(free)
	if entries, _ := os.ReadDir(free); code != proceed || errs != "" || len(d.locks) != 0 || len(entries) != 0 {
		t.Errorf("no lock: exit %d, stderr %q, locks %v, directory %v", code, errs, d.locks, entries)
	}
}

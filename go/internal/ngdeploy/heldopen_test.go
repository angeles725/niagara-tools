package ngdeploy

import (
	"os"
	"strings"
	"syscall"
	"testing"
)

// heldOpenLine is the line after the ERROR of a jar that could not be replaced or moved because a program
// holds it open (retro 2026-11-15-coordination-stretch-deltas D1).
const heldOpenLine = "[ng-deploy]   a running station or Workbench holds its module jars open; stop it and run again\n"

// A jar of the station that cannot be renamed in (Permission denied, as Windows answers for a jar a running
// station holds) is the exit 40 line, then the line that names the likely cause.
func TestACopyRefusedByTheFileSystemNamesARunningStation(t *testing.T) {
	f := newFixture(t)
	upgrade(t, f)
	failRename(t, renameInUX)
	code := f.run("--no-backup")
	if code != ExitCopy || !strings.HasSuffix(f.err, "[ng-deploy] ERROR copy failed for demo-ux.jar: Permission denied; "+copyUndone+heldOpenLine) {
		t.Errorf("exit %d, stderr %q", code, f.err)
	}
	f.station("old-rt", "old-ux", 0o640)
}

// A home jar that cannot be moved aside for the same reason is the exit 10 line, then the same line.
func TestAHomeJarRefusedByTheFileSystemNamesARunningStation(t *testing.T) {
	f := newFixture(t)
	modules := withHome(t, f, true)
	failRename(t, "/home/modules/demo-wb.jar"+asideSuffix)
	code := f.run("--no-backup")
	want := "[ng-deploy] ERROR build not started: demo-wb.jar could not be moved aside in " + modules + " (Permission denied); nothing was replaced: " + modules + " holds the jars it held before the build\n" + heldOpenLine
	if code != ExitEnv || !strings.HasSuffix(f.err, want) {
		t.Errorf("exit %d, stderr %q", code, f.err)
	}
}

// Only EACCES, EPERM and EBUSY name a running station: a failure for another reason keeps the one line.
func TestAFailureForAnotherReasonDoesNotNameARunningStation(t *testing.T) {
	for errno, held := range map[syscall.Errno]bool{syscall.EACCES: true, syscall.EPERM: true, syscall.EBUSY: true, syscall.ENOSPC: false, syscall.EXDEV: false} {
		f := newFixture(t)
		upgrade(t, f)
		t.Cleanup(func() { rename = os.Rename })
		rename = func(from, to string) error {
			if strings.HasSuffix(to, "/demo-ux.jar") && strings.HasSuffix(from, stagedSuffix) {
				return &os.LinkError{Op: "rename", Old: from, New: to, Err: errno}
			}
			return os.Rename(from, to)
		}
		code := f.run("--no-backup")
		if code != ExitCopy || strings.HasSuffix(f.err, heldOpenLine) != held {
			t.Errorf("%v: exit %d, stderr %q", errno, code, f.err)
		}
		rename = os.Rename
	}
}

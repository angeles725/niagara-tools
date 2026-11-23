package ngdeploy

import (
	"os"
	"strings"
	"syscall"
	"testing"
)

// heldOpenLine is the line after the ERROR of a jar that could not be replaced or moved because a program
// holds it open (retro 2026-11-15-coordination-stretch-deltas D1).
const heldOpenLine = "[ng-deploy]   a running station or Workbench of that installation holds its module jars open: stop it first (on Windows the station is a service or station.exe: stop the service or end the process, and close Workbench), then run again\n"

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

// EBUSY and ETXTBSY, and EACCES or EPERM in a writable directory, name a running station; a failure for
// another reason keeps the one line.
func TestAFailureForAnotherReasonDoesNotNameARunningStation(t *testing.T) {
	for errno, held := range map[syscall.Errno]bool{syscall.EACCES: true, syscall.EPERM: true, syscall.EBUSY: true, syscall.ETXTBSY: true, syscall.ENOSPC: false, syscall.EXDEV: false} {
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

// EACCES or EPERM in a directory whose mode lets this user write is the Windows sharing violation (a running
// station); in one that does not, it is a permission problem and the line says so instead. EBUSY and
// ETXTBSY name a running station either way (retro review R3/R4 held-open hint).
func TestTheHintTellsAHeldJarFromAPermissionProblem(t *testing.T) {
	t.Cleanup(func() { writable = func(dir string) bool { return syscall.Access(dir, 2|1) == nil } })
	for _, tc := range []struct {
		errno    syscall.Errno
		writable bool
		held     bool
		denied   bool
	}{
		{syscall.EACCES, true, true, false}, {syscall.EACCES, false, false, true},
		{syscall.EPERM, true, true, false}, {syscall.EPERM, false, false, true},
		{syscall.EBUSY, false, true, false}, {syscall.ETXTBSY, false, true, false},
		{syscall.ENOSPC, true, false, false}, {syscall.ENOSPC, false, false, false},
	} {
		writable = func(string) bool { return tc.writable }
		err := &os.LinkError{Op: "rename", Old: "a", New: "b", Err: tc.errno}
		if heldOpen(err, "/d") != tc.held || permissionDenied(err, "/d") != tc.denied {
			t.Errorf("%v, writable %v: held %v denied %v", tc.errno, tc.writable, heldOpen(err, "/d"), permissionDenied(err, "/d"))
		}
	}
}

// A station directory this user may not write is refused at the probe with the line about its permissions,
// never with the one that tells the operator to stop a running station.
func TestAReadOnlyStationNamesItsPermissionsNotARunningStation(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("root writes everywhere")
	}
	f := newFixture(t)
	upgrade(t, f)
	f.chmod(f.modules, 0o555)
	code := f.run("--no-backup")
	want := "[ng-deploy] ERROR copy refused: demo-rt.jar in " + f.modules + " cannot be moved (Permission denied); nothing was built or copied\n" +
		"[ng-deploy]   " + f.modules + " does not let this user create, rename or remove files (no write permission): fix its permissions or ownership, then run again; a running station is not the cause\n"
	if code != ExitCopy || !strings.HasSuffix(f.err, want) || strings.Contains(f.err, "Workbench") {
		t.Errorf("exit %d, stderr %q", code, f.err)
	}
}

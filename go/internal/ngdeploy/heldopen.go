package ngdeploy

import (
	"errors"
	"syscall"
)

// writable reports whether this user may create, rename and remove files in dir as its mode says
// (access(2) with W_OK|X_OK); a test replaces it.
var writable = func(dir string) bool { return syscall.Access(dir, 2|1) == nil }

// heldOpen reports whether a jar in dir could not be replaced or moved because another program holds it
// open. EBUSY and ETXTBSY always mean that. A WSL mount of a Windows drive answers a rename of a jar a running
// station or Workbench holds with EACCES (the Windows sharing violation, seen on a live station): that is
// read as held open only when the directory's mode lets this user write there, so a plain permission
// problem is not (permissionDenied).
func heldOpen(err error, dir string) bool {
	switch {
	case errors.Is(err, syscall.EBUSY) || errors.Is(err, syscall.ETXTBSY):
		return true
	case errors.Is(err, syscall.EACCES) || errors.Is(err, syscall.EPERM):
		return writable(dir)
	}
	return false
}

// permissionDenied reports a refusal that the directory's own mode explains.
func permissionDenied(err error, dir string) bool {
	return (errors.Is(err, syscall.EACCES) || errors.Is(err, syscall.EPERM)) && !writable(dir)
}

// failHeld is fail, followed by the line that says why a jar in dir could not be moved when the error tells:
// a running station holds it open (heldOpen), or the directory does not let this user write (permissionDenied).
func (d *deploy) failHeld(err error, dir string, code int, format string, a ...any) int {
	d.fail(code, format, a...)
	switch {
	case heldOpen(err, dir):
		d.warn(indent + hintHeldOpen)
	case permissionDenied(err, dir):
		d.warn(indent+hintNoPermission, dir)
	}
	return code
}

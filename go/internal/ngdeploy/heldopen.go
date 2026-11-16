package ngdeploy

import (
	"errors"
	"syscall"
)

// heldOpen reports whether a jar could not be replaced or moved for the reason a file system gives when
// another program holds the file open: Windows keeps a running station's (or Workbench's) module jars open,
// and a WSL mount answers a rename or a removal of one with EACCES, EPERM or EBUSY.
func heldOpen(err error) bool {
	return errors.Is(err, syscall.EACCES) || errors.Is(err, syscall.EPERM) || errors.Is(err, syscall.EBUSY)
}

// failHeld is fail, followed by the line that names a running station when err is heldOpen.
func (d *deploy) failHeld(err error, code int, format string, a ...any) int {
	d.fail(code, format, a...)
	if heldOpen(err) {
		d.warn(indent + hintHeldOpen)
	}
	return code
}

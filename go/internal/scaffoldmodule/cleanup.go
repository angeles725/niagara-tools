package scaffoldmodule

import (
	"fmt"
	"os"
)

// claim creates dest for this run. scaffold has already refused an existing destination, so a directory
// claim made is the run's own: os.Mkdir never adopts a path that appeared in between.
func claim(dest string) *failure {
	if err := os.Mkdir(dest, 0o777); err != nil {
		return failf(ExitEnv, writeFault, err)
	}
	return nil
}

// discard removes the partial tree of a failed emit below a claimed dest and adds one line to the failure
// saying so (or that its files stay, when the removal fails). The failure keeps its exit code.
func discard(dest string, f *failure) *failure {
	if err := os.RemoveAll(dest); err != nil {
		return &failure{f.text + fmt.Sprintf(partialStays, dest, err), f.code}
	}
	return &failure{f.text + fmt.Sprintf(partialRemoved, dest), f.code}
}

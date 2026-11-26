package buildmodule

import (
	"os"
	"path/filepath"

	"github.com/angeles725/niagara-tools/go/internal/strerror"
)

// moduleStateFile is the module's own record, in its module root: the build-state.v1 envelope of <MOD> lives with
// the module, so nobody has to remember which Niagara version it is for (#1158).
const moduleStateFile = "BUILD-STATE.md"

// stateRecord reads the file the module's build-state.v1 envelope is looked up in: --build-state when given, else
// <module-root>/BUILD-STATE.md when that file exists, else the kit's BUILD-STATE.src.md. file is "" when there is
// no file to read (the default one is missing); a file that exists but holds no envelope of the module records
// nothing, and the kit's is not consulted then. A file that was named, or is the module's own, and cannot be read
// is code ExitEnv after its message.
func (b *build) stateRecord() (file, text string, code int) {
	file, mustRead := b.opt.buildState, b.opt.buildState != ""
	if !mustRead {
		if own := filepath.Join(b.root, moduleStateFile); isRegular(own) {
			file, mustRead = own, true
		} else {
			file = filepath.Join(b.here, kitStateFile)
		}
	}
	data, err := os.ReadFile(file)
	if err != nil {
		if mustRead {
			b.warn(msgStateUnreadable, file, strerror.Text(err))
			return file, "", ExitEnv
		}
		return "", "", proceed
	}
	return file, string(data), proceed
}

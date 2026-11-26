package ngdeploy

import (
	"os"
	"path/filepath"

	"github.com/angeles725/niagara-tools/go/internal/strerror"
)

// moduleStateFile is the module's own record, in its module root (the directory of its gradlew): the
// build-state.v1 envelope of the module lives with the module (#1158).
const moduleStateFile = "BUILD-STATE.md"

// stateRecord reads the file the module's envelope is looked up in: --build-state when given, else
// <module-root>/BUILD-STATE.md when that file exists, else the kit's BUILD-STATE.src.md. file is "" when there
// is no file to read (the default one is missing); a file that exists but holds no envelope of the module records
// nothing, and the kit's is not consulted then. A file that was named, or is the module's own, and cannot be
// read is exit 10 after its message.
func (d *deploy) stateRecord() (file, text string, code int) {
	file, mustRead := d.buildState, d.buildState != ""
	if !mustRead {
		own := filepath.Join(dirname(d.env("GRADLEW_PATH")), moduleStateFile)
		if is, err := regular(own); err == nil && is {
			file, mustRead = own, true
		} else {
			file = filepath.Join(d.scriptDir, kitStateFile)
		}
	}
	data, err := os.ReadFile(file)
	if err != nil {
		if mustRead {
			return file, "", d.fail(ExitEnv, msgStateUnreadable, file, strerror.Text(err))
		}
		return "", "", proceed
	}
	return file, string(data), proceed
}

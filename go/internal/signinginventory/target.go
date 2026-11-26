package signinginventory

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/angeles725/niagara-tools/go/internal/buildstatefields"
	"github.com/angeles725/niagara-tools/go/internal/niagaraversion"
)

// maxProfileBytes bounds a signing profile read: a real profile is a few hundred bytes. [ev: issue #1160]
const maxProfileBytes = 1 << 20

var errNotProfileFile = errors.New("not a regular file of at most 1 MiB")

// readProfile reads a profile only when it is a regular file (after following a symlink) of at most
// maxProfileBytes, so a FIFO or a device in a profile directory cannot hang the run or exhaust memory.
func readProfile(path string) ([]byte, error) {
	st, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if !st.Mode().IsRegular() || st.Size() > maxProfileBytes {
		return nil, errNotProfileFile
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, maxProfileBytes+1))
	if err == nil && len(data) > maxProfileBytes {
		err = errNotProfileFile
	}
	return data, err
}

// recordTarget adds the row of the module's declared niagara_target (#1158) and keeps it for the jar rows. A
// malformed record is a FAIL row.
func (r *run) recordTarget(text, file, module string) {
	t, err := buildstatefields.ModuleTarget(text, module)
	switch {
	case err != nil:
		r.add("FAIL", "niagara-target", "%s: %v (%s)", module, err, file)
	case t.Version == "":
		r.add("INFO", "niagara-target", "%s: none recorded in %s", module, file)
	default:
		r.target = t.Version
		r.add("INFO", "niagara-target", "%s: Niagara %s (recorded in %s)", module, t.Version, file)
	}
}

// targetRow is the FAIL row of a jar compiled for another major.minor than the recorded niagara_target.
func (r *run) targetRow(rel, compiled string) {
	want, got := niagaraversion.MajorMinor(r.target), niagaraversion.MajorMinor(compiled)
	if want == "" || got == "" || want == got {
		return
	}
	r.add("FAIL", "jar-target", "%s", fmt.Sprintf("%s: compiled for Niagara %s, but the module records niagara_target %s",
		rel, compiled, r.target))
}

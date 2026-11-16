package ngdeploy

import (
	"errors"
	"fmt"
	"io/fs"
	"os"

	"github.com/angeles725/niagara-tools/go/internal/strerror"
)

// savedBackup is the backup this run wrote, as the copy step needs it to put a jar back.
type savedBackup struct {
	file   string // _backups/<MODULE>-pre-<time>.tar.gz; "" when the run wrote none
	prefix string // what precedes a jar's name in the archive: "" or "<modules directory>/"
}

// putBack undoes the one rename that went through before a later one failed, so the station holds the
// pair it held: a jar the station did not have is removed, any other is put back from the backup of this
// run. It returns what the exit 40 line says after the jar that failed: the previous pair is back, or which
// jar is the new one, which was not replaced, and what the operator must do before the station restarts.
func (d *deploy) putBack(first, failed *stagedJar) string {
	half := func(why string, a ...any) string {
		return fmt.Sprintf(outcomeHalf, first.name, failed.name, fmt.Sprintf(why, a...))
	}
	switch {
	case !first.had:
		if err := os.Remove(first.target); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return half(whyNotRemoved, first.name, strerror.Text(err))
		}
		return fmt.Sprintf(outcomeRemoved, first.name)
	case d.saved.file == "":
		return half(whyNoBackup, first.name)
	}
	if err := d.restore(first); err != nil {
		return half(whyNotRestored, first.name, d.saved.file, strerror.Text(err))
	}
	return fmt.Sprintf(outcomeRestored, first.name, d.saved.file)
}

// restore writes the jar the backup holds to the staged name, checks it is as long as the jar the copy
// replaced, gives it that jar's mode and renames it into place: the station never sees part of a jar.
func (d *deploy) restore(jar *stagedJar) error {
	file, err := os.OpenFile(jar.staged, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	status := d.run(file, d.s.Err, "tar", "-xzOf", d.saved.file, d.saved.prefix+jar.name)
	if err := file.Close(); err != nil {
		return err
	}
	if status != 0 {
		return fmt.Errorf(whyTarFailed, status)
	}
	if st, err := os.Stat(jar.staged); err != nil || st.Size() != jar.size {
		return errors.New(whyNotTheBytes)
	}
	if err := os.Chmod(jar.staged, jar.mode); err != nil {
		return err
	}
	return rename(jar.staged, jar.target)
}

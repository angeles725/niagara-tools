// Package atomicfile replaces a file through a temporary file in its own directory and a
// rename, so a reader never sees a truncated file and a failure leaves the old one whole.
//
// Mode: an existing target keeps its exact permission bits; a new file is 0644 under the
// process umask. The temporary name is unpredictable and created with O_EXCL, so it never
// overwrites another file; a name that is taken is retried with a fresh one.
package atomicfile

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// maxAttempts bounds the O_EXCL retries on a temporary name that already exists.
const maxAttempts = 16

// Seams the tests replace to force a failure or a name collision.
var (
	tempSuffix = randomSuffix
	writeData  = writeAll
	rename     = os.Rename
)

// Write replaces path with data. On any error the temporary file is removed and path is
// left as it was.
func Write(path string, data []byte) error {
	keep := os.FileMode(0)
	if st, err := os.Stat(path); err == nil {
		keep = st.Mode().Perm()
	}
	f, err := createTemp(path)
	if err != nil {
		return err
	}
	tmp := f.Name()
	err = writeData(f, data)
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err == nil && keep != 0 {
		err = os.Chmod(tmp, keep)
	}
	if err == nil {
		err = rename(tmp, path)
	}
	if err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

// createTemp opens a new ".<base>-<suffix>.tmp" next to path with mode 0644, which the
// kernel masks with the umask (a new target's final mode).
func createTemp(path string) (*os.File, error) {
	dir, base := filepath.Split(path)
	for i := 0; i < maxAttempts; i++ {
		name := filepath.Join(dir, "."+base+"-"+tempSuffix()+".tmp")
		f, err := os.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if errors.Is(err, fs.ErrExist) {
			continue
		}
		return f, err
	}
	return nil, fmt.Errorf("atomicfile: no free temporary file name next to %s after %d attempts", path, maxAttempts)
}

func randomSuffix() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func writeAll(f *os.File, data []byte) error {
	_, err := f.Write(data)
	return err
}

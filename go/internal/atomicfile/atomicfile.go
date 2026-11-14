// Package atomicfile replaces a file through a temporary file in its own directory and a
// rename, so a reader never sees a truncated file and a failure leaves the old one whole.
//
// Mode: an existing target keeps its exact permission bits (000 included), and its temporary
// copy is 0600 until then; a new file is 0644 under the process umask. The temporary name is unpredictable and created with O_EXCL, so it never
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
	keep, existing := os.FileMode(0), false
	if st, err := os.Stat(path); err == nil {
		keep, existing = st.Mode().Perm(), true
	}
	perm := os.FileMode(0o644) // a new target: 0644 under the umask
	if existing {
		perm = 0o600 // never wider than a private target while the data is written
	}
	f, err := createTemp(path, perm)
	if err != nil {
		return err
	}
	tmp := f.Name()
	err = writeData(f, data)
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err == nil && existing {
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

// createTemp opens a new ".<base>-<suffix>.tmp" next to path with mode perm, which the kernel
// masks with the umask.
func createTemp(path string, perm os.FileMode) (*os.File, error) {
	dir, base := filepath.Split(path)
	for i := 0; i < maxAttempts; i++ {
		name := filepath.Join(dir, "."+base+"-"+tempSuffix()+".tmp")
		f, err := os.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, perm)
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

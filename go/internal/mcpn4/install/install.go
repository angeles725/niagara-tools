// Package install copies the running mcp-n4 binary to a stable path (`mcp-n4 install`), so station
// launchers never execute a binary inside a worktree that may be removed or rebuilt.
package install

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
)

// Name is the installed binary's file name under the prefix.
const Name = "mcp-n4"

var (
	rename     = os.Rename
	executable = os.Executable
)

// Install copies src to <prefix>/mcp-n4 atomically: a temporary file in the prefix is written,
// synced, made 0755 and renamed over the target, so a reader sees the old binary or the new one,
// never a partial file, and a failure leaves the old binary and no temporary file. changed is false
// when the target already holds src's bytes (nothing is written).
func Install(src, prefix string) (dst string, changed bool, err error) {
	dst = filepath.Join(prefix, Name)
	data, err := os.ReadFile(src)
	if err != nil {
		return dst, false, err
	}
	if old, rerr := os.ReadFile(dst); rerr == nil && bytes.Equal(old, data) {
		if st, serr := os.Stat(dst); serr == nil && st.Mode().Perm() == 0o755 {
			return dst, false, nil
		}
	}
	if err := os.MkdirAll(prefix, 0o755); err != nil {
		return dst, false, err
	}
	tmp, err := os.CreateTemp(prefix, "."+Name+"-*.tmp")
	if err != nil {
		return dst, false, err
	}
	err = fill(tmp, data)
	if err == nil {
		err = rename(tmp.Name(), dst)
	}
	if err != nil {
		_ = os.Remove(tmp.Name())
		return dst, false, err
	}
	return dst, true, syncDir(prefix)
}

// fill writes data to f, syncs it, makes it 0755 and closes it.
func fill(f *os.File, data []byte) error {
	_, err := io.Copy(f, bytes.NewReader(data))
	if err == nil {
		err = f.Sync()
	}
	if err == nil {
		err = f.Chmod(0o755)
	}
	return errors.Join(err, f.Close())
}

// syncDir makes the rename durable; a directory that cannot be opened or synced is not an error
// of the install (the rename already happened).
func syncDir(dir string) error {
	d, err := os.Open(dir)
	if err != nil {
		return nil
	}
	_ = d.Sync()
	return d.Close()
}

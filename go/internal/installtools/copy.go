package installtools

import (
	"os"
	"syscall"
)

// Copy is `cp src dst` for one regular file: an existing dst is rewritten in place (same inode, same
// mode, written through a symlink), a new one gets src's permission bits under the umask.
func Copy(src, dst string) error {
	st, err := os.Stat(src)
	if err != nil {
		return err
	}
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, st.Mode().Perm())
	if err != nil {
		return err
	}
	_, err = f.Write(data)
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	return err
}

// AddExec is `chmod +x path`: the execute bits the umask allows are added, the others kept.
func AddExec(path string) error {
	st, err := os.Stat(path)
	if err != nil {
		return err
	}
	mask := syscall.Umask(0)
	syscall.Umask(mask)
	return os.Chmod(path, st.Mode().Perm()|(0o111&^os.FileMode(mask)))
}

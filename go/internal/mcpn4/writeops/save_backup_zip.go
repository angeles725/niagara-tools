package writeops

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"
)

// zipWalk is what one walk of the station directory wrote and left out.
type zipWalk struct {
	files                                int
	hasBog, existed                      bool // existed: dest was already there (never removed)
	excludedFiles, excludedDirs, skipped []any
}

// walkZip writes every kept regular file below root into the new file dest (O_EXCL, 0600). Links
// and special files are listed, not followed; skip (the state dir, when it lies inside root) is
// left out as a directory.
func walkZip(root, dest, skip string) (*zipWalk, error) {
	w := &zipWalk{excludedFiles: []any{}, excludedDirs: []any{}, skipped: []any{}}
	f, err := os.OpenFile(dest, os.O_WRONLY|os.O_CREATE|os.O_EXCL|syscall.O_CLOEXEC, 0o600)
	if err != nil {
		w.existed = os.IsExist(err)
		return w, err
	}
	zw := zip.NewWriter(f)
	err = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, p)
		rel = filepath.ToSlash(rel)
		switch {
		case rel == ".":
			return nil
		case d.Type()&fs.ModeSymlink != 0 || !d.IsDir() && !d.Type().IsRegular():
			w.skipped = append(w.skipped, rel)
			return nil
		case d.IsDir() && (p == skip || backupExcluded(rel, true)):
			w.excludedDirs = append(w.excludedDirs, rel)
			return fs.SkipDir
		case d.IsDir():
			return nil
		case backupExcluded(rel, false):
			w.excludedFiles = append(w.excludedFiles, rel)
			return nil
		}
		w.files++
		w.hasBog = w.hasBog || rel == "config.bog"
		return addZipFile(zw, p, rel)
	})
	if cerr := zw.Close(); err == nil {
		err = cerr
	}
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	return w, err
}

func addZipFile(zw *zip.Writer, p, rel string) error {
	src, err := os.Open(p)
	if err != nil {
		return err
	}
	defer src.Close()
	info, err := src.Stat()
	if err != nil {
		return err
	}
	h, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}
	h.Name, h.Method = rel, zip.Deflate
	out, err := zw.CreateHeader(h)
	if err == nil {
		_, err = io.Copy(out, src)
	}
	return err
}

func fileDigest(p string) (string, int64, error) {
	f, err := os.Open(p)
	if err != nil {
		return "", 0, err
	}
	defer f.Close()
	h := sha256.New()
	n, err := io.Copy(h, f)
	return hex.EncodeToString(h.Sum(nil)), n, err
}

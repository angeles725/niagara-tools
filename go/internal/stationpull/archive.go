package stationpull

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
)

// Zip writes the files under root, less the excluded ones, to archive as <prefix>/<relative path>. It
// returns how many files it zipped and how many it left out; a failed write leaves no archive behind.
func Zip(root, archive, prefix string, ex Excludes) (zipped, excluded int, err error) {
	files, err := copied(root)
	if err != nil {
		return 0, 0, err
	}
	out, err := os.OpenFile(archive, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return 0, 0, err
	}
	zw := zip.NewWriter(out)
	for _, rel := range files {
		if ex.Excluded(rel) {
			excluded++
			continue
		}
		if err = addFile(zw, filepath.Join(root, filepath.FromSlash(rel)), prefix+"/"+rel); err != nil {
			break
		}
		zipped++
	}
	if err == nil {
		err = zw.Close()
	}
	if cerr := out.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		os.Remove(archive)
		return 0, 0, err
	}
	return zipped, excluded, nil
}

func addFile(zw *zip.Writer, src, name string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	info, err := in.Stat()
	if err != nil {
		return err
	}
	h, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}
	h.Name, h.Method = name, zip.Deflate
	w, err := zw.CreateHeader(h)
	if err != nil {
		return err
	}
	_, err = io.Copy(w, in)
	return err
}

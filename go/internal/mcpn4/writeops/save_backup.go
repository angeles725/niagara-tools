package writeops

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/safety"
)

// The exclude lists of the station's BackupService (backup-rt BBackupService excludeFiles and
// excludeDirectories defaults, 4.15): file name globs anywhere, and the station-home directories
// that hold the history and alarm databases and the web cache.
var (
	backupExcludeFiles = []string{"*.hdb", "*.adb", "*.lock", "*backup*", "console.*", "config.bog.b*", "config_backup*"}
	backupExcludeDirs  = []string{"history", "alarm", "webFileCache"}
)

// backupSubdir is where the zips go under the state dir.
const backupSubdir = "backups"

// backupExcluded reports whether the station-home relative path rel (slash separated) is left out.
func backupExcluded(rel string, dir bool) bool {
	if dir {
		for _, d := range backupExcludeDirs {
			if rel == d {
				return true
			}
		}
		return false
	}
	base := path.Base(rel)
	for _, g := range backupExcludeFiles {
		if ok, _ := path.Match(g, base); ok {
			return true
		}
	}
	return false
}

// zipStationHome zips home into <stateDir>/backups/<station>-<UTC time>.zip (0600, never over an
// existing file; a failure removes the partial zip) and returns what it wrote: path, files, bytes
// of the zip, its sha256, and the excluded files and directories. Links are not followed: a
// symbolic link is listed as skipped.
func zipStationHome(home, stateDir, station string, now time.Time) (*pyjson.Object, error) {
	dir := safety.Join(stateDir, backupSubdir)
	if err := safety.PrivateDir(dir); err != nil {
		return nil, err
	}
	dest := safety.Join(dir, station+"-"+now.UTC().Format("20060102T150405Z")+".zip")
	f, err := os.OpenFile(dest, os.O_WRONLY|os.O_CREATE|os.O_EXCL|syscall.O_CLOEXEC, 0o600)
	if err != nil {
		return nil, err
	}
	files, excludedFiles, excludedDirs, skipped := 0, []any{}, []any{}, []any{}
	zw := zip.NewWriter(f)
	err = filepath.WalkDir(home, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(home, p)
		rel = filepath.ToSlash(rel)
		switch {
		case rel == ".":
			return nil
		case d.Type()&fs.ModeSymlink != 0 || !d.IsDir() && !d.Type().IsRegular():
			skipped = append(skipped, rel)
			return nil
		case backupExcluded(rel, d.IsDir()):
			if d.IsDir() {
				excludedDirs = append(excludedDirs, rel)
				return fs.SkipDir
			}
			excludedFiles = append(excludedFiles, rel)
			return nil
		case d.IsDir():
			return nil
		}
		files++
		return addZipFile(zw, p, rel)
	})
	if cerr := zw.Close(); err == nil {
		err = cerr
	}
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		os.Remove(dest)
		return nil, err
	}
	sum, size, err := fileDigest(dest)
	if err != nil {
		return nil, err
	}
	return obj("path", dest, "files", files, "bytes", size, "sha256", sum, "excluded_files", excludedFiles,
		"excluded_dirs", excludedDirs, "skipped_links", skipped,
		"rules", obj("files", strings.Join(backupExcludeFiles, ";"), "dirs", strings.Join(backupExcludeDirs, ";"))), nil
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
	w, err := zw.CreateHeader(h)
	if err == nil {
		_, err = io.Copy(w, src)
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

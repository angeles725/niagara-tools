package writeops

import (
	"errors"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/safety"
)

// The default exclude lists of the BackupService (backup-rt BBackupService excludeFiles and
// excludeDirectories, 4.15): file name globs matched on the base name, and the station-home
// directories that hold the history and alarm databases and the web cache. A site may have
// changed its station's lists; these are the defaults.
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
// existing file) and returns what it wrote. home is resolved first (a symbolic link to the
// station directory is walked as that directory); a zip without config.bog is an error, and any
// error removes the zip. The destination directory must be a private directory, not a link.
func zipStationHome(home, stateDir, station string, now time.Time) (*pyjson.Object, error) {
	root, err := filepath.EvalSymlinks(home)
	if err != nil {
		return nil, err
	}
	dir := safety.Join(stateDir, backupSubdir)
	if err := backupDir(dir); err != nil {
		return nil, err
	}
	skip, _ := filepath.EvalSymlinks(stateDir) // a state dir inside the station home is not zipped
	dest := safety.Join(dir, station+"-"+now.UTC().Format("20060102T150405Z")+".zip")
	w, err := walkZip(root, dest, skip)
	if err == nil && !w.hasBog {
		err = errors.New("no config.bog in " + root + ": --station-home must be the station directory")
	}
	var sum string
	var size int64
	if err == nil {
		sum, size, err = fileDigest(dest)
	}
	if err != nil {
		if w == nil || !w.existed {
			os.Remove(dest)
		}
		return nil, err
	}
	return obj("path", dest, "files", w.files, "bytes", size, "sha256", sum, "excluded_files", w.excludedFiles,
		"excluded_dirs", w.excludedDirs, "skipped_links", w.skipped,
		"rules", obj("files", strings.Join(backupExcludeFiles, ";"), "dirs", strings.Join(backupExcludeDirs, ";"))), nil
}

// backupDir makes dir 0700, or accepts an existing directory that is not a link and that no group
// or other user can read or write.
func backupDir(dir string) error {
	info, err := os.Lstat(dir)
	if os.IsNotExist(err) {
		return safety.PrivateDir(dir)
	}
	switch {
	case err != nil:
		return err
	case !info.IsDir():
		return errors.New(dir + " is not a directory (a link or a file): refusing to write the backup there")
	case info.Mode().Perm()&0o077 != 0:
		return errors.New(dir + " is open to other users (mode " + info.Mode().Perm().String() + "): chmod 700 it first")
	}
	return nil
}

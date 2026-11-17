package readtools

import (
	"errors"
	"os"
	"strings"
	"syscall"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/safety"
	"github.com/angeles725/niagara-tools/go/internal/strerror"
)

// The local file of n4_export_history: its name, its directory and its CSV lines.

// exportLine is one CSV record of an export (excel dialect, as n4_bql_query's output_file).
func exportLine(id, timestamp, value, status string) string {
	return csvLine([]string{id, timestamp, value, status})
}

// exportFileName is <station>-<UTC time to the millisecond>.csv; a station name character that
// is not a letter, a digit, '_', '-' or '$' becomes '_', so the name never leaves its directory.
func exportFileName(station string, at time.Time) string {
	name := strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || strings.ContainsRune("_-$", r) {
			return r
		}
		return '_'
	}, station)
	if name == "" {
		name = "station"
	}
	return name + "-" + at.UTC().Format("20060102T150405.000Z") + ".csv"
}

// exportDir makes the state dir and dir 0700 when missing, or accepts an existing dir that is a
// real directory (not a link) no group or other user can read or write.
func exportDir(dir string) error {
	info, err := os.Lstat(dir)
	switch {
	case os.IsNotExist(err):
		if err := safety.PrivateDir(safety.Dirname(dir)); err != nil {
			return err
		}
		return safety.PrivateDir(dir)
	case err != nil:
		return err
	case !info.IsDir():
		return errors.New(dir + " is not a directory (a link or a file): refusing to write the export there")
	case info.Mode().Perm()&0o077 != 0:
		return errors.New(dir + " is open to other users (mode " + info.Mode().Perm().String() + "): chmod 700 it first")
	}
	return nil
}

// exportWriteFailed is the ToolError of an export file the OS refused.
func exportWriteFailed(path string, err error) error {
	why := err.Error()
	var errno syscall.Errno
	if errors.As(err, &errno) {
		why = strerror.Text(errno)
	}
	return toolError("the export file %s could not be written (%s); nothing was left behind", path, why)
}

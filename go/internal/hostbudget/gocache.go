package hostbudget

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
)

func gbString(b uint64) string { return fmt.Sprintf("%.1fGB", float64(b)/bytesPerGB) }

const cleanHint = "hint: go clean -cache"

// gocacheRow sums the apparent size of every regular file under the cache, walking at most
// GocacheMaxEntries entries. Apparent size, not allocated blocks: it is what the cache holds and
// what `go clean -cache` frees, and it keeps a sparse test fixture honest.
func gocacheRow(c Config) Row {
	limit := uint64(c.GocacheGB) * bytesPerGB
	switch c.GocacheDir {
	case "":
		return Row{Warn, "gocache", "?", gbString(limit), "(GOCACHE could not be resolved)"}
	case "off":
		return Row{Pass, "gocache", "off", gbString(limit), ""}
	}
	var size uint64
	entries, stopped, walkErr := 0, false, error(nil)
	err := filepath.WalkDir(c.GocacheDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if path == c.GocacheDir && errors.Is(err, fs.ErrNotExist) {
				return fs.SkipAll // a cache that was never built (or was just cleaned) is empty
			}
			walkErr = err
			if d != nil && d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if entries++; entries > c.GocacheMaxEntries {
			stopped = true
			return fs.SkipAll
		}
		if d.Type().IsRegular() {
			info, err := d.Info()
			if err != nil {
				walkErr = err
				return nil
			}
			size += uint64(info.Size())
		}
		return nil
	})
	if err != nil {
		walkErr = err
	}
	res := "gocache"
	used := gbString(size)
	switch {
	case stopped:
		return Row{Warn, res, ">=" + used, gbString(limit),
			fmt.Sprintf("(walk stopped after %d entries; %s) %s", c.GocacheMaxEntries, c.GocacheDir, cleanHint)}
	case walkErr != nil:
		return Row{Warn, res, ">=" + used, gbString(limit), fmt.Sprintf("(walk incomplete: %v)", walkErr)}
	case size > limit:
		return Row{Warn, res, used, gbString(limit), fmt.Sprintf("(%s) %s", c.GocacheDir, cleanHint)}
	}
	return Row{Pass, res, used, gbString(limit), "(" + c.GocacheDir + ")"}
}

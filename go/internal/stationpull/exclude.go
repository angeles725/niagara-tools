package stationpull

import (
	"path"
	"strings"
)

// Excludes are glob patterns (* and ?) matched against a file's name and against each directory name of
// its path, case-sensitively.
type Excludes struct{ Files, Dirs []string }

// DefaultExcludes are the BackupService defaults: files that are live, regenerated or already backups,
// and the history, alarm, web cache and recovery directories.
func DefaultExcludes() Excludes {
	return Excludes{
		Files: []string{"*.hdb", "*.adb", "*.lock", "*backup*", "console.*", "config.bog.b*", "config_backup*"},
		Dirs:  []string{"history", "alarm", "webFileCache", "dataRecovery"},
	}
}

// SplitList splits a ';' separated flag value, dropping blank items.
func SplitList(s string) []string {
	var out []string
	for _, item := range strings.Split(s, ";") {
		if item = strings.TrimSpace(item); item != "" {
			out = append(out, item)
		}
	}
	return out
}

// Validate refuses a malformed pattern.
func (e Excludes) Validate() error {
	for _, p := range append(append([]string{}, e.Files...), e.Dirs...) {
		if _, err := path.Match(p, ""); err != nil {
			return err
		}
	}
	return nil
}

func matchAny(patterns []string, name string) bool {
	for _, p := range patterns {
		if ok, _ := path.Match(p, name); ok {
			return true
		}
	}
	return false
}

// Excluded reports whether the file at rel (slash separated, relative to the station) is left out.
func (e Excludes) Excluded(rel string) bool {
	parts := strings.Split(rel, "/")
	for _, dir := range parts[:len(parts)-1] {
		if matchAny(e.Dirs, dir) {
			return true
		}
	}
	return matchAny(e.Files, parts[len(parts)-1])
}

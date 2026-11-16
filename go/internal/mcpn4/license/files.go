package license

import (
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyxml"
)

// File is what a licence file states in the attributes of its root element, "" for one it does
// not have. Nothing else of the file is kept: not its features, not its signature.
type File struct {
	Generated, Version, MaintenanceExpiration string
	hostID                                    string
}

// ForHost reports whether the file was issued for the host with this id.
func (f File) ForHost(hostID string) bool { return f.hostID != "" && f.hostID == hostID }

// ReadFiles reads, from the Niagara installation at home (LicensesDir below it), the licence
// files named wanted: the names come from the page, and only a directory entry of exactly that
// name is opened. files holds the ones read; skipped says, per wanted name that is there but
// was not read, why (a Skip constant). A wanted name in neither is not in the directory. err
// is the directory not being listable.
func ReadFiles(home string, wanted []string) (files map[string]File, skipped map[string]string, err error) {
	dir := filepath.Join(home, filepath.FromSlash(LicensesDir))
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, nil, err
	}
	exact, folded := map[string]bool{}, map[string]bool{}
	for _, e := range entries {
		exact[e.Name()], folded[strings.ToLower(e.Name())] = true, true
	}
	files, skipped = map[string]File{}, map[string]string{}
	opened := 0
	for _, name := range wanted {
		_, seen := files[name]
		switch {
		case seen || skipped[name] != "":
		case name != filepath.Base(name) || !strings.HasSuffix(name, fileSuffix): // never a path, never another kind of file
		case !exact[name]:
			if folded[strings.ToLower(name)] {
				skipped[name] = SkipCase
			}
		case opened >= MaxFiles:
			skipped[name] = SkipCap
		default:
			opened++
			if f, why := readFile(filepath.Join(dir, name)); why != "" {
				skipped[name] = why
			} else {
				files[name] = f
			}
		}
	}
	return files, skipped, nil
}

// readFile is the root attributes of one licence file, or why it was not read.
func readFile(path string) (File, string) {
	f, why := openRegular(path)
	if why != "" {
		return File{}, why
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, maxFileBytes+1))
	if err != nil {
		return File{}, SkipUnreadable
	}
	if len(data) > maxFileBytes {
		return File{}, SkipLarge
	}
	root, err := pyxml.Parse(strings.TrimPrefix(string(data), utf8BOM))
	if err != nil || root.Tag != rootTag {
		return File{}, SkipNotLicense
	}
	get := func(name string) string {
		v, _ := root.Get(name)
		return Clip(Clean(v), MaxNameRunes)
	}
	return File{Generated: get(generatedAttr), Version: get(versionAttr),
		MaintenanceExpiration: get(maintenanceAttr), hostID: get(hostIDAttr)}, ""
}

package stationpull

import (
	"fmt"
	"hash/crc32"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

// copied lists the regular files under root as slash separated relative paths.
func copied(root string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil || !d.Type().IsRegular() {
			return err
		}
		rel, err := filepath.Rel(root, p)
		files = append(files, filepath.ToSlash(rel))
		return err
	})
	return files, err
}

func fileCRC(p string) (uint32, error) {
	f, err := os.Open(p)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	h := crc32.NewIEEE()
	_, err = io.Copy(h, f)
	return h.Sum32(), err
}

// Verify checks every listed row against the files under root and returns how many CRCs matched plus a
// sorted list of problems. A row matches by name: its path when it has directories, else the file name
// wherever the file sits. Rows with the same name are matched one to one with the copied files of that
// name. A row whose CRC is 0 (not computed) needs the file to exist but is not counted as verified.
func Verify(root string, rows []Row) (int, []string) {
	files, err := copied(root)
	if err != nil {
		return 0, []string{"cannot read the copy: " + err.Error()}
	}
	byName := map[string][]Row{}
	for _, r := range rows {
		n := strings.TrimPrefix(strings.TrimPrefix(strings.ReplaceAll(r.Name, `\`, "/"), "./"), "/")
		byName[n] = append(byName[n], r)
	}
	verified, problems := 0, []string{}
	for name, rs := range byName {
		var local []string
		for _, f := range files {
			if f == name || strings.HasSuffix(f, "/"+name) || (!strings.Contains(name, "/") && path.Base(f) == name) {
				local = append(local, f)
			}
		}
		v, p := matchRows(root, name, rs, local)
		verified += v
		problems = append(problems, p...)
	}
	sort.Strings(problems)
	return verified, problems
}

func matchRows(root, name string, rows []Row, local []string) (int, []string) {
	if len(local) == 0 {
		return 0, []string{"missing: " + name}
	}
	sums := make([]uint32, len(local))
	used := make([]bool, len(local))
	for i, f := range local {
		var err error
		if sums[i], err = fileCRC(filepath.Join(root, filepath.FromSlash(f))); err != nil {
			return 0, []string{"unreadable: " + f + ": " + err.Error()}
		}
	}
	verified, problems := 0, []string(nil)
	for _, r := range rows {
		if r.CRC == 0 {
			continue
		}
		hit := false
		for i := range local {
			if !used[i] && sums[i] == r.CRC {
				used[i], hit = true, true
				break
			}
		}
		if hit {
			verified++
			continue
		}
		problems = append(problems, fmt.Sprintf("crc mismatch: %s (listed %d, copied %s)", name, r.CRC, listCRCs(sums)))
	}
	return verified, problems
}

func listCRCs(sums []uint32) string {
	var out []string
	for _, s := range sums {
		out = append(out, fmt.Sprint(s))
	}
	return strings.Join(out, ", ")
}

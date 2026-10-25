package niagaratest

import (
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// userHomeSnapshot records what the NRE writes in a real Niagara user home when it ignores the scratch
// niagara_user_home: every entry under %USERPROFILE%\Niagara4.*\<brand>\registry and \security, keyed by its
// Windows path (files by size and mtime, dirs by name). The runner compares it before the copy and after
// cleanup: the sandbox is proven used by the real location staying unchanged, not by the knob being set.
func userHomeSnapshot(prof winDir) (snapshot, error) {
	s := snapshot{}
	for _, sub := range []string{"registry", "security"} {
		roots, err := filepath.Glob(filepath.Join(prof.wsl, "Niagara4.*", "*", sub))
		if err != nil {
			return nil, err
		}
		for _, root := range roots {
			err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
				if err != nil {
					return err
				}
				rel, _ := filepath.Rel(prof.wsl, p)
				k, v := prof.win+`\`+strings.ReplaceAll(rel, "/", `\`), "dir"
				if !d.IsDir() {
					info, err := d.Info()
					if err != nil {
						return err
					}
					v = info.ModTime().UTC().String() + " " + strconv.FormatInt(info.Size(), 10) + " bytes"
				}
				s[k] = v
				return nil
			})
			if err != nil {
				return nil, err
			}
		}
	}
	return s, nil
}

func isDir(p string) bool {
	info, err := os.Stat(p)
	return err == nil && info.IsDir()
}

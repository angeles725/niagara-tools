package signinginventory

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/angeles725/niagara-tools/go/internal/modjar"
	"github.com/angeles725/niagara-tools/go/internal/niagaraversion"
)

// userDirs are the places of the user's home that hold signing profiles: the plugin's default profile
// directory and the one the kit's signing notes use (<alias>.signing.xml).
var userDirs = []string{"/.tridium/security", "/.config/n4-signing"}

// homeDirs are the places of a Niagara installation searched for profiles.
var homeDirs = []string{"/etc", "/security"}

// discover is the profile files to list, in order, without duplicates: the --profile files as given, else the
// user's places; and the places of each --home, whose version is printed. A --profile or --home that is
// not there is an unreadable input (an error), a place with nothing in it is not.
func (r *run) discover(o options) ([]string, error) {
	var out []string
	seen := map[string]bool{}
	add := func(p string) {
		if abs, err := filepath.Abs(p); err == nil && !seen[abs] {
			seen[abs] = true
			out = append(out, abs)
		}
	}
	for _, p := range o.profiles {
		if err := exists(p, false); err != nil {
			return nil, err
		}
		add(p)
	}
	if len(o.profiles) == 0 {
		if home := modjar.HomeDir(r.s.Env); home != "" {
			for _, d := range userDirs {
				for _, p := range profilesIn(home + d) {
					add(p)
				}
			}
		}
	}
	for _, h := range o.homes {
		if err := exists(h, true); err != nil {
			return nil, err
		}
		r.install(h)
		for _, d := range homeDirs {
			for _, p := range profilesIn(h + d) {
				add(p)
			}
		}
	}
	if len(out) == 0 {
		r.add("INFO", "profiles", "no signing profile found in the places searched (see --help); name one with --profile")
	}
	return out, nil
}

// exists is nil when p is a directory (dir) or a regular file, else the error naming p.
func exists(p string, dir bool) error {
	st, err := os.Stat(p)
	switch {
	case err != nil:
		return unreadable(err)
	case dir && !st.IsDir():
		return fmt.Errorf("%s: not a directory", p)
	case !dir && !st.Mode().IsRegular():
		return fmt.Errorf("%s: not a file", p)
	}
	return nil
}

// profilesIn is the *.signing.xml files directly in dir.
func profilesIn(dir string) []string {
	files, _ := filepath.Glob(filepath.Join(dir, "*.signing.xml"))
	return files
}

// install is the INFO row of a --home: its Niagara version, or the WARN that it holds no baja.jar.
func (r *run) install(home string) {
	version, root, err := niagaraversion.InstallVersion(home)
	if err != nil {
		r.add("WARN", "install", "%s: %v", home, err)
		return
	}
	r.add("INFO", "install", "%s  Niagara %s", root, version)
}

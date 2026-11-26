// Package stationpull makes a restorable copy of a station the platform daemon manages (issue #1169):
// plat fget -recurse copies ~stations/NAME, the CRCs `plat flist -recurse` lists are checked against the
// copied files, and the copy is zipped without the BackupService's default exclusions. Measured on a
// 4.14.0.162 lab host: plat exits 0 and prints a `niagarad version ?` exception after a good copy, so
// success is judged from the files, never from plat's exit code. The credential comes from the same
// mode-0600 env file as plat-read and is masked in everything printed.
package stationpull

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/platread"
)

const (
	exitUsage = 3
	usage     = "usage: station-pull [--env FILE] [--home DIR] [--dry-run] [--exclude-files 'a;b'] " +
		"[--exclude-dirs 'x;y'] --station NAME --dest DIR"
)

var stationName = regexp.MustCompile(`^[A-Za-z0-9_.-]+$`)

// now is the archive timestamp source (a test replaces it).
var now = time.Now

type options struct {
	env, home, station, dest string
	dry                      bool
	ex                       Excludes
}

// Main is the station-pull command.
func Main(args []string, s cli.Stdio) int {
	o := options{env: platread.DefaultEnv(s), home: s.Env("NIAGARA_HOME"), ex: DefaultExcludes()}
	if msg := o.parse(args); msg != "" {
		return bad(s, msg)
	}
	if !stationName.MatchString(o.station) {
		return bad(s, "--station must match ^[A-Za-z0-9_.-]+$")
	}
	if o.dest == "" {
		return bad(s, "--dest is required")
	}
	if o.home == "" {
		return bad(s, "no Niagara home: set NIAGARA_HOME or pass --home")
	}
	if err := o.ex.Validate(); err != nil {
		return bad(s, "bad exclude pattern: "+err.Error())
	}
	dest, err := filepath.Abs(o.dest)
	if err == nil {
		err = checkDest(dest)
	}
	if err != nil {
		return bad(s, "--dest: "+err.Error())
	}
	cr, err := platread.LoadEnv(o.env)
	if err != nil {
		return bad(s, err.Error())
	}
	exe := platread.PlatExe(o.home)
	local, err := NativePath(filepath.Join(dest, "station"), exe)
	if err != nil {
		return bad(s, "--dest: "+err.Error())
	}
	p := puller{s: s, cr: cr, exe: exe, o: o, dest: dest, local: local}
	if o.dry {
		fmt.Fprintln(s.Out, "plat "+strings.Join(p.fget(true), " "))
		fmt.Fprintln(s.Out, "plat "+strings.Join(p.flist(true), " "))
		return 0
	}
	return p.pull()
}

func (o *options) parse(args []string) string {
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--dry-run" {
			o.dry = true
			continue
		}
		var target *string
		switch a {
		case "--env":
			target = &o.env
		case "--home":
			target = &o.home
		case "--station":
			target = &o.station
		case "--dest":
			target = &o.dest
		case "--exclude-files", "--exclude-dirs":
		default:
			return "unexpected argument " + a
		}
		if i++; i >= len(args) {
			return a + " needs a value"
		}
		switch {
		case target != nil:
			*target = args[i]
		case a == "--exclude-files":
			o.ex.Files = SplitList(args[i])
		default:
			o.ex.Dirs = SplitList(args[i])
		}
	}
	return ""
}

func bad(s cli.Stdio, msg string) int {
	fmt.Fprintln(s.Err, "station-pull: "+msg)
	fmt.Fprintln(s.Err, usage)
	return exitUsage
}

// checkDest accepts a missing path or an empty directory.
func checkDest(dest string) error {
	info, err := os.Stat(dest)
	switch {
	case os.IsNotExist(err):
		return nil
	case err != nil:
		return err
	case !info.IsDir():
		return fmt.Errorf("%s is not a directory", dest)
	}
	entries, err := os.ReadDir(dest)
	if err == nil && len(entries) > 0 {
		return fmt.Errorf("%s is not empty (a pull needs a fresh directory)", dest)
	}
	return err
}

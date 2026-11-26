// Package platread runs one read-only plat verb against a Niagara platform daemon (issue #850), with the
// daemon credential read from a mode-0600 env file and never printed. Live on a 4.14.0.162 lab host:
// `plat details -secure` and `plat liststations` answer over TLS port 5011. plat reads a missing credential
// with System.console(), which is null when stdin is not a terminal (BAbstractPlatformCommand), so the
// credential can only reach plat on its command line: -usr: and -pwd:, with -noinput so a wrong one fails.
package platread

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

const (
	exitUsage = 3
	usage     = "usage: plat-read [--env FILE] [--home DIR] [--dry-run] details|liststations|ipconfig"
	masked    = "<masked>"
)

// readVerbs are the plat verbs that only read the host (B1298); the others start, stop, install, write
// files, set the time or tell a station, and this command refuses them.
var readVerbs = map[string]bool{"details": true, "liststations": true, "ipconfig": true}

type call struct {
	verb, home, env string
	dry             bool
}

// Main is the plat-read command.
func Main(args []string, s cli.Stdio) int {
	c := call{home: s.Env("NIAGARA_HOME"), env: s.Env("NIAGARA_PLAT_ENV")}
	if c.env == "" {
		if h, err := os.UserHomeDir(); err == nil {
			c.env = filepath.Join(h, ".config", "niagara-private", "lab-platform.env")
		}
	}
	for i := 0; i < len(args); i++ {
		switch a := args[i]; {
		case a == "--dry-run":
			c.dry = true
		case (a == "--env" || a == "--home") && i+1 < len(args):
			if i++; a == "--env" {
				c.env = args[i]
			} else {
				c.home = args[i]
			}
		case strings.HasPrefix(a, "-") || c.verb != "":
			return bad(s, "unexpected argument "+a)
		default:
			c.verb = a
		}
	}
	if c.verb == "" {
		return bad(s, "")
	}
	if !readVerbs[c.verb] {
		return bad(s, "plat-read only reads: "+c.verb+" changes the host or a station; use Workbench or plat directly")
	}
	if c.home == "" {
		return bad(s, "no Niagara home: set NIAGARA_HOME or pass --home")
	}
	cred, err := loadEnv(c.env)
	if err != nil {
		return bad(s, err.Error())
	}
	return c.run(s, cred)
}

func bad(s cli.Stdio, msg string) int {
	if msg != "" {
		fmt.Fprintln(s.Err, "plat-read: "+msg)
	}
	fmt.Fprintln(s.Err, usage)
	return exitUsage
}

// argv is the plat command line; shown masks the password.
func (c call) argv(cr cred, shown bool) []string {
	pwd := cr.password
	if shown {
		pwd = masked
	}
	return []string{c.verb, "-h:ip:" + cr.host, "-p:" + cr.port, "-secure", "-noinput", "-usr:" + cr.user, "-pwd:" + pwd}
}

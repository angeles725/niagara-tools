// Package signinginventory is the read-only inventory of the signing material a developer has: for every
// signing profile it can find, each keystore alias with its certificate (subject, issuer, days left, code
// signing or not, the file it lives in), and with --module what the module records (expected_signer, the
// signer and the Niagara version of its built jars). It never prints a password, a key or a keystore
// secret: a store it cannot open without one is listed as locked, with its path only. (issue #1160)
package signinginventory

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/signingdoctor"
)

// Exit codes of signing-inventory.
const (
	ExitOK    = 0 // every row is PASS, INFO or WARN
	ExitFail  = 1 // a FAIL row
	ExitUsage = 2 // bad arguments or an unreadable input
)

// ToolbeltEnv is set by the wrapper to the kit's toolbelt directory: the kit's BUILD-STATE.src.md is
// ../BUILD-STATE.src.md from it.
const ToolbeltEnv = "SIGNING_INVENTORY_TOOLBELT"

const usage = `usage: signing-inventory [--home <niagara_home>]... [--profile <profile.xml>]...
                         [--module <module-root> [--module-name <MOD>] [--build-state <file>]]
Read-only inventory of the signing profiles and keystores: per alias the certificate subject and issuer, the
days left (WARN under 60, FAIL expired), whether it is a code-signing certificate (not a CA) and the file it
lives in. A store that needs a password the profile does not hold is listed as locked, with its path only.
Never prints a password or a key.
Places searched (no disk scan): --profile files as given; else the user's ~/.tridium/security/*.signing.xml
and ~/.config/n4-signing/*.signing.xml; for each --home its version (modules/baja.jar) and its etc/ and
security/ *.signing.xml. The keystore is listed with the keytool of $JAVA_HOME, else the one on PATH.
--module: also the expected_signer recorded in BUILD-STATE (--build-state, else <module-root>/BUILD-STATE.md,
else the kit's BUILD-STATE.src.md) for module --module-name (default: the directory name), the signer and the
Niagara version of each <module-root>/**/build/libs/*.jar. FAIL when the expected signer is in no listed store
or only expired, or a built jar is signed by another.
Rows: PASS|WARN|FAIL|INFO, a check id, the detail. Exit 0 / 1 a FAIL row / 2 usage or unreadable input.
`

type options struct {
	homes, profiles          []string
	module, name, buildState string
}

// Main implements signing-inventory.
func Main(args []string, s cli.Stdio) int {
	var o options
	fs := flag.NewFlagSet("signing-inventory", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	// A given but empty option (an unset variable in --home "$H") is refused, never read as "not given".
	empty := ""
	list := func(dst *[]string, name string) func(string) error {
		return func(v string) error {
			if v == "" && empty == "" {
				empty = name
			}
			*dst = append(*dst, v)
			return nil
		}
	}
	fs.Func("home", "", list(&o.homes, "home"))
	fs.Func("profile", "", list(&o.profiles, "profile"))
	fs.StringVar(&o.module, "module", "", "")
	fs.StringVar(&o.name, "module-name", "", "")
	fs.StringVar(&o.buildState, "build-state", "", "")
	if err := fs.Parse(args); err != nil || fs.NArg() != 0 {
		fmt.Fprint(s.Err, usage)
		return ExitUsage
	}
	fs.Visit(func(f *flag.Flag) {
		if g, ok := f.Value.(flag.Getter); ok && g.Get() == "" && empty == "" {
			empty = f.Name
		}
	})
	if empty != "" {
		fmt.Fprintf(s.Err, "signing-inventory: --%s is empty\n", empty)
		return ExitUsage
	}
	if o.module == "" && (o.name != "" || o.buildState != "") {
		fmt.Fprint(s.Err, "signing-inventory: --module-name and --build-state need --module\n")
		return ExitUsage
	}
	r := &run{s: s, now: time.Now(), javaHome: s.Env("JAVA_HOME")}
	if err := r.inventory(o); err != nil {
		fmt.Fprintf(s.Err, "signing-inventory: %v\n", err)
		return ExitUsage
	}
	code := ExitOK
	for _, row := range r.rows {
		fmt.Fprintf(s.Out, "%-4s  %s  %s\n", row.Status, row.Check, row.Detail)
		if row.Status == "FAIL" {
			code = ExitFail
		}
	}
	return code
}

// run is one inventory: the rows so far and the aliases found in the stores it could list.
type run struct {
	s        cli.Stdio
	now      time.Time
	javaHome string
	rows     []signingdoctor.Row
	found    []alias
	listed   int    // stores listed
	locked   int    // stores that could not be listed
	target   string // the module's recorded niagara_target, "" when none
}

func (r *run) add(status, check, format string, a ...any) {
	r.rows = append(r.rows, signingdoctor.Row{Status: status, Check: check, Detail: fmt.Sprintf(format, a...)})
}

// inventory discovers the profiles, lists them, then reads the module. An error is an unreadable input.
func (r *run) inventory(o options) error {
	profiles, err := r.discover(o)
	if err != nil {
		return err
	}
	for _, p := range profiles {
		r.profile(p)
	}
	if o.module == "" {
		return nil
	}
	return r.module(o)
}

// unreadable is the error for an input that is not there; the OS error names the path and the cause.
func unreadable(err error) error {
	var pe *os.PathError
	if errors.As(err, &pe) {
		return fmt.Errorf("%s: %s", pe.Path, strings.TrimPrefix(pe.Err.Error(), "syscall: "))
	}
	return err
}

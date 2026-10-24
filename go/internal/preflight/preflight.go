// Package preflight is the BUILD-LOOP §0.b environment preflight of toolbelt/preflight.sh: before a
// gradle build it checks the paths (win-path), the JDK (jdk8), the Niagara plugin pin (plugin-pin),
// module jars held by a running station (jar-lock), a sibling checkout at a higher module version
// (version-drift) and, with --sample, the retarget of a project someone else shipped (sample-*).
// Every check prints one `PASS|FAIL|WARN|SKIP  <check>  <detail>` row and the run continues after a
// FAIL, so one run shows every problem. A check is one func in the checks list: a new row (BUILD-STATE
// G95 shared-host) is one more entry.
package preflight

import (
	"bytes"
	"fmt"
	"os/exec"
	"regexp"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/lint"
)

// Exit codes of preflight.sh.
const (
	ExitOK    = 0 // every row PASS, WARN or SKIP
	ExitFail  = 1 // any FAIL row
	ExitUsage = 2
	ExitEnv   = 3 // niagara_home or the gradle root is not a directory
)

const usage = "usage: preflight.sh [--jvm-dir <d>] [--plugin-version <v>] [--sample] <niagara_home> <gradle-root>\n"

// quotedVersion is the first "X.Y.Z" string that names a version (plugin pin, defaultModuleVersion).
var quotedVersion = regexp.MustCompile(`"[0-9]+\.[0-9]+\.[0-9]+"`)

// Host runs the external commands preflight shells out to (lsof, java -version, df); tests replace it.
type Host struct {
	// Output runs name with args and returns its stdout, with stderr interleaved when combined. A
	// command that cannot start or fails returns whatever it printed, possibly nothing.
	Output func(name string, args []string, combined bool) []byte
}

// System is the Host of the real process.
var System = Host{Output: func(name string, args []string, combined bool) []byte {
	var b bytes.Buffer
	cmd := exec.Command(name, args...)
	cmd.Stdout = &b
	if combined {
		cmd.Stderr = &b
	}
	_ = cmd.Run() // a failed run is judged by its output, as the bash `... || true` did
	return b.Bytes()
}}

// Main is preflight with the System host.
func Main(args []string, s cli.Stdio) int { return Command(System)(args, s) }

// run is one invocation: its options, its output and whether a FAIL row was printed.
type run struct {
	s                   cli.Stdio
	h                   Host
	jvmDir, plugin      string
	sample, failed, win bool
	nh, gr              string
}

// checks run in this order after the win-path rows; a check reading the paths returns early on r.win.
var checks = []func(*run){(*run).jdk8}

// Command is preflight over the host h.
func Command(h Host) cli.Main {
	return func(args []string, s cli.Stdio) int {
		r := &run{s: s, h: h, jvmDir: "/usr/lib/jvm", plugin: s.Env("NIAGARA_PLUGIN_VERSION")}
		pos, ok := r.parse(args)
		if !ok || len(pos) != 2 {
			fmt.Fprint(s.Err, usage)
			return ExitUsage
		}
		r.nh, r.gr = pos[0], pos[1]
		for _, p := range []struct{ name, path string }{{"niagara_home", r.nh}, {"gradle-root", r.gr}} {
			if strings.Contains(p.path, "C:") || strings.Contains(p.path, `\`) {
				r.row("FAIL", "win-path", p.name+" is a Windows path — use /mnt/c/... in WSL instead")
				r.win = true
			}
		}
		if !r.win {
			for _, p := range []struct{ name, path string }{{"niagara_home", r.nh}, {"gradle-root", r.gr}} {
				if !lint.IsDir(p.path) {
					fmt.Fprintf(s.Err, "preflight: %s not found: %s\n", p.name, p.path)
					return ExitEnv
				}
			}
		}
		for _, c := range checks {
			c(r)
		}
		if r.failed {
			return ExitFail
		}
		return ExitOK
	}
}

// parse reads the options up to the first operand (or --) and returns the operands; false is a usage error.
func (r *run) parse(args []string) ([]string, bool) {
	for len(args) > 0 {
		switch a := args[0]; {
		case a == "--jvm-dir" || a == "--plugin-version":
			if len(args) < 2 {
				return nil, false
			}
			if a == "--jvm-dir" {
				r.jvmDir = args[1]
			} else {
				r.plugin = args[1]
			}
			args = args[2:]
		case a == "--sample":
			r.sample, args = true, args[1:]
		case a == "--":
			return args[1:], true
		case strings.HasPrefix(a, "-"):
			return nil, false
		default:
			return args, true
		}
	}
	return args, true
}

// row prints `%-4s  %-12s  %s`; a FAIL makes the run exit 1.
func (r *run) row(status, check, detail string) {
	fmt.Fprintf(r.s.Out, "%-4s  %-12s  %s\n", status, check, detail)
	if status == "FAIL" {
		r.failed = true
	}
}

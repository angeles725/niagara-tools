// Package labstation starts, stops and restarts a local lab station without an interactive console (issue #848), so
// a module build loop can install a jar and restart the station from a script. The station reads its console from a
// named pipe the station itself holds open (O_RDWR, so it never sees end of file): stop writes quit to it and waits
// for the shutdown line, and console sends one read-only verb and prints its answer. On a 4.14 lab station from WSL
// (station.exe through interop) quit stopped the station in about 9 s: Station stopped, then *** Station shutdown ***.
package labstation

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

const (
	exitFail  = 1
	exitUsage = 3
	usage     = "usage: lab-station start|stop|restart|status [--home DIR] [--state DIR] [--timeout SECONDS] " +
		"[--dry-run] <station> | console <station> <verb>"
	startedMark  = "*** Station Started"
	shutdownMark = "*** Station shutdown ***"
)

var stationName = regexp.MustCompile(`^[A-Za-z0-9_]+$`)

// consoleVerbs are the console verbs that only read (the 4.14 console's help lists help, version, save, quit, kill,
// threads, gc and heap); save, quit and kill change state and are left out.
var consoleVerbs = map[string]bool{"gc": true, "heap": true, "help": true, "threads": true, "version": true}

type lab struct {
	name, home, exe, dir string
	timeout              time.Duration
	s                    cli.Stdio
}

func (l *lab) path(f string) string { return filepath.Join(l.dir, f) }

// Main is the lab-station command.
func Main(args []string, s cli.Stdio) int {
	getenv := s.Getenv
	if getenv == nil {
		getenv = os.Getenv
	}
	if len(args) == 0 {
		return bad(s, "")
	}
	cmd, l, dry := args[0], &lab{home: getenv("NIAGARA_HOME"), timeout: 180 * time.Second, s: s}, false
	state := getenv("LAB_STATION_STATE")
	if state == "" {
		state = filepath.Join(os.TempDir(), "lab-station")
	}
	var pos []string
	for i := 1; i < len(args); i++ {
		switch a := args[i]; a {
		case "--dry-run":
			dry = true
		case "--home", "--state", "--timeout":
			if i+1 >= len(args) {
				return bad(s, a+" needs a value")
			}
			i++
			switch a {
			case "--home":
				l.home = args[i]
			case "--state":
				state = args[i]
			default:
				n, err := strconv.Atoi(args[i])
				if err != nil || n < 1 {
					return bad(s, "--timeout takes whole seconds")
				}
				l.timeout = time.Duration(n) * time.Second
			}
		default:
			pos = append(pos, a)
		}
	}
	want := 1
	if cmd == "console" {
		want = 2
	}
	if len(pos) != want || !stationName.MatchString(pos[0]) {
		return bad(s, "")
	}
	l.name, l.dir = pos[0], filepath.Join(state, pos[0])
	if cmd == "console" && !consoleVerbs[pos[1]] {
		fmt.Fprintf(s.Err, "lab-station: console verb %s is refused (allowed: gc heap help threads version; "+
			"stop with lab-station stop)\n", pos[1])
		return exitUsage
	}
	switch cmd {
	case "start", "stop", "restart", "status", "console":
	default:
		return bad(s, "unknown command "+cmd)
	}
	if cmd == "start" || cmd == "restart" {
		if l.exe = findExe(l.home); l.exe == "" {
			fmt.Fprintf(s.Err, "lab-station: no bin/station.exe or bin/station under %q (set --home or NIAGARA_HOME)\n", l.home)
			return exitUsage
		}
	}
	if dry {
		l.plan(cmd)
		return 0
	}
	switch cmd {
	case "start":
		return l.start()
	case "stop":
		return l.stop()
	case "restart":
		if _, ok := l.running(); ok {
			if c := l.stop(); c != 0 {
				return c
			}
		}
		return l.start()
	case "status":
		return l.status()
	}
	return l.console(pos[1])
}

func bad(s cli.Stdio, why string) int {
	if why != "" {
		fmt.Fprintf(s.Err, "lab-station: %s\n", why)
	}
	fmt.Fprintln(s.Err, usage)
	return exitUsage
}

func findExe(home string) string {
	for _, n := range []string{"station.exe", "station"} {
		if p := filepath.Join(home, "bin", n); home != "" && fileExists(p) {
			return p
		}
	}
	return ""
}

func fileExists(p string) bool { _, err := os.Stat(p); return err == nil }

func (l *lab) plan(cmd string) {
	fmt.Fprintln(l.s.Out, "lab-station: dry run, nothing done")
	if cmd == "stop" || cmd == "restart" {
		fmt.Fprintf(l.s.Out, "  stop: write quit to %s, wait for %s\n", l.path("in"), shutdownMark)
	}
	if cmd == "start" || cmd == "restart" {
		fmt.Fprintf(l.s.Out, "  start: %s %s, stdin %s, log %s, wait for %s\n", l.exe, l.name, l.path("in"),
			l.path("out.log"), startedMark)
	}
	if cmd == "status" || cmd == "console" {
		fmt.Fprintf(l.s.Out, "  %s: read %s (reads only)\n", cmd, l.path("pid"))
	}
}

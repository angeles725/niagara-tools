// Package stationsnapshot is station-snapshot (toolbelt/station-snapshot.sh): it copies a station directory's
// config.bog and console*.txt into an out dir and writes manifest.json there (sha256, bytes, a UTC stamp, and
// history/** as size pointers that are never copied), without writing to the station dir. Output, files and
// exit codes are those of the bash script it replaces, including the messages its shell and coreutils printed.
package stationsnapshot

import (
	"fmt"
	"strings"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/gen"
)

// Exit codes of station-snapshot (unchanged from the bash tool).
const (
	ExitOK     = 0 // every listed file copied, manifest written
	ExitFailed = 1 // config.bog missing, or a listed file unreadable or not copied
	ExitUsage  = 3 // usage/env error (K20)
)

// ScriptEnv names the script path the bash shell put in front of its own error messages; the toolbelt wrapper
// exports its own path there.
const ScriptEnv = "STATION_SNAPSHOT_SCRIPT"

// Line numbers of the bash script that its shell error messages named.
const (
	lineCdStation = 55
	lineCdOut     = 61
	lineCdParent  = 63
	lineHistWc    = 140
	lineManifest  = 172
)

// Host is what station-snapshot reads besides its arguments and the filesystem: the clock.
type Host struct{ Now func() time.Time }

// Main is station-snapshot on the real host.
func Main(args []string, s cli.Stdio) int { return Command(Host{Now: time.Now})(args, s) }

// Command is station-snapshot over h.
func Command(h Host) cli.Main {
	return func(args []string, s cli.Stdio) int {
		if len(args) < 2 {
			fmt.Fprint(s.Err, "usage: station-snapshot.sh <station-dir> <out-dir>\n")
			return ExitUsage
		}
		stationDir, outDir := args[0], args[1]
		if !gen.IsDir(stationDir) {
			fmt.Fprintf(s.Err, "station-snapshot: not a directory: %s\n", stationDir)
			return ExitUsage
		}
		if !onPath("sha256sum", s.Env("PATH")) {
			fmt.Fprint(s.Err, "station-snapshot: sha256sum not found in PATH\n")
			return ExitUsage
		}
		r := run{s: s, script: s.Env(ScriptEnv), out: outDir}
		if r.script == "" {
			r.script = "station-snapshot.sh"
		}
		r.station = r.cd(stationDir, lineCdStation)
		if abs := r.outAbs(); abs == r.station || strings.HasPrefix(abs, r.station+"/") {
			fmt.Fprint(s.Err, "station-snapshot: <out-dir> must not be inside <station-dir>\n")
			return ExitUsage
		}
		if !mkdirP(outDir, s.Err) {
			fmt.Fprintf(s.Err, "station-snapshot: cannot create out-dir: %s\n", outDir)
			return ExitUsage
		}
		return r.snapshot(h.Now().UTC().Format("20060102T150405Z"))
	}
}

// run is one snapshot: the station's logical absolute path, the out dir as spelled, and what was recorded.
type run struct {
	s               cli.Stdio
	script, station string
	out             string
	files, pointers []string
	failed          bool
}

// outAbs is the path the out dir is checked against: its logical absolute path when it is a directory, else
// its parent's with its base name appended, else the out dir as spelled.
func (r *run) outAbs() string {
	parent, base := dirname(r.out), basename(r.out)
	switch {
	case gen.IsDir(r.out):
		return r.cd(r.out, lineCdOut)
	case gen.IsDir(parent):
		return r.cd(parent, lineCdParent) + "/" + base
	}
	return r.out
}

// cd is the bash `$(cd dir && pwd)`: dir's logical absolute path, or "" after the shell's cd message.
func (r *run) cd(dir string, line int) string {
	p, err := logicalCd(dir)
	if err != nil {
		r.shellErr(line, "cd: "+dir+": "+errText(err))
		return ""
	}
	return p
}

// shellErr prints a message as the bash shell did: "<script>: line <n>: <msg>".
func (r *run) shellErr(line int, msg string) {
	fmt.Fprintf(r.s.Err, "%s: line %d: %s\n", r.script, line, msg)
}

// row prints one result row.
func (r *run) row(status, rel, detail string) {
	fmt.Fprintf(r.s.Out, "%-4s  %-14s  %s  %s\n", status, "station-snap", rel, detail)
}

// snapshot copies config.bog and console*.txt, records history pointers, writes manifest.json and reports.
func (r *run) snapshot(stamp string) int {
	if isRegular(r.station + "/config.bog") {
		r.snapFile("config.bog")
	} else {
		r.row("FAIL", "config.bog", "not found")
		r.failed = true
	}
	for _, name := range consoles(r.station) {
		r.snapFile(name)
	}
	r.historyPointers()
	if err := writeFile(r.out+"/manifest.json", manifest(basename(r.station), stamp, r.files, r.pointers)); err != nil {
		r.shellErr(lineManifest, r.out+"/manifest.json: "+errText(err))
	}
	fmt.Fprintf(r.s.Out, "station-snapshot: %d file(s) copied; manifest.json written to %s\n", len(r.files), r.out)
	if r.failed {
		return ExitFailed
	}
	return ExitOK
}

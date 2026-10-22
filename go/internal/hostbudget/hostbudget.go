// Package hostbudget reports and gates the shared-host resources that took the WSL host down on
// 2026-10-22 (quality round 8 T41): the init.scope cgroup reached pids.max 48,129 and every fork
// failed, ~/.cache/go-build grew to 84 GB and C: reached 99%. Check reads cgroup v2 pids counters,
// the system thread budget, the Go build cache size and the free space of the given mounts, and
// returns one PASS|WARN|FAIL row per resource. [ev: retro r8-t41-host-safety Δ1] [ev: retro r8-t41-host-safety Δ3]
package hostbudget

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Status is the verdict for one resource.
type Status int

// Verdicts, ordered by severity.
const (
	Pass Status = iota
	Warn
	Fail
)

func (s Status) String() string {
	switch s {
	case Pass:
		return "PASS"
	case Warn:
		return "WARN"
	default:
		return "FAIL"
	}
}

// Row is one output line: `<STATUS> <resource> <used>/<limit>[ <note>]`.
type Row struct {
	Status                      Status
	Resource, Used, Limit, Note string
}

func (r Row) String() string {
	s := fmt.Sprintf("%s %s %s/%s", r.Status, r.Resource, r.Used, r.Limit)
	if r.Note != "" {
		s += " " + r.Note
	}
	return s
}

// Disk is the space of one filesystem in bytes, as df counts it: Used = blocks - free,
// Avail = blocks available to an unprivileged user (so Used+Avail excludes the root reserve).
type Disk struct{ Used, Avail uint64 }

// Fixed WARN thresholds of the spec (ledger.d/2026-10-22-r8-t41-host-safety.md); the FAIL
// thresholds are flags.
const (
	WarnPidsPct = 40
	WarnDiskPct = 80
)

const bytesPerGB = 1_000_000_000 // decimal GB: a 21 GB cache is above a 20 GB limit in either unit

// Config selects what Check reads and the thresholds it applies.
type Config struct {
	CgroupRoot string // cgroup v2 root; every direct child with pids.{current,max} is checked
	ProcRoot   string // proc root for sys/kernel/threads-max and loadavg
	PidsPct    int    // FAIL a pids counter above this percent of its limit (WARN above WarnPidsPct)
	DiskPct    int    // FAIL a mount above this percent used (WARN above WarnDiskPct)
	GocacheGB  int    // WARN the Go build cache above this many GB
	// GocacheDir is the Go build cache; "" means it could not be resolved (WARN), "off" disables it.
	GocacheDir string
	// GocacheMaxEntries bounds the cache walk; a cache with more entries is reported as a WARN.
	GocacheMaxEntries int
	Mounts            []string // checked after "/", which is always checked
}

// Probe holds the host calls that tests replace.
type Probe struct {
	Statfs func(path string) (Disk, error)
}

// Check returns one row per resource. Its only error is an unreadable CgroupRoot: without it the
// fork budget, the resource that aborted every session, cannot be judged at all.
func Check(c Config, p Probe) ([]Row, error) {
	rows, err := pidsRows(c)
	if err != nil {
		return nil, err
	}
	rows = append(rows, threadsRow(c))
	rows = append(rows, gocacheRow(c))
	return append(rows, diskRows(c, p)...), nil
}

// classify applies `FAIL above failPct, WARN above warnPct` without division: used*100 > limit*pct.
func classify(used, limit uint64, failPct, warnPct int) Status {
	switch {
	case used*100 > limit*uint64(failPct):
		return Fail
	case used*100 > limit*uint64(warnPct):
		return Warn
	}
	return Pass
}

func pct(used, limit uint64) uint64 {
	if limit == 0 {
		return 100
	}
	return used * 100 / limit
}

func readTrim(path string) (string, error) {
	b, err := os.ReadFile(path)
	return strings.TrimSpace(string(b)), err
}

// pidsRows checks every direct child cgroup of the root. A child without pids files (a mount unit
// with no pids controller) is skipped, as is pids.max = "max" (unlimited); a file that exists but
// cannot be read or parsed is a WARN, never a silent pass.
func pidsRows(c Config) ([]Row, error) {
	entries, err := os.ReadDir(c.CgroupRoot)
	if err != nil {
		return nil, fmt.Errorf("cannot read cgroup root %s: %w", c.CgroupRoot, err)
	}
	var rows []Row
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		res := "pids:" + e.Name()
		dir := filepath.Join(c.CgroupRoot, e.Name())
		cur, errCur := readTrim(filepath.Join(dir, "pids.current"))
		max, errMax := readTrim(filepath.Join(dir, "pids.max"))
		if errors.Is(errCur, fs.ErrNotExist) || errors.Is(errMax, fs.ErrNotExist) {
			continue
		}
		if errCur != nil || errMax != nil {
			rows = append(rows, Row{Warn, res, "?", "?", "(unreadable: " + errors.Join(errCur, errMax).Error() + ")"})
			continue
		}
		if max == "max" {
			continue
		}
		n, errN := strconv.ParseUint(cur, 10, 64)
		m, errM := strconv.ParseUint(max, 10, 64)
		if errN != nil || errM != nil {
			rows = append(rows, Row{Warn, res, cur, max, "(unparsable pids counters)"})
			continue
		}
		st := classify(n, m, c.PidsPct, WarnPidsPct)
		rows = append(rows, Row{st, res, cur, max, fmt.Sprintf("(%d%% used, fail above %d%%)", pct(n, m), c.PidsPct)})
	}
	return rows, nil
}

// threadsRow compares the system-wide task count (the total in loadavg's 4th field, running/total)
// with kernel.threads-max, under the same thresholds as a pids cgroup.
func threadsRow(c Config) Row {
	maxS, err := readTrim(filepath.Join(c.ProcRoot, "sys/kernel/threads-max"))
	if err != nil {
		return Row{Warn, "threads", "?", "?", "(unreadable threads-max: " + err.Error() + ")"}
	}
	load, err := readTrim(filepath.Join(c.ProcRoot, "loadavg"))
	if err != nil {
		return Row{Warn, "threads", "?", maxS, "(unreadable loadavg: " + err.Error() + ")"}
	}
	total := ""
	if f := strings.Fields(load); len(f) >= 4 {
		if _, after, ok := strings.Cut(f[3], "/"); ok {
			total = after
		}
	}
	n, errN := strconv.ParseUint(total, 10, 64)
	m, errM := strconv.ParseUint(maxS, 10, 64)
	if errN != nil || errM != nil {
		return Row{Warn, "threads", "?", maxS, "(unparsable loadavg or threads-max)"}
	}
	st := classify(n, m, c.PidsPct, WarnPidsPct)
	return Row{st, "threads", total, maxS, fmt.Sprintf("(%d%% used, fail above %d%%)", pct(n, m), c.PidsPct)}
}

func gbString(b uint64) string { return fmt.Sprintf("%.1fGB", float64(b)/bytesPerGB) }

const cleanHint = "hint: go clean -cache"

// gocacheRow sums the apparent size of every regular file under the cache, walking at most
// GocacheMaxEntries entries. Apparent size, not allocated blocks: it is what the cache holds and
// what `go clean -cache` frees, and it keeps a sparse test fixture honest.
func gocacheRow(c Config) Row {
	limit := uint64(c.GocacheGB) * bytesPerGB
	switch c.GocacheDir {
	case "":
		return Row{Warn, "gocache", "?", gbString(limit), "(GOCACHE could not be resolved)"}
	case "off":
		return Row{Pass, "gocache", "off", gbString(limit), ""}
	}
	var size uint64
	entries, stopped, walkErr := 0, false, error(nil)
	err := filepath.WalkDir(c.GocacheDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if path == c.GocacheDir && errors.Is(err, fs.ErrNotExist) {
				return fs.SkipAll // a cache that was never built (or was just cleaned) is empty
			}
			walkErr = err
			if d != nil && d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if entries++; entries > c.GocacheMaxEntries {
			stopped = true
			return fs.SkipAll
		}
		if d.Type().IsRegular() {
			info, err := d.Info()
			if err != nil {
				walkErr = err
				return nil
			}
			size += uint64(info.Size())
		}
		return nil
	})
	if err != nil {
		walkErr = err
	}
	res := "gocache"
	used := gbString(size)
	switch {
	case stopped:
		return Row{Warn, res, ">=" + used, gbString(limit),
			fmt.Sprintf("(walk stopped after %d entries; %s) %s", c.GocacheMaxEntries, c.GocacheDir, cleanHint)}
	case walkErr != nil:
		return Row{Warn, res, ">=" + used, gbString(limit), fmt.Sprintf("(walk incomplete: %v)", walkErr)}
	case size > limit:
		return Row{Warn, res, used, gbString(limit), fmt.Sprintf("(%s) %s", c.GocacheDir, cleanHint)}
	}
	return Row{Pass, res, used, gbString(limit), "(" + c.GocacheDir + ")"}
}

// diskRows checks "/" and every mount once, in order.
func diskRows(c Config, p Probe) []Row {
	seen := map[string]bool{}
	var rows []Row
	for _, m := range append([]string{"/"}, c.Mounts...) {
		if seen[m] {
			continue
		}
		seen[m] = true
		res := "disk:" + m
		d, err := p.Statfs(m)
		if err != nil {
			rows = append(rows, Row{Warn, res, "?", "?", "(statfs: " + err.Error() + ")"})
			continue
		}
		size := d.Used + d.Avail
		st := classify(d.Used, size, c.DiskPct, WarnDiskPct)
		rows = append(rows, Row{st, res, gbString(d.Used), gbString(size),
			fmt.Sprintf("(%d%% used, fail above %d%%)", pct(d.Used, size), c.DiskPct)})
	}
	return rows
}

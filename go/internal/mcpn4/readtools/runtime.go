package readtools

import (
	"fmt"
	"math"
	"path/filepath"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// StationRuntime is n4_station_runtime (see runtime_data.go): the uptime, loaded_modules and
// jars_on_disk sections, the differences between the last two, and restart_needed.
func (t *Tools) StationRuntime(args *pyjson.Object) (*pyjson.Object, error) {
	maxRows, timeout := intArg(args, "max_rows", runtimeDefaultRows), intArg(args, "timeout_s", 60)
	start := t.Ctx.Clock()
	uptime, upSecs, upKnown, err := t.runtimeUptime(timeout)
	if err != nil {
		return nil, err
	}
	// the boot on this machine's clock: the jar times it is compared with are this machine's too
	boot := time.Now().Add(-time.Duration(upSecs * float64(time.Second)))
	loadedSection, parts, partsKnown, err := t.runtimeLoaded(timeout, maxRows)
	if err != nil {
		return nil, err
	}
	station := t.Ctx.Session.StationName
	out := obj("station", station, "uptime", uptime, "loaded_modules", loadedSection)
	var why []string
	if !upKnown {
		why = append(why, runtimeUnknownUptime)
	}
	if !partsKnown {
		why = append(why, runtimeUnknownLoaded)
	}
	disk, scan, diskKnown := t.runtimeDisk(station, parts, boot, upKnown, timeout)
	out.Set("jars_on_disk", disk)
	var mismatches []*pyjson.Object
	switch {
	case !diskKnown:
		why = append(why, runtimeUnknownDisk)
		out.Set("mismatches", nil)
		out.Set("mismatch_count", nil)
	default:
		if len(scan.unreadable) > 0 {
			why = append(why, fmt.Sprintf(runtimeUnknownJars, len(scan.unreadable)))
		}
		if scan.cut {
			why = append(why, fmt.Sprintf(runtimeUnknownScanCut, runtimeMaxJars))
		}
		mismatches = compareRuntime(parts, scan, boot, upKnown)
		listed := make([]any, 0, min(len(mismatches), maxRows))
		for _, m := range mismatches[:min(len(mismatches), maxRows)] {
			listed = append(listed, m)
		}
		out.Set("mismatches", listed)
		out.Set("mismatch_count", len(mismatches))
		out.Set("mismatches_truncated", len(mismatches) > maxRows)
	}
	verdict, note := restartVerdict(len(mismatches), why)
	out.Set("restart_needed", verdict)
	out.Set("restart_note", note)
	out.Set("elapsed_ms", int(math.RoundToEven((t.Ctx.Clock()-start)*1000)))
	t.recordRead("n4_station_runtime", "", args, len(parts))
	return out, nil
}

// runtimeDisk is the jars_on_disk section: unavailable without --niagara-home for the station or
// when its modules directory cannot be listed; known is true when the scan ran.
func (t *Tools) runtimeDisk(station string, parts []loadedPart, boot time.Time, bootKnown bool, timeout int) (*pyjson.Object, diskScan, bool) {
	home, ok := t.NiagaraHomes[station]
	if !ok {
		return obj("status", runtimeUnavailable, "note", fmt.Sprintf(runtimeNoHome, station)), diskScan{}, false
	}
	dir := filepath.Join(home, runtimeModulesDir)
	loaded := map[string]bool{}
	for _, p := range parts {
		loaded[p.name] = true
	}
	scan, err := scanJars(dir, loaded, boot, bootKnown, time.Now().Add(time.Duration(timeout)*time.Second))
	if err != nil {
		return obj("status", runtimeUnavailable, "note", fmt.Sprintf(runtimeNoDir, station, t.Ctx.Scrub(err.Error()))), diskScan{}, false
	}
	read := 0
	for _, j := range scan.jars {
		if j.read {
			read++
		}
	}
	out := obj("status", runtimeStatusOK, "modules_dir", dir, "count", len(scan.jars), "versions_read", read,
		"complete", !scan.cut && len(scan.unreadable) == 0)
	if len(scan.unreadable) > 0 {
		out.Set("unreadable", strs(scan.unreadable))
	}
	if len(scan.skipped) > 0 {
		out.Set("skipped", strs(scan.skipped))
	}
	return out, scan, true
}

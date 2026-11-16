package reportmodule

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"regexp"
	"strings"
)

// crashRow is the detail of the fail-closed row of a member that exited non-zero without a FAIL row
// (exitedNoFail, lint-link-target-flags): a crash is news in every run, so it is never baselined.
var crashRow = regexp.MustCompile(`^exited [0-9]+ with no FAIL row`)

// baseline is the FAIL baseline of one run: the keys <module-root>/.report-baseline lists and the FAIL rows of
// the run sorted against them. Without the file (--write-baseline alone) it only records the run's keys.
type baseline struct {
	path    string
	grouped bool            // the file was read: FAIL rows are held back and printed as the NEW and BASE groups
	listed  map[string]bool // key in the file -> a FAIL row of this run matched it
	order   []string        // the file's keys in file order, each once
	fresh   []failRow       // NEW: not listed, or a floor check row
	known   []failRow       // BASE: listed
	refused []string        // WARN details of the listed keys a never-baselined row matched, in row order
	keys    []string        // the key of every FAIL row that may be baselined, in row order
	floorN  int             // floor check FAIL rows (never baselined)
	crashN  int             // FAIL rows of a member that exited non-zero with no FAIL row (never baselined)
}

// loadBaseline reads <root>/.report-baseline. It returns nil when the run has no baseline to apply and none to
// write (no file or --no-baseline, and no --write-baseline), and an error for a file that cannot be read.
func loadBaseline(o options) (*baseline, error) {
	b := &baseline{path: strings.TrimRight(o.root, "/") + "/" + baselineName}
	if !o.noBaseline {
		data, err := os.ReadFile(b.path)
		switch {
		case err == nil:
			b.read(string(data))
		case errors.Is(err, fs.ErrNotExist):
		default:
			// A path that cannot even be looked up (a root that is not searchable) is a run without a baseline:
			// every FAIL row counts, as before the file existed. A file that is there and unreadable is refused.
			if _, statErr := os.Lstat(b.path); statErr == nil {
				return nil, err
			}
		}
	}
	if !b.grouped && !o.writeBaseline {
		return nil, nil
	}
	return b, nil
}

// read takes the keys of a baseline file: one per line, trailing blanks dropped, blank and "#" lines ignored.
func (b *baseline) read(data string) {
	b.grouped, b.listed = true, map[string]bool{}
	for _, ln := range strings.Split(data, "\n") {
		key := strings.TrimRight(ln, " \t\r")
		if key == "" || key[0] == '#' {
			continue
		}
		if _, dup := b.listed[key]; !dup {
			b.listed[key] = false
			b.order = append(b.order, key)
		}
	}
}

// hold takes a FAIL row and reports whether it is held back from the row stream (a run with a baseline file
// prints it later, in its group). A floor check row and a crashed member's row are NEW whatever the file lists.
func (b *baseline) hold(w failRow, floor bool) bool {
	if b == nil {
		return false
	}
	key := rowKey(w)
	_, listed := b.listed[key]
	if listed {
		b.listed[key] = true
	}
	refused := ""
	switch {
	case floor:
		b.floorN++
		refused = msgFloorListed
	case crashRow.MatchString(w.detail):
		b.crashN++
		refused = msgCrashListed
	}
	switch {
	case refused != "":
		if listed {
			b.refused = append(b.refused, refused+key)
		}
		b.fresh = append(b.fresh, w)
	case listed:
		b.keys = append(b.keys, key)
		b.known = append(b.known, w)
	default:
		b.keys = append(b.keys, key)
		b.fresh = append(b.fresh, w)
	}
	return b.grouped
}

// failGroups prints the held FAIL rows as the NEW and BASE groups, counting the NEW ones as the FAIL rows of
// the run (the verdict and the exit code read them alone), then one WARN row per stale baseline line and per
// listed row that is never baselined (a floor check row, a crashed member's row). A run without a baseline file prints nothing here.
func (r *report) failGroups() {
	b := r.bl
	if b == nil || !b.grouped {
		return
	}
	fmt.Fprintf(r.out, msgNewRows, b.path)
	r.group(b.fresh)
	for range b.fresh {
		r.count(failed)
	}
	fmt.Fprintf(r.out, msgBaseRows, b.path)
	r.group(b.known)
	for _, key := range b.order {
		if !b.listed[key] {
			r.emit("(module)", warn, baselineCheck, msgStale+key)
		}
	}
	for _, detail := range b.refused {
		r.emit("(module)", warn, baselineCheck, detail)
	}
}

// group prints the rows of one group as FAIL rows, or "(none)".
func (r *report) group(rows []failRow) {
	if len(rows) == 0 {
		fmt.Fprint(r.out, msgNoRows)
	}
	for _, w := range rows {
		fmt.Fprintf(r.out, "%s  %s  %s  %s\n", w.label, failed, w.check, w.detail)
	}
}

// failSplit is what follows the FAIL count of the summary in a run with a baseline file: the NEW and BASE
// counts and the file, so "0 new" never reads as "0 FAIL".
func (r *report) failSplit() string {
	if r.bl == nil || !r.bl.grouped {
		return ""
	}
	return fmt.Sprintf(msgFailSplit, r.n[failed], len(r.bl.known), r.bl.path)
}

// baseFails is the number of BASE rows: FAIL rows of the run that the verdict does not read.
func (r *report) baseFails() int {
	if r.bl == nil {
		return 0
	}
	return len(r.bl.known)
}

package commissioningverify

import (
	"fmt"
	"io"
)

// Row statuses.
const (
	pass   = "PASS"
	failed = "FAIL"
	warn   = "WARN"
	skip   = "SKIP"
	manual = "MANUAL"
)

// report prints the rows and counts them per status.
type report struct {
	out io.Writer
	n   map[string]int
}

func (r *report) emit(status, check, detail string) {
	fmt.Fprintf(r.out, "%s  commissioning  %s  %s\n", status, check, detail)
	r.n[status]++
}

// valuesOwed prints the values-owed rows of the table at path (text read by Main): one MANUAL row per open value,
// one PASS row when none is open, one MANUAL row when the module declares no table.
func (r *report) valuesOwed(path, text string) {
	if path == "" {
		r.emit(manual, "values-owed", "no docs/values-owed.md — declare the values owed by the field (slot, owed by, unit, safe default, status) or state none (BUILD-LOOP §6.b)")
		return
	}
	rows := owedRows(path, text)
	for _, d := range rows {
		r.emit(manual, "values-owed", d)
	}
	if len(rows) == 0 {
		r.emit(pass, "values-owed", "every value owed by the field is provided ("+path+")")
	}
}

// footer is the manual-only footer: live steps this tool cannot statically verify.
var footer = [][2]string{
	{"hot-reload-console", "after deploy: run triage-console.sh — confirm no own-module load failures before hand-off"},
	{"plant-control", "confirm the station actually controls the plant (runtime only — not statically verifiable)"},
	{"per-instance-values", "confirm per-instance runtime values match physical setpoints for every room/unit"},
	{"persisted-state-restart", "persisted operator state (HOA modes, setpoints, run hours, backups) seeded or saved BEFORE the first restart and re-read after it (BUILD-LOOP §6.b)"},
	{"alarm-routing", "each alarm source has an alarm class and at least one recipient; a test alarm reaches the console and is acknowledged (BUILD-LOOP §6.b)"},
	{"consumer-impact", "every point whose meaning changed (feature doc Consumer impact table) was notified to its downstream consumers (BUILD-LOOP §6.b)"},
	{"link-source-audit", "run obix-link-audit.sh --map docs/wiring-map.md against the live facade: every declared link-in comes from its declared source (MATCH)"},
	{"servlet-response-headers", "BWebServlet module: curl -sI -H 'X-Requested-With: XMLHttpRequest' -u admin:pass http://<station>/<module>/api/equipment | grep -iE 'x-content-type-options|x-frame-options|content-security-policy' — the station's header providers must reach the servlet (see types/security.md §7)"},
}

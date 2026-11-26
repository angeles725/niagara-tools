package dashboardsave

import (
	"regexp"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/lint"
)

// Check codes of the two per-path findings (the check column of a WARN row).
const (
	checkPath = "write-path-no-save"
	checkStop = "stop-drops-save"
)

const msgPath = "handler writes station state with the request Context but no path from it requests a station" +
	" save (requestSave() directly or through a same-module helper): that write is lost on the next restart;" +
	" call requestSave() after the write succeeds, or declare persistence: <where> in MODULE-MAP.md" +
	" (types/dashboard.md § Write path)"

const msgStop = "stopped() cancels the pending save ticket without a flush: a write made inside the settle window" +
	" before a restart or a servlet stop is never saved; read !saveTicket.isExpired() before cancel() and call" +
	" station.save() in a try/catch that logs (types/dashboard.md § Write path)"

var (
	// ticketCancel cancels a save ticket: <name>Ticket.cancel(.
	ticketCancel = regexp.MustCompile(`[A-Za-z0-9_]*[Tt]icket[A-Za-z0-9_]*\s*\.\s*cancel\s*\(`)
	// flushCall is a station save made now: <x>.save(, <x>.saveAsync(.
	flushCall = regexp.MustCompile(`\.\s*save[A-Za-z]*\s*\(`)
)

// source is one part file of a module: its path, code and whether it is a -ux file.
type source struct {
	path    string
	methods []method
	writes  []int // the write lines of a -ux file
}

// pathFindings are the module's per-path WARNs: a handler with no path to a save (only in a module that saves;
// without one every write line is already a FAIL), then each stopped() that cancels the save ticket unflushed.
func pathFindings(srcs []source, saves bool) []lint.Finding {
	var all []method
	for _, s := range srcs {
		all = append(all, s.methods...)
	}
	savers := closure(all, savesStation)
	flushers := closure(all, flushCall.MatchString)
	var out []lint.Finding
	if saves {
		for _, s := range srcs {
			out = append(out, unsavedHandlers(s, savers)...)
		}
	}
	for _, s := range srcs {
		for _, m := range s.methods {
			if m.name != "stopped" || !ticketCancel.MatchString(m.body) || flushes(m.body, flushers) {
				continue
			}
			at := ticketCancel.FindStringIndex(m.body)[0]
			out = append(out, lint.Finding{Check: checkStop, Severity: lint.Warn, Path: s.path,
				Line: m.first + strings.Count(m.body[:at], "\n"), Msg: msgStop})
		}
	}
	return out
}

// closure is the names of the methods whose body has direct (matched by is) or calls a method of the set.
func closure(all []method, is func(string) bool) map[string]bool {
	set := map[string]bool{}
	for changed := true; changed; {
		changed = false
		for _, m := range all {
			if !set[m.name] && (is(m.body) || callsAny(m.body, set)) {
				set[m.name], changed = true, true
			}
		}
	}
	return set
}

// callsAny reports a call by name to a method of the set.
func callsAny(body string, set map[string]bool) bool {
	for name := range set {
		if calls(body, name) {
			return true
		}
	}
	return false
}

func calls(body, name string) bool {
	return regexp.MustCompile(`(^|[^A-Za-z0-9_])` + regexp.QuoteMeta(name) + `\s*\(`).MatchString(body)
}

func flushes(body string, flushers map[string]bool) bool {
	return flushCall.MatchString(body) || callsAny(body, flushers)
}

// unsavedHandlers are the methods of a -ux file that write and reach no save, one WARN each on the first write
// line. Only a file where some method does reach a save is judged (it is the servlet owning the save; a helper
// file that never saves is left to the module-wide rule), and a write helper that a saving method of the file
// calls is covered by that caller.
func unsavedHandlers(s source, savers map[string]bool) []lint.Finding {
	reach := func(m method) bool { return savesStation(m.body) || callsAny(m.body, savers) }
	judged := false
	for _, m := range s.methods {
		judged = judged || reach(m)
	}
	var out []lint.Finding
	for _, m := range s.methods {
		line := firstWrite(s.writes, m)
		if !judged || line == 0 || reach(m) || calledBySaver(s.methods, m, reach) {
			continue
		}
		out = append(out, lint.Finding{Check: checkPath, Severity: lint.Warn, Path: s.path, Line: line, Msg: msgPath})
	}
	return out
}

// firstWrite is the first write line inside m (0 when m writes nothing).
func firstWrite(writes []int, m method) int {
	for _, l := range writes {
		if l >= m.first && l <= m.end {
			return l
		}
	}
	return 0
}

func calledBySaver(all []method, m method, reach func(method) bool) bool {
	for _, o := range all {
		if o.name != m.name && calls(o.body, m.name) && reach(o) {
			return true
		}
	}
	return false
}

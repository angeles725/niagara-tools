package reportmodule

import "strings"

// How a lint member's parsed rows (rows.go) are relayed into the report and closed with a verdict row.

// relay is what one member_rows call relayed: rows in all, a FAIL row (exact), a row whose status starts with
// FAIL, and the WARN and ADVISORY row counts.
type relay struct {
	rows        int
	fail, failP bool
	warn, adv   int
}

// memberRows relays the selected rows of a member's output as `<label>  <ST>  <check>  <detail>` rows.
func (r *report) memberRows(label, out string, f filter) relay {
	var rl relay
	for _, w := range parseRows(out, f) {
		r.emit(label, w.st, w.check, w.detail)
		rl.rows++
		switch {
		case w.st == "FAIL":
			rl.fail, rl.failP = true, true
		case strings.HasPrefix(w.st, "FAIL"):
			rl.failP = true
		case w.st == "WARN":
			rl.warn++
		case w.st == "ADVISORY":
			rl.adv++
		}
	}
	return rl
}

// relayRows relays a member's `FAIL|WARN|ADVISORY  <check>  <path>:<line>  <reason>` rows (site mode).
func (r *report) relayRows(label, out string) relay {
	return r.memberRows(label, out, filter{sel: "FAIL|WARN|ADVISORY"})
}

// verdictAfterRelay is the member verdict row after relayRows: no row at all when a FAIL row was relayed (the
// FAIL rows are the verdict), PASS "clean" when no WARN or ADVISORY row was either, else PASS "no failing row
// (N WARN[, M ADVISORY] row(s) above)".
func (r *report) verdictAfterRelay(label, check string, rl relay) {
	n := rl.warn + rl.adv
	switch {
	case rl.fail:
	case n == 0:
		r.emit(label, pass, check, "clean")
	default:
		what := itoa(rl.warn) + " WARN"
		if rl.adv > 0 {
			what += ", " + itoa(rl.adv) + " ADVISORY"
		}
		if n == 1 {
			what += " row above"
		} else {
			what += " rows above"
		}
		r.emit(label, pass, check, "no failing row ("+what+")")
	}
}

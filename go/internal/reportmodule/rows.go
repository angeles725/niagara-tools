package reportmodule

import "strings"

// mode is how the shared row parser (the bash ROW_AWK) reads a member's row fields.
type mode int

const (
	site   mode = iota // `ST  check  path:line  reason` -> detail `<basename>:<line>  <reason>`
	swap               // as site, but the member prints `check  ST  ...` (lint-wb-threading)
	base               // `ST  check  path  reason` -> detail `<basename>  <reason>` (no :line split)
	nosite             // `ST  check  site  reason` -> detail `<reason>`
	ws                 // whitespace-split `ST check path detail...` -> detail = the words from field 4, one space apart
)

// filter selects a member's rows: sel and asw are "|"-separated line prefixes (the bash EREs ^(sel) and
// ^(asw)), sel "" takes every non-empty line, a line matching asw is relayed as WARN whatever sel says, and keep
// (non-empty) keeps only the rows whose status, up to a "|", is exactly keep.
type filter struct {
	sel  string
	mode mode
	keep string
	asw  string
}

// row is one relayed member row.
type row struct{ st, check, detail string }

// parseRows is ROW_AWK and member_rows' record split over a member's whole output: one row per selected line,
// in order. Fields are split on runs of 2+ whitespace (one space or tab never splits); the
// `<st>|<check>|<detail>` record is split back at its first two "|", so a "|" inside the status or check
// shifts the fields exactly as the shell's ${p%%|*} / ${p#*|} split did.
func parseRows(out string, f filter) []row {
	var rows []row
	for _, ln := range strings.Split(out, "\n") {
		forced := f.asw != "" && hasPrefixIn(ln, f.asw)
		if !forced && (f.sel == "" && ln == "" || f.sel != "" && !hasPrefixIn(ln, f.sel)) {
			continue
		}
		var st, chk, det string
		if f.mode == ws { // keep and asw never apply here (ROW_AWK printed the ws record first)
			w := awkFields(ln)
			st, chk = field(w, 0), field(w, 1)
			if len(w) > 3 {
				det = strings.Join(w[3:], " ")
			}
		} else {
			a := splitRuns(ln)
			st, chk = field(a, 0), field(a, 1)
			if f.mode == swap {
				st, chk = chk, st
			}
			at := field(a, 2)
			reason := ""
			for i := 3; i < len(a); i++ {
				if reason == "" {
					reason = a[i]
				} else {
					reason += "  " + a[i]
				}
			}
			switch f.mode {
			case nosite:
				det = reason
			case base:
				det = lastPart(at) + "  " + reason
			default:
				path, line, _ := strings.Cut(at, ":")
				det = lastPart(path) + ":" + line + "  " + reason
			}
			if f.keep != "" && cutBefore(st, "|") != f.keep {
				continue
			}
			if forced {
				st = "WARN"
			}
		}
		rec := st + "|" + chk + "|" + det
		s, rest, _ := strings.Cut(rec, "|")
		c, d, _ := strings.Cut(rest, "|")
		rows = append(rows, row{s, c, d})
	}
	return rows
}

// hasPrefixIn reports whether s starts with one of the "|"-separated prefixes.
func hasPrefixIn(s, prefixes string) bool {
	for _, p := range strings.Split(prefixes, "|") {
		if strings.HasPrefix(s, p) {
			return true
		}
	}
	return false
}

func field(a []string, i int) string {
	if i < len(a) {
		return a[i]
	}
	return ""
}

// awkFields is awk's default field split: runs of spaces and tabs separate, leading and trailing ones ignored.
func awkFields(s string) []string {
	return strings.FieldsFunc(s, func(r rune) bool { return r == ' ' || r == '\t' })
}

// splitRuns is gawk's split(s, a, /[[:space:]]{2,}/) in the C locale: a run of two or more ASCII whitespace
// bytes separates, so a leading run gives an empty first field and a trailing run an empty last one; "" has
// no field.
func splitRuns(s string) []string {
	if s == "" {
		return nil
	}
	var out []string
	start := 0
	for i := 0; i < len(s); {
		if !isSpace(s[i]) {
			i++
			continue
		}
		j := i
		for j < len(s) && isSpace(s[j]) {
			j++
		}
		if j-i >= 2 {
			out = append(out, s[start:i])
			start = j
		}
		i = j
	}
	return append(out, s[start:])
}

func isSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\v' || c == '\f' || c == '\r'
}

// lastPart is ROW_AWK's bname: the text after the last "/" ("" for "" and for a path ending in "/").
func lastPart(p string) string { return p[strings.LastIndexByte(p, '/')+1:] }

// cutBefore is the shell's ${s%%<sep>*}: s up to the first sep (all of s when there is none).
func cutBefore(s, sep string) string {
	if i := strings.Index(s, sep); i >= 0 {
		return s[:i]
	}
	return s
}

// cutAfter is the shell's ${s#*<sep>}: s after the first sep (all of s when there is none).
func cutAfter(s, sep string) string {
	if i := strings.Index(s, sep); i >= 0 {
		return s[i+len(sep):]
	}
	return s
}

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

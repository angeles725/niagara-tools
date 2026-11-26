package reportmodule

// vendored is lint-vendored on src/rc when src/rc/vendor/ is for the lint to judge (toJudge), next to vendor-floor:
// a vendored file at a version the kit's known-defects table lists is a WARN row (#1145; the lint is #1123, and
// reads the table by its own default path). An rc dir without vendor/ gets no row at all, as the lint is clean
// there too, and a clean vendor/ gets none either (the rows are the WARNs). Exit 3 (an unreadable table or
// directory) is an env fault row and any other non-zero exit a FAIL row, never a silent pass (lintRows).
func (r *report) vendored(a artifact) {
	const check = "lint-vendored"
	rc := a.dir + "/src/rc"
	if !toJudge(rc + "/vendor") {
		return
	}
	out, code, ok := r.x.member(check, rc)
	if r.truncated(a.name, check, check+".sh", ok) {
		return
	}
	r.lintRows(a.name, check, code, out, filter{sel: "WARN", mode: base}, none)
}

package ledgercheck

import (
	"regexp"
	"strings"
)

// niagaraCol is the optional Versions column naming the version of the niagara_home a build used (the 2026-11-25
// ledger-niagara change, #1026): a ledger without it is valid, and so is a row that leaves the cell empty or —.
const niagaraCol = "Niagara"

const niagaraForm = " is not X.Y.Z[.W] (the niagara_home version the build used; — when unknown)"

var niagaraVersion = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+(\.[0-9]+)?$`)

// niagaraOK reports a Niagara cell that is empty, —, the sample row's placeholder (LC9 already warned) or a version.
func niagaraOK(v, sample string) bool {
	return v == "" || v == none || niagaraVersion.MatchString(v) || (v == sample && placeholder.MatchString(v))
}

// withoutNiagara is the header and the sample row minus their Niagara cell: what a ledger without the column holds
// (nil, nil when the template has no such column).
func withoutNiagara(header, sample []string) (h, smp []string) {
	for i, x := range header {
		if x == niagaraCol {
			drop := func(r []string) []string {
				if i >= len(r) {
					return nil
				}
				return append(append([]string{}, r[:i]...), r[i+1:]...)
			}
			return drop(header), drop(sample)
		}
	}
	return nil, nil
}

// Latest is the newest Versions row of a ledger: its module, its module version and the Niagara version it
// records ("" when the cell is empty, —, or not a version).
type Latest struct{ Module, Version, Niagara string }

// LatestRow is the last data row of the Versions table (newest last) of a ledger with a Niagara column; false when
// the ledger has no such column or no row. The template's sample row is not a row. preflight's ledger-niagara
// reads it.
func LatestRow(text string) (Latest, bool) {
	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	at := tableAfter(lines, "## Versions")
	if at < 0 {
		return Latest{}, false
	}
	hdr := cells(lines[at])
	col := map[string]int{}
	for i, h := range hdr {
		col[h] = i
	}
	ni, ok := col[niagaraCol]
	if !ok || len(hdr) < 3 {
		return Latest{}, false
	}
	var last Latest
	found := false
	for _, i := range versionRows(lines) {
		r := cells(lines[i])
		if i == at || separator(r) || sameCells(r, hdr) || len(r) != len(hdr) || placeholder.MatchString(r[col["Module"]]) {
			continue
		}
		last, found = Latest{r[col["Module"]], r[col["Version"]], ""}, true
		if niagaraVersion.MatchString(r[ni]) {
			last.Niagara = r[ni]
		}
	}
	return last, found
}

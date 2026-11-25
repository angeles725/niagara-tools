package ledgercheck

import (
	"strings"
	"testing"
)

// headerN is the template's Versions header: the legacy one with the optional Niagara column after Version.
const headerN = "| Module | Version | Niagara | Date | Status | What changed | Defect found | Root cause | Class | Resolved by | Prevention (retro + Δ) |"

// ledgerN is ledger() with the Niagara column in its header.
func ledgerN(rows ...string) string {
	return strings.Replace(ledger(rows...), header+"\n|---|---|---|---|---|---|---|---|---|---|\n",
		headerN+"\n|---|---|---|---|---|---|---|---|---|---|---|\n", 1)
}

func rowN(niagara string) string {
	return "| Pan | 1.2.0 | " + niagara + " | 2026-10-01 | deployed | first cut | — | — | — | — | — |"
}

const (
	notNiagara = "LC10: niagara version "
	wantForm   = " is not X.Y.Z[.W] (the niagara_home version the build used; — when unknown)\n"
)

// TestNiagaraColumn: the optional Niagara column is accepted, absent, empty or as a version; a malformed one FAILs.
func TestNiagaraColumn(t *testing.T) {
	cases := []struct {
		name, body string
		code       int
		out        string
	}{
		{"a version of four parts", ledgerN(rowN("4.14.0.162")), 0, ""},
		{"a version of three parts", ledgerN(rowN("4.15.6")), 0, ""},
		{"the unknown marker", ledgerN(rowN("—")), 0, ""},
		{"an empty cell", ledgerN(rowN("")), 0, ""},
		{"an empty table under the new header", ledgerN(), 0, ""},
		// Mutation: LC10-form -- accepting any text leaves a typo in the column that preflight can never compare
		{"a malformed version", ledgerN(rowN("4.14")), 1, "FAIL  ledger-check  L.md:7  " + notNiagara + "4.14" + wantForm},
		{"free text", ledgerN(rowN("latest")), 1, "FAIL  ledger-check  L.md:7  " + notNiagara + "latest" + wantForm},
		{"a trailing build tag", ledgerN(rowN("4.14.0.162-rc")), 1, "FAIL  ledger-check  L.md:7  " + notNiagara + "4.14.0.162-rc" + wantForm},
		// Mutation: LC10-header -- reading cells by the legacy position puts Niagara's text in the Date column
		{"a row under the new header is read by the new columns", ledgerN(rowN("4.14.0.162"),
			"| Pan | 1.2.1 | 4.14.0.162 | 2026-13-01 | installed | x | — | — | 9 | — | — |"), 1,
			"FAIL  ledger-check  L.md:8  LC5: status installed is not one of built, packaged, deployed, superseded, do-not-install\n" +
				"FAIL  ledger-check  L.md:8  LC6: date 2026-13-01 is not a YYYY-MM-DD date\n" +
				"FAIL  ledger-check  L.md:8  LC7: class 9 is not in the Root-cause classes table\n"},
		{"a legacy row under the new header is a cell-count FAIL", ledgerN(good), 1,
			"FAIL  ledger-check  L.md:7  LC4: row has 10 cells, want 11\n"},
		{"a new row under the legacy header is a cell-count FAIL", ledger(rowN("4.14.0.162")), 1,
			"FAIL  ledger-check  L.md:7  LC4: row has 11 cells, want 10\n"},
		{"the sample row of the new template is the placeholder WARN", ledgerN("| <Module> | <x.y.z> | <4.x.y.z> | <YYYY-MM-DD> | built | <one line> | — | — | — | — | — |"), 0,
			"WARN  ledger-check  L.md:7  LC9: template placeholder left in the row (replace or delete it)\n"},
		{"a left placeholder Niagara cell WARNs and is no LC10", ledgerN("| Pan | 1.2.0 | <4.x.y.z> | 2026-10-01 | built | x | — | — | — | — | — |"), 0,
			"WARN  ledger-check  L.md:7  LC9: template placeholder left in the row (replace or delete it)\n"},
		{"a repeated new header after a blank line is not data", ledgerN(rowN("4.14.0.162"), "", headerN, "|---|---|---|---|---|---|---|---|---|---|---|", rowN("4.15.6.20")), 0, ""},
		{"a header that is neither the template's nor its legacy form", strings.Replace(ledgerN(), "| Version | Niagara |", "| Niagara | Version |", 1), 1,
			"FAIL  ledger-check  L.md:5  LC3: the Versions table header differs from the template's (" + headerN + ")\n"},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			code, out, errs := run(t, tt.body)
			if code != tt.code || out != tt.out || errs != "" {
				t.Errorf("got (%d, %q, %q)\nwant (%d, %q)", code, out, errs, tt.code, tt.out)
			}
		})
	}
}

// TestLatest: the Niagara version of the newest Versions row, the one preflight compares with the install.
func TestLatest(t *testing.T) {
	cases := []struct {
		name, body string
		want       Latest
		ok         bool
	}{
		{"the last row wins", ledgerN(rowN("4.13.2.18"), rowN("4.15.6.20")), Latest{"Pan", "1.2.0", "4.15.6.20"}, true},
		{"CRLF line ends", strings.ReplaceAll(ledgerN(rowN("4.15.6.20")), "\n", "\r\n"), Latest{"Pan", "1.2.0", "4.15.6.20"}, true},
		{"an empty cell is no version", ledgerN(rowN("4.13.2.18"), rowN("")), Latest{"Pan", "1.2.0", ""}, true},
		{"the unknown marker is no version", ledgerN(rowN("—")), Latest{"Pan", "1.2.0", ""}, true},
		{"a malformed cell is no version", ledgerN(rowN("4.x")), Latest{"Pan", "1.2.0", ""}, true},
		{"rows after a blank line and a repeated header count", ledgerN(rowN("4.13.2.18"), "", headerN, "|---|---|---|---|---|---|---|---|---|---|---|", rowN("4.15.6.20")),
			Latest{"Pan", "1.2.0", "4.15.6.20"}, true},
		{"the sample row is no row", ledgerN(rowN("4.13.2.18"), "| <Module> | <x.y.z> | <4.x.y.z> | <YYYY-MM-DD> | built | <one line> | — | — | — | — | — |"),
			Latest{"Pan", "1.2.0", "4.13.2.18"}, true},
		{"a legacy ledger has no Niagara column", ledger(good), Latest{}, false},
		{"an empty table has no row", ledgerN(), Latest{}, false},
		{"text that is no ledger", "just notes\n", Latest{}, false},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if got, ok := LatestRow(tt.body); got != tt.want || ok != tt.ok {
				t.Errorf("LatestRow = (%+v, %v), want (%+v, %v)", got, ok, tt.want, tt.ok)
			}
		})
	}
}

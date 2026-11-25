package preflight

import (
	"archive/zip"
	"os"
	"strings"
	"testing"
)

const (
	ledgerHead = "# VERSION-LEDGER — client\n\n## Versions\n\n" +
		"| Module | Version | Niagara | Date | Status | What changed | Defect found | Root cause | Class | Resolved by | Prevention (retro + Δ) |\n" +
		"|---|---|---|---|---|---|---|---|---|---|---|\n"
	legacyHead = "# VERSION-LEDGER — client\n\n## Versions\n\n" +
		"| Module | Version | Date | Status | What changed | Defect found | Root cause | Class | Resolved by | Prevention (retro + Δ) |\n" +
		"|---|---|---|---|---|---|---|---|---|---|\n"
	legacyRow = "| Pan | 1.2.0 | 2026-10-01 | built | x | — | — | — | — | — |\n"
)

func row(niagara string) string {
	return "| Pan | 1.2.0 | " + niagara + " | 2026-10-01 | built | x | — | — | — | — | — |\n"
}

// niagaraRows are the ledger-niagara rows of a preflight run over the gradle root gr and niagara_home nh.
func niagaraRows(t *testing.T, d, nh, gr string) []string {
	t.Helper()
	code, out, errOut := pf((&fakeHost{}).host(), nil, "--jvm-dir", d+"/jvm", nh, gr)
	var rows []string
	for _, l := range strings.Split(out, "\n") {
		if strings.Contains(l, "ledger-niagara") {
			rows = append(rows, l)
		}
	}
	if errOut != "" || (code != ExitOK && code != ExitFail) {
		t.Fatalf("exit %d, stderr %q", code, errOut)
	}
	return rows
}

// writeBaja makes <nh>/modules/baja.jar with the vendorVersion a Niagara installation ships.
func writeBaja(t *testing.T, nh, version string) {
	t.Helper()
	if err := os.MkdirAll(nh+"/modules", 0o755); err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(nh + "/modules/baja.jar")
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	w, err := zw.Create("META-INF/module.xml")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = w.Write([]byte(`<module name="baja" vendor="Tridium" vendorVersion="` + version + `" nre="true">` + "\n"))
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestLedgerNiagara(t *testing.T) {
	const warnFrom = "WARN  ledger-niagara  "
	for _, c := range []struct {
		name, nhName, ledger string // nhName: the niagara_home directory name; ledger: the gradle root's odd/VERSION-LEDGER.md
		want                 string // the row, with @D@ for the temp dir ("" = no row)
	}{
		{"another version", "nh-4.15.6.20", ledgerHead + row("4.13.2.18"),
			warnFrom + "@D@/gr/odd/VERSION-LEDGER.md latest row (Pan 1.2.0) names Niagara 4.13.2.18, but niagara_home is 4.15.6.20 (@D@/nh-4.15.6.20)" +
				" — the module was built against another Niagara version (rebuild against this install, or use the one the ledger names)"},
		{"the same version", "nh-4.15.6.20", ledgerHead + row("4.15.6.20"), ""},
		{"the ledger version is a prefix of the install's", "nh-4.15.6.20", ledgerHead + row("4.15.6"), ""},
		{"only the latest row counts", "nh-4.15.6.20", ledgerHead + row("4.13.2.18") + row("4.15.6.20"), ""},
		{"an older row is not the latest", "nh-4.15.6.20", ledgerHead + row("4.15.6.20") + row("4.13.2.18"),
			warnFrom + "@D@/gr/odd/VERSION-LEDGER.md latest row (Pan 1.2.0) names Niagara 4.13.2.18, but niagara_home is 4.15.6.20 (@D@/nh-4.15.6.20)" +
				" — the module was built against another Niagara version (rebuild against this install, or use the one the ledger names)"},
		{"a ledger without the column", "nh-4.15.6.20", legacyHead + legacyRow, ""},
		{"an empty cell", "nh-4.15.6.20", ledgerHead + row(""), ""},
		{"the unknown marker", "nh-4.15.6.20", ledgerHead + row("—"), ""},
		{"an empty table", "nh-4.15.6.20", ledgerHead, ""},
		{"an install whose version cannot be read", "nh", ledgerHead + row("4.13.2.18"),
			warnFrom + "@D@/gr/odd/VERSION-LEDGER.md latest row (Pan 1.2.0) names Niagara 4.13.2.18, but the version of niagara_home @D@/nh" +
				" could not be read (no modules/baja.jar, no X.Y.Z in its path) — nothing compared"},
		{"no ledger", "nh-4.15.6.20", "", ""},
	} {
		t.Run(c.name, func(t *testing.T) {
			d := t.TempDir()
			mk(t, d, "jvm/", c.nhName+"/modules/", "gr/settings.gradle.kts")
			if c.ledger != "" {
				mk(t, d, "gr/odd/VERSION-LEDGER.md="+c.ledger)
			}
			got := niagaraRows(t, d, d+"/"+c.nhName, d+"/gr")
			want := []string{}
			if c.want != "" {
				want = []string{strings.ReplaceAll(c.want, "@D@", d)}
			}
			if strings.Join(got, "\n") != strings.Join(want, "\n") {
				t.Errorf("rows %q, want %q", got, want)
			}
		})
	}
}

// The install's version is the vendorVersion of its modules/baja.jar, never its directory name.
func TestLedgerNiagaraReadsBajaJar(t *testing.T) {
	d := t.TempDir()
	mk(t, d, "jvm/", "gr/settings.gradle.kts", "gr/odd/VERSION-LEDGER.md="+ledgerHead+row("4.14.0.162"))
	writeBaja(t, d+"/nh-4.13.0", "4.14.0.162")
	if got := niagaraRows(t, d, d+"/nh-4.13.0", d+"/gr"); len(got) != 0 {
		t.Errorf("baja.jar says 4.14.0.162, the ledger too, but rows %q", got)
	}
	writeBaja(t, d+"/nh-4.14.0.162", "4.15.6.20")
	if got := niagaraRows(t, d, d+"/nh-4.14.0.162", d+"/gr"); len(got) != 1 || !strings.Contains(got[0], "but niagara_home is 4.15.6.20") {
		t.Errorf("baja.jar says 4.15.6.20, the directory name 4.14.0.162: rows %q", got)
	}
}

// A client repository keeps one ledger for every module root below it: the parent of the gradle root is read when
// the root has none of its own, and the root's own ledger wins.
func TestLedgerNiagaraParentLedger(t *testing.T) {
	d := t.TempDir()
	mk(t, d, "jvm/", "nh-4.15.6.20/modules/", "client/mod/settings.gradle.kts", "client/odd/VERSION-LEDGER.md="+ledgerHead+row("4.13.2.18"))
	got := niagaraRows(t, d, d+"/nh-4.15.6.20", d+"/client/mod")
	if len(got) != 1 || !strings.Contains(got[0], d+"/client/odd/VERSION-LEDGER.md latest row") {
		t.Errorf("the parent's ledger: rows %q", got)
	}
	mk(t, d, "client/mod/odd/VERSION-LEDGER.md="+ledgerHead+row("4.15.6.20"))
	if got := niagaraRows(t, d, d+"/nh-4.15.6.20", d+"/client/mod"); len(got) != 0 {
		t.Errorf("the root's own ledger wins: rows %q", got)
	}
}

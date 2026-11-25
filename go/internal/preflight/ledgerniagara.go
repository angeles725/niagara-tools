package preflight

import (
	"os"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/ledgercheck"
	"github.com/angeles725/niagara-tools/go/internal/lint"
	"github.com/angeles725/niagara-tools/go/internal/niagaraversion"
)

// ledgerNiagara warns when the latest row of the client's VERSION-LEDGER.md names a Niagara version other than the
// version of this run's niagara_home: the module was built against one Niagara and would be deployed from another
// (the 4.13-built module on a 4.15 JACE of F24; #1026). The ledger is <gradle-root>/odd/VERSION-LEDGER.md, else the
// one beside it in the gradle root's parent (a client repository keeps one for every module root below it). The
// latest row is the last one of the Versions table (newest last), whatever its module. Silent when there is no
// ledger, no Niagara column, or the latest row leaves the cell empty, — or malformed (ledger-check FAILs that).
// The install's version is the vendorVersion of its modules/baja.jar, else the last X.Y.Z of its path (as
// sample-home reads it); the two are equal when the shorter is a leading part of the longer (4.15.6 and 4.15.6.20).
// WARN only: the exit code is unchanged.
func (r *run) ledgerNiagara() {
	if r.win {
		return
	}
	path, latest, ok := r.latestLedgerRow()
	if !ok || latest.Niagara == "" {
		return
	}
	have, _, err := niagaraversion.InstallVersion(r.nh)
	if err != nil {
		have = lastVersion(r.nh)
	}
	where := path + " latest row (" + latest.Module + " " + latest.Version + ") names Niagara " + latest.Niagara + ", but "
	switch {
	case have == "":
		r.row("WARN", "ledger-niagara", where+"the version of niagara_home "+r.nh+
			" could not be read (no modules/baja.jar, no X.Y.Z in its path) — nothing compared")
	case !sameNiagara(latest.Niagara, have):
		r.row("WARN", "ledger-niagara", where+"niagara_home is "+have+" ("+r.nh+
			") — the module was built against another Niagara version (rebuild against this install, or use the one the ledger names)")
	}
}

// latestLedgerRow is the ledger's path and its latest row; false when no ledger is readable or it has no
// Niagara column.
func (r *run) latestLedgerRow() (string, ledgercheck.Latest, bool) {
	dirs := []string{r.gr}
	if parent := lint.Dirname(r.gr); !sharedDir(parent) {
		dirs = append(dirs, parent)
	}
	for _, d := range dirs {
		path := strings.TrimRight(d, "/") + "/odd/VERSION-LEDGER.md"
		if b, err := os.ReadFile(path); err == nil {
			latest, ok := ledgercheck.LatestRow(string(b))
			return path, latest, ok
		}
	}
	return "", ledgercheck.Latest{}, false
}

// sameNiagara is the equality of two dotted versions up to the shorter one's length.
func sameNiagara(a, b string) bool {
	pa, pb := strings.Split(a, "."), strings.Split(b, ".")
	for i := 0; i < len(pa) && i < len(pb); i++ {
		if pa[i] != pb[i] {
			return false
		}
	}
	return true
}

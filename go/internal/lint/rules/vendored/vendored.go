// Package vendored is the lint-vendored rule (issue #1123): the ledger of vendored browser libraries,
// <rc-dir>/vendor/VENDORED.md (types/frontend-standard.md, Vendored libraries). A RootRule over an rc dir, all
// WARN (exit 1 under --strict):
//
//	a vendored file whose library and version the known-defects table lists   WARN, names the defect and its fix
//	a vendored file that no ledger row lists                                  WARN, says what row to add
//	rc/vendor holds vendored files and there is no VENDORED.md                WARN once, says how to create it
//	a ledger row with an empty cell or a version that is no r-number or semver  WARN
//
// An rc dir with no vendor/, or one whose vendor/ holds only the ledger, THIRD-PARTY.md and licence files, has
// nothing to check (clean). The known-defects table is a data file of the kit, handed over with --defects (the
// wrapper toolbelt/lint-vendored.sh passes build-n4-module-kit/data/vendored-defects.md); one that cannot be read
// whole is an env fault (exit 3). The sibling lint-vendor-floor parses the files with node; this rule reads text.
package vendored

import (
	"errors"
	"os"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/lint"
)

// ID is the lint name printed in every row.
const ID = "lint-vendored"

// LedgerName is the ledger file in rc/vendor.
const LedgerName = "VENDORED.md"

// Spec registers the rule with the contract of toolbelt/lint-vendored.sh: [--strict] [--defects <file>] <rc-dir>.
// An rc dir with no vendor files is clean (EmptyOK), as lint-vendor-floor leaves an rc dir with no vendor/.
func Spec() lint.Spec {
	return lint.Spec{Root: rule{}, Mode: lint.Valued, Option: "--defects", ValuedStrict: true, EmptyOK: true,
		BarePathRow: true, Usage: "usage: lint-vendored.sh [--strict] [--defects <file>] <rc-dir>"}
}

type rule struct{ defects string } // defects: the --defects file of this run, "" when not given

func (rule) ID() string              { return ID }
func (rule) Severity() lint.Severity { return lint.Warn }

// Configure keeps the --defects value for the run.
func (rule) Configure(a lint.Args) (lint.Configured, error) { return rule{defects: a.Option}, nil }

// CheckRoot checks the vendored files under root/vendor against the ledger and the known-defects table.
func (r rule) CheckRoot(root string) ([]lint.Finding, error) {
	root = strings.TrimRight(root, "/")
	dir := root + "/vendor"
	if st, err := os.Stat(dir); err != nil && !os.IsNotExist(err) {
		return nil, errors.New("cannot read " + dir + ": " + err.Error())
	} else if err != nil || !st.IsDir() {
		return nil, nil
	}
	files, err := lint.Walk(dir, lint.Selection{Names: []string{"*"}})
	if err != nil {
		return nil, errors.New("cannot list every file under " + dir + ": " + err.Error())
	}
	var vendored []string // paths relative to rc/vendor
	for _, f := range files {
		if rel := strings.TrimPrefix(f, dir+"/"); !isNote(rel) {
			vendored = append(vendored, rel)
		}
	}
	if len(vendored) == 0 {
		return nil, nil
	}
	defects, err := loadDefects(r.defects)
	if err != nil {
		return nil, err
	}
	ledger := dir + "/" + LedgerName
	if !lint.IsFile(ledger) {
		return []lint.Finding{{Path: dir, Msg: "has vendored files but no " + LedgerName + "; create it with the table | " +
			strings.Join(ledgerHeader, " | ") + " | and one row per vendored file (types/frontend-standard.md, Vendored libraries)"}}, nil
	}
	entries, out, found, err := readLedger(ledger)
	if err != nil {
		return nil, errors.New("cannot read " + ledger + ": " + err.Error())
	}
	if !found {
		return []lint.Finding{{Path: ledger, Msg: "has no table with the header | " + strings.Join(ledgerHeader, " | ") + " |"}}, nil
	}
	for _, rel := range vendored {
		e, listed := entries[rel]
		if !listed {
			out = append(out, lint.Finding{Path: dir + "/" + rel, Msg: "not listed in rc/vendor/" + LedgerName +
				"; add a row: | " + rel + " | <library> | <version> | <source URL> |"})
			continue
		}
		out = append(out, defectRows(dir+"/"+rel, e, defects)...)
	}
	return out, nil
}

// isNote: a file of rc/vendor that is no vendored code: the ledger and THIRD-PARTY.md at the top, and a
// LICENSE, COPYING or NOTICE file anywhere.
func isNote(rel string) bool {
	if rel == LedgerName || rel == "THIRD-PARTY.md" {
		return true
	}
	base := strings.ToUpper(rel[strings.LastIndex(rel, "/")+1:])
	for _, p := range []string{"LICENSE", "LICENCE", "COPYING", "NOTICE"} {
		if strings.HasPrefix(base, p) {
			return true
		}
	}
	return false
}

// defectRows is the WARN of every known defect that e's library and version carry.
func defectRows(path string, e entry, defects []defect) []lint.Finding {
	if !e.verOK {
		return nil // the ledger row already has its own WARN
	}
	var out []lint.Finding
	who := e.lib + " " + e.version + " (" + LedgerName + ":" + itoa(e.line) + ")"
	for _, d := range defects {
		if !d.applies(e.lib, e.file) {
			continue
		}
		hit, ok := affects(d.cons, e.ver)
		switch {
		case !ok:
			out = append(out, lint.Finding{Path: path, Msg: who + ": cannot compare with the known-defects table entry " +
				joinRaw(d.cons) + " (" + d.text + "); check by hand"})
		case hit:
			out = append(out, lint.Finding{Path: path, Msg: who + " has a known defect: " + d.text + "; fix: " + d.fix})
		}
	}
	return out
}

func joinRaw(cs []constraint) string {
	raw := make([]string, len(cs))
	for i, c := range cs {
		raw[i] = c.raw
	}
	return strings.Join(raw, ", ")
}

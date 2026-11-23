// Package rdbmsdialect is lint-rdbms-dialect-descriptor (G88): a module that fixes the database dialect of a
// history export in code, or reaches for Postgres, which the platform ships no dialect for. The contract is the
// G88 spec (build-n4-module-kit/ledger.d/2026-10-23-r8-u9-c68-c71.md); the evidence and the fix are
// types/cloud-connector.md § 11; toolbelt/lint-rdbms-dialect-descriptor.sh is the wrapper and its header is
// the user-facing contract.
//
// It is a lint.TreeRule: every *.java under a -rt part's src/ of one module root (srcTest and build pruned).
// Matching is on code: comments are blanked, and a `new` expression is matched with literals blanked as well,
// so a copy in a comment or a string never matches; the Postgres URL is matched in the literal itself.
package rdbmsdialect

import (
	"regexp"
	"sort"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
	"github.com/angeles725/niagara-tools/go/internal/lint"
	"github.com/angeles725/niagara-tools/go/internal/lint/rules/linescan"
)

// ID is the lint name every row prints in its check column.
const ID = "lint-rdbms-dialect-descriptor"

// Spec registers the rule: [--strict] <module-root>... (WARN rows: exit 0, 1 under --strict; each root on its
// own). A root with no -rt part src/ file is the ERROR row (exit 3). Its roots are module roots, so n4lint --all
// leaves it out (Spec.ModuleRoot).
func Spec() lint.Spec {
	return lint.Spec{Tree: rule{}, Mode: lint.Each, ModuleRoot: true,
		Usage:      "usage: lint-rdbms-dialect-descriptor.sh [--strict] <module-root>...",
		Select:     lint.Selection{Names: []string{"*.java"}, Prune: []string{"srcTest", "build"}},
		EmptyError: lint.EmptyRow{Msg: "no *.java under a -rt part's src/ (wrong path: pass the module root)"}}
}

type rule struct{}

func (rule) ID() string              { return ID }
func (rule) Severity() lint.Severity { return lint.Warn }

const (
	msgFixed    = "hard-coded dialect descriptor: take it from getHistories().getExportDescriptorType().getInstance() (types/cloud-connector.md § 11)"
	msgPostgres = "postgres jdbc url: Postgres has no shipped dialect, ship a BRdbms subclass (types/cloud-connector.md § 11)"
	pgURL       = "jdbc:postgresql:"
)

var (
	// rtSrc is a file under <part>-rt/src/.
	rtSrc = regexp.MustCompile(`(^|/)[^/]+-rt/src/`)
	// fixed is a history export of one of the four shipped dialects built in code; whitespace, newlines
	// included, may sit before the parenthesis.
	fixed = regexp.MustCompile(`\bnew\s+B(?:SqlServer|MySQL|Oracle|HsqlDb)HistoryExport\s*\(`)
	// dialect is a class extending BRdbms: the module ships its own dialect.
	dialect = regexp.MustCompile(`\bextends\s+BRdbms\b`)
)

// CheckTree reports the fixed dialects, and the Postgres URLs when no class of the module extends BRdbms.
func (rule) CheckTree(files []*javasrc.File) ([]lint.Finding, error) {
	var part []*javasrc.File
	own := false
	for _, f := range files {
		if rtSrc.MatchString(f.Path) {
			part = append(part, f)
			own = own || dialect.MatchString(strings.Join(linescan.Code(f), "\n"))
		}
	}
	if len(part) == 0 {
		return nil, lint.ErrEmptyRoot
	}
	var out []lint.Finding
	for _, f := range part {
		var rows []lint.Finding
		code := strings.Join(linescan.Code(f), "\n")
		for _, m := range fixed.FindAllStringIndex(code, -1) {
			rows = append(rows, lint.Finding{Path: f.Path, Line: 1 + strings.Count(code[:m[0]], "\n"), Msg: msgFixed})
		}
		if !own {
			for i, line := range f.Stripped() {
				if strings.Contains(line, pgURL) {
					rows = append(rows, lint.Finding{Path: f.Path, Line: i + 1, Msg: msgPostgres})
				}
			}
		}
		sort.SliceStable(rows, func(a, b int) bool { return rows[a].Line < rows[b].Line })
		out = append(out, rows...)
	}
	return out, nil
}

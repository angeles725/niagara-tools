// Package inlinesiteliteral is lint-inline-site-literal (G42): site or station data (an ORD, an IPv4 address,
// a site or room name) written as a literal outside the module's site table, so it cannot be changed in one
// place. The contract is the deferred G42 spec (build-n4-module-kit/ledger.d/2026-10-22-modular-line-efficient-
// quality.md); the doctrine is METHODOLOGY.md § Domain correctness "site and station data in one table";
// toolbelt/lint-inline-site-literal.sh is the wrapper and its header is the user-facing contract.
//
// It is a lint.TreeRule over one module root: every *.java under a -rt or -ux part's src/ and every *.js under
// an rc/ directory, srcTest/ (test sources), build/ and node_modules/ not entered. Java literals come from the
// shared scanner (javasrc Strip + BlankLiterals: comments out), JavaScript literals from jssrc (strings and
// template literals; comments and regex literals are not literals). A file is the site table when
// --site-table names it or its first line starts with "// site-table"; its literals are never rows.
package inlinesiteliteral

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
	"github.com/angeles725/niagara-tools/go/internal/jssrc"
	"github.com/angeles725/niagara-tools/go/internal/lint"
)

// ID is the lint name n4lint accepts and every row prints in its check column (the kit's lint-<name>
// convention; the deferred spec's row wrote "inline-site-literal").
const ID = "lint-inline-site-literal"

// Spec registers the rule: [--strict] [--site-table <path>]... [--site-pattern <regex>]... <module-root> in any
// order (Mode Valued), WARN rows (exit 1 under --strict), left out of n4lint --all (a module root, and its
// options exist only when it runs alone).
func Spec() lint.Spec {
	return lint.Spec{
		Tree: &rule{}, Mode: lint.Valued, Options: []string{"--site-table", "--site-pattern"}, ValuedStrict: true, ModuleRoot: true,
		Usage:      "usage: lint-inline-site-literal.sh [--strict] [--site-table <path>]... [--site-pattern <regex>]... <module-root>",
		Select:     lint.Selection{Names: []string{"*.java", "*.js"}, Prune: []string{"srcTest", "build", "node_modules"}},
		EmptyError: lint.EmptyRow{Msg: "no *.java under a -rt or -ux part's src/ and no *.js under an rc/ (wrong path: pass the module root)"},
	}
}

// rule holds the options Configure read (a configured copy; the registered rule holds none): the site-table files (absolute) and the site patterns.
type rule struct {
	tables   map[string]bool
	patterns []*regexp.Regexp
}

func (*rule) ID() string              { return ID }
func (*rule) Severity() lint.Severity { return lint.Warn }

var (
	javaPart = regexp.MustCompile(`(^|/)[^/]+-(rt|ux)/src/`)
	rcDir    = regexp.MustCompile(`(^|/)rc/`)
	// slotPath is each slot ORD part's path, alone or as one part of an ORD ("slot:/Rooms/Room1",
	// "station:|slot:/Drivers|bql:..."); siteSlot decides which paths are site data.
	slotPath = regexp.MustCompile(`(?:^|\|)slot:/([^|]*)`)
	// handle is a handle ORD, alone or as one part of an ORD ("h:2e4f", "station:|h:2e4f"); "h:mm" is not.
	handle = regexp.MustCompile(`(^|\|)h:[0-9a-fA-F]+($|\|)`)
	ipv4   = regexp.MustCompile(`(^|[^0-9.])((25[0-5]|2[0-4][0-9]|1[0-9][0-9]|[1-9]?[0-9])\.){3}(25[0-5]|2[0-4][0-9]|1[0-9][0-9]|[1-9]?[0-9])($|[^0-9.])`)
)

const maxShown = 72 // bytes of a literal a row shows before "..."

// Configure reads --site-table (each a regular file) and --site-pattern (each a valid regex).
func (*rule) Configure(a lint.Args) (lint.Configured, error) {
	r := &rule{tables: map[string]bool{}}
	for _, p := range a.Values["--site-table"] {
		if st, err := os.Stat(p); err != nil || !st.Mode().IsRegular() {
			return nil, fmt.Errorf("%s: --site-table file not found: %s\n", ID, p)
		}
		r.tables[abs(p)] = true
	}
	for _, p := range a.Values["--site-pattern"] {
		re, err := regexp.Compile(p)
		if err != nil {
			return nil, fmt.Errorf("%s: bad --site-pattern %s: %v\n", ID, p, err)
		}
		r.patterns = append(r.patterns, re)
	}
	return r, nil
}

// CheckTree reports, per file in walk order, the first site literal of each line outside the site table.
func (r *rule) CheckTree(files []*javasrc.File) ([]lint.Finding, error) {
	var out []lint.Finding
	read := false
	for _, f := range files {
		var lits []literal
		switch {
		case strings.HasSuffix(f.Path, ".java") && javaPart.MatchString(f.Path):
			javasrc.BlankLiterals(f.Stripped(), func(l javasrc.Literal) {
				if l.Quote == '"' {
					lits = append(lits, literal{l.Line, javasrc.Unescape(l.Text)})
				}
			})
		case strings.HasSuffix(f.Path, ".js") && rcDir.MatchString(f.Path):
			for _, t := range jssrc.Tokens(f.Raw) {
				if t.Kind == jssrc.String || t.Kind == jssrc.Template {
					lits = append(lits, literal{t.Line, javasrc.Unescape(t.Text)})
				}
			}
		default:
			continue
		}
		read = true
		if r.siteTable(f) {
			continue
		}
		last := -1
		for _, l := range lits {
			if kind := r.kind(l.text); kind != "" && l.line != last {
				last = l.line
				out = append(out, lint.Finding{Path: f.Path, Line: l.line + 1,
					Msg: kind + ` literal outside the site table: "` + shown(l.text) + `"`})
			}
		}
	}
	if !read {
		return nil, lint.ErrEmptyRoot
	}
	return out, nil
}

// literal is one string or template literal: its 0-based line and its text, escapes read.
type literal struct {
	line int
	text string
}

// siteTable reports a file --site-table names or whose first line starts with "// site-table".
func (r *rule) siteTable(f *javasrc.File) bool {
	if r.tables[abs(f.Path)] {
		return true
	}
	return len(f.Raw) > 0 && strings.HasPrefix(strings.TrimSpace(f.Raw[0]), "// site-table")
}

// kind classifies a literal: "ord" (a slot ORD below the root or a handle ORD), "ip" (holds an IPv4 address), "site"
// (matches a --site-pattern), or "" when it is none.
func (r *rule) kind(s string) string {
	switch {
	case siteSlot(s) || handle.MatchString(s):
		return "ord"
	case ipv4.MatchString(s):
		return "ip"
	}
	for _, re := range r.patterns {
		if re.MatchString(s) {
			return "site"
		}
	}
	return ""
}

// siteSlot reports a slot ORD part naming site data: a path below the station root that is not the Services
// folder or under it. A query ORD that names no slot below the root ("station:|alarm:|bql:...",
// "station:|slot:/|bql:...") reads the same on every station (tuning 2026-10-24), and so does a service ORD
// ("station:|slot:/Services/ChiDashboardService", any path under /Services): module structure, not site data
// (decision 2026-10-24, retro go-lint-inline-site-literal Δ2).
func siteSlot(s string) bool {
	for _, m := range slotPath.FindAllStringSubmatch(s, -1) {
		if p := m[1]; p != "" && p != "Services" && !strings.HasPrefix(p, "Services/") {
			return true
		}
	}
	return false
}

// shown is the literal as a row prints it: at most maxShown bytes, then "...".
func shown(s string) string {
	if len(s) > maxShown {
		return s[:maxShown] + "..."
	}
	return s
}

// abs is p made absolute and clean (p itself when that fails).
func abs(p string) string {
	if a, err := filepath.Abs(p); err == nil {
		return a
	}
	return filepath.Clean(p)
}

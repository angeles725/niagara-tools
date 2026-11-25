// Package bqlunbounded is the lint-bql-unbounded rule: a bql: query literal in module Java or JavaScript whose
// select is unbounded. WARN rows (FAIL under --strict), one SUMMARY row for the run. The station evaluates a BQL
// select as extent -> where -> aggregate -> distinct -> order -> top -> projection (bql-rt
// SelectQuery.java:62-87 in organized/bql/bql-rt/vineflower/com/tridium/bql/), so only a top <n> stops the read;
// BTopTable.java:57-63 (collection/, same tree) counts rows and cuts the cursor, after an order by has sorted
// everything. [CERT] types/queries.md §3.
//
//	no-top            WARN  a query with no top <n> on an ORD that is not history: or alarm: (a tree walk, a station
//	                        table, a history: listing): every row of the extent comes back.
//	history-no-window WARN  a history:<path> query with no top <n> and no timestamp window: a ?period=, start= or
//	                        end= parameter of the ORD (HistoryQuery grammar, bajadoc-clean javax/baja/history/
//	                        HistoryQuery.txt:7-19) or a timestamp predicate in the where clause. With no predicate
//	                        BHistoryDbTable.bqlQuery scans the whole history; a timestamp range is the one thing it
//	                        pushes down as timeQuery (organized/history/history-rt/vineflower/com/tridium/history/db/
//	                        BHistoryDbTable.java:232-262, scan at :249-250, timeQuery at :252-261).
//	alarm-unindexed   WARN  an alarm: query with no top <n> and none of the four index paths, `from openAlarms`,
//	                        `from ackPendingAlarms` (matched with the exact case the station compares, equals), a
//	                        `uuid =` or `source =` predicate, nor a timestamp range (a fifth path, timeQuery). Any
//	                        other alarm query is scan() of the whole alarm database (niagara-help/source/alarm-rt/
//	                        javax/baja/alarm/AlarmDbConnection.java:52-110; types/queries.md §3, §6).
//	order-by-no-top   WARN  a bounded query (a window or an index path) with an order by and no top <n>: order by
//	                        materializes every row it is given before any cap (SelectQuery.java:75-77, then top at
//	                        :79-81). An unbounded query with an order by keeps its own check and says so in its text.
//	waiver-no-reason  FAIL  a `// lint-bql-unbounded:` marker with nothing after the colon; the query it sits on is
//	                        not waived.
//
// Reading: the literals of the file (javasrc.Tokens over the comment-stripped Java lines, jssrc.Tokens for .js: a
// comment, a Javadoc or a text block quoted in a comment is no code; a """ text block and a JavaScript template
// without ${} are literals). A literal is a query when `bql:` (after the start, a | or a blank) is followed by
// select or from; the query ends at a | outside single quotes (a |view: suffix). `top <n>` is read in the select
// head before from, with a number or a %d, %s, ? or {n} placeholder. A literal that is added to something else
// with + or +=, or a JavaScript template with a substitution, is built at run time: it is counted in the SUMMARY
// row (non-literal-skipped), never judged; a run of literals added only to each other is one literal.
// Waiver: `// lint-bql-unbounded: <reason>` on the line of the literal's first byte or on the comment-only line above.
//
// Limits, false negatives: only a literal that holds bql: is seen (BqlQuery.make("select ..."), a name or a
// constant passed to BOrd.make, and a built string that shows no bql: fragment are not); a window passed as a
// Context or a loop that stops early is not seen. False positives: an aggregate (count(*)) has one row but reads
// them all and top does not help it; an `or` in the where clause makes a timestamp predicate or a uuid/source
// predicate count for nothing (a disjoint range would be bounded); delta=true is not a window. Waive those.
package bqlunbounded

import (
	"fmt"
	"sort"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
	"github.com/angeles725/niagara-tools/go/internal/lint"
)

// ID is the lint name printed in every row.
const ID = "lint-bql-unbounded"

// Spec registers the rule with the argument contract of toolbelt/lint-bql-unbounded.sh. StrictFails prints the
// WARN rows as FAIL under --strict; Joined gives the rule every root's files, for one SUMMARY row.
func Spec() lint.Spec {
	return lint.Spec{
		Tree:        rule{},
		Mode:        lint.Each,
		Joined:      true,
		Usage:       "usage: lint-bql-unbounded.sh [--help] [--strict] <src-root>...",
		Select:      lint.Selection{Names: []string{"*.java", "*.js"}, Except: []string{"*.min.js"}, Prune: []string{"build", "node_modules"}},
		EmptyError:  lint.EmptyRow{Msg: "no *.java or *.js sources found (wrong path or empty source tree)"},
		StrictFails: true,
	}
}

type rule struct{}

func (rule) ID() string              { return ID }
func (rule) Severity() lint.Severity { return lint.Warn }

// stats are the SUMMARY counts: queries judged, flagged, waived, and built strings left unjudged.
type stats struct{ checked, flagged, waived, skipped int }

// CheckTree reports the flagged queries of every file in walk order, then the SUMMARY row (none when the tree
// holds no query and no built string).
func (rule) CheckTree(files []*javasrc.File) ([]lint.Finding, error) {
	var out []lint.Finding
	var total stats
	for _, f := range files {
		fds, st := scan(f)
		out = append(out, fds...)
		total.checked += st.checked
		total.flagged += st.flagged
		total.waived += st.waived
		total.skipped += st.skipped
	}
	if total.checked == 0 && total.skipped == 0 {
		return out, nil // no query at all: no SUMMARY row, so n4lint --all stays quiet on a tree without BQL
	}
	msg := fmt.Sprintf("checked=%d flagged=%d waived=%d non-literal-skipped=%d", total.checked, total.flagged, total.waived, total.skipped)
	return append(out, lint.Finding{Severity: lint.Summary, Msg: msg}), nil
}

// scan judges the queries of one file; its rows are in line order, a bare marker before the query on its line.
func scan(f *javasrc.File) ([]lint.Finding, stats) {
	var out []lint.Finding
	var st stats
	for _, n := range bare(f.Raw) {
		out = append(out, lint.Finding{Path: f.Path, Line: n, Severity: lint.Fail, Msg: "waiver-no-reason: " + marker + " needs a reason after the colon"})
	}
	for _, e := range exprs(f) {
		if !bqlAt.MatchString(e.text) {
			continue
		}
		if e.dynamic {
			st.skipped++
			continue
		}
		q, ok := find(e.text)
		if !ok {
			continue
		}
		st.checked++
		v, bad := judge(q)
		switch {
		case !bad:
		case waived(f.Raw, e.line):
			st.waived++
		default:
			st.flagged++
			out = append(out, lint.Finding{Path: f.Path, Line: e.line, Msg: v.check + `: "` + snippet(q) + `"` + v.text + " (" + v.see + ")"})
		}
	}
	sort.SliceStable(out, func(a, b int) bool { return out[a].Line < out[b].Line })
	return out, st
}

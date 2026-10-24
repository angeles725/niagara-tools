package citeresolve

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/citeparse"
	"github.com/angeles725/niagara-tools/go/internal/cli"
)

// Exit codes of kit-cite-resolve.
const (
	ExitOK    = 0 // every pointer resolved, or findings without --strict
	ExitFound = 1 // under --strict, any pointer that is a finding (not RESOLVED, HISTORICAL_DRIFT, UNVERIFIABLE_KIND or SKIPPED_TEMPLATE)
	ExitEnv   = 3 // usage, a missing or unreadable root or card, or a TOOL_ERROR row
)

const usage = "usage: kit-cite-resolve [--kit <dir>] [--research <dir>] [--modules <dir>] [--memory <dir>]" +
	" [--clients <dir>] [--version <ver>] [--format tsv|json] [--strict] <card.md>...\n"

// order is the status order of the summary line.
var order = []Status{Resolved, MissingFile, MissingSection, MissingDelta, LineOutOfRange, BareID, Unparsed, Ambiguous, ToolError,
	HistoricalDrift, UnverifiableKind, SkippedTemplate}

// Row is one output row.
type Row struct {
	Card    string `json:"card"`
	Line    int    `json:"line"`
	Kind    string `json:"kind"`
	Target  string `json:"target"`
	Locator string `json:"locator"`
	Status  Status `json:"status"`
	Detail  string `json:"detail"`
	Scope   Scope  `json:"scope"`
}

// Main implements kit-cite-resolve: one row per evidence pointer of each card, then a summary on stderr.
// It is a pre-filter: RESOLVED says the cited place exists, never that it says what the card claims.
func Main(args []string, s cli.Stdio) int {
	fs := flag.NewFlagSet("kit-cite-resolve", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	home := s.Env("HOME")
	var r Roots
	fs.StringVar(&r.Kit, "kit", "", "kit root (default: the nearest directory above the first card holding retros/INDEX.md)")
	fs.StringVar(&r.Research, "research", filepath.Join(home, "niagara-research"), "research repository")
	fs.StringVar(&r.Modules, "modules", filepath.Join(home, "modulos_niagara_n4"), "client module trees")
	fs.StringVar(&r.Clients, "clients", "", "client checkouts searched by base name (default: <modules>/Cliente)")
	fs.StringVar(&r.Memory, "memory", "", "memory directory (default: the Claude project memory of --research)")
	fs.StringVar(&r.Version, "version", "4.15.6.20", "niagara-help version of unversioned devguide/guide pointers")
	format := fs.String("format", "tsv", "tsv or json")
	strict := fs.Bool("strict", false, "exit 1 on any finding")
	if err := fs.Parse(args); err != nil || fs.NArg() == 0 {
		fmt.Fprint(s.Err, usage)
		return ExitEnv
	}
	if *format != "tsv" && *format != "json" {
		fmt.Fprintf(s.Err, "kit-cite-resolve: --format must be tsv or json, not %q\n", *format)
		return ExitEnv
	}
	cards := fs.Args()
	if r.Kit == "" {
		if r.Kit = findKit(cards[0]); r.Kit == "" {
			fmt.Fprintf(s.Err, "kit-cite-resolve: no kit root (retros/INDEX.md) above %s; pass --kit\n", cards[0])
			return ExitEnv
		}
	}
	if r.Memory == "" {
		abs, _ := filepath.Abs(r.Research)
		r.Memory = filepath.Join(home, ".claude", "projects", strings.ReplaceAll(abs, "/", "-"), "memory")
	}
	if r.Clients == "" {
		r.Clients = filepath.Join(r.Modules, "Cliente")
	}
	r.Repo = filepath.Dir(filepath.Clean(r.Kit))
	for _, root := range []struct{ name, dir string }{{"kit", r.Kit}, {"research", r.Research}, {"modules", r.Modules},
		{"clients", r.Clients}, {"memory", r.Memory}} {
		if _, err := os.ReadDir(root.dir); err != nil {
			fmt.Fprintf(s.Err, "kit-cite-resolve: cannot read %s root %s: %v\n", root.name, root.dir, err)
			return ExitEnv
		}
	}
	texts := make([]string, len(cards))
	for i, c := range cards {
		b, err := os.ReadFile(c)
		if err != nil {
			fmt.Fprintf(s.Err, "kit-cite-resolve: cannot read card %s: %v\n", c, err)
			return ExitEnv
		}
		texts[i] = string(b)
	}
	rows := resolveAll(New(r), NewClassifier(r.Kit, HistoryRules), cards, texts)
	write(s.Out, rows, *format)
	return summarize(s.Err, rows, *strict)
}

func resolveAll(e *Engine, classify Classifier, cards, texts []string) []Row {
	var rows []Row
	for i, path := range cards {
		c := e.NewCard(path)
		scope, family := classify(path)
		for _, p := range citeparse.Card(texts[i]) {
			res := scoped(e.Resolve(p, c), scope, family)
			target := strings.TrimSpace(p.Target + " " + p.File)
			rows = append(rows, Row{path, p.Line, string(p.Kind), target, p.Locator(), res.Status, res.Detail, scope})
		}
	}
	return rows
}

func write(w io.Writer, rows []Row, format string) {
	if format == "json" {
		fmt.Fprint(w, "[")
		for i, r := range rows {
			b, _ := json.Marshal(r)
			if i > 0 {
				fmt.Fprint(w, ",")
			}
			fmt.Fprintf(w, "\n%s", b)
		}
		fmt.Fprint(w, "\n]\n")
		return
	}
	cell := strings.NewReplacer("\t", " ", "\n", " ", "\r", " ")
	for _, r := range rows {
		fmt.Fprintf(w, "%s\t%d\t%s\t%s\t%s\t%s\t%s\n", cell.Replace(r.Card), r.Line, r.Kind, cell.Replace(r.Target),
			cell.Replace(r.Locator), r.Status, cell.Replace(r.Detail))
	}
}

// summarize prints the per-status counts of all rows, then of the LIVE and of the HISTORY rows, then the
// findings of each scope, and returns the exit code.
func summarize(w io.Writer, rows []Row, strict bool) int {
	byScope := map[Scope][]Row{}
	findings := map[Scope]int{}
	for _, r := range rows {
		byScope[r.Scope] = append(byScope[r.Scope], r)
		if !passing[r.Status] {
			findings[r.Scope]++
		}
	}
	n := counts(rows)
	fmt.Fprintf(w, "kit-cite-resolve: %d pointers: %s\n", len(rows), countList(n))
	for _, sc := range []Scope{Live, History} {
		fmt.Fprintf(w, "kit-cite-resolve: %s %d pointers: %s\n", sc, len(byScope[sc]), countList(counts(byScope[sc])))
	}
	fmt.Fprintf(w, "kit-cite-resolve: strict findings: %s %d, %s %d\n", Live, findings[Live], History, findings[History])
	switch {
	case n[ToolError] > 0:
		return ExitEnv
	case strict && findings[Live]+findings[History] > 0:
		return ExitFound
	}
	return ExitOK
}

func counts(rows []Row) map[Status]int {
	n := map[Status]int{}
	for _, r := range rows {
		n[r.Status]++
	}
	return n
}

// countList renders the non-zero counts in status order, or "none".
func countList(n map[Status]int) string {
	var parts []string
	for _, st := range order {
		if n[st] > 0 {
			parts = append(parts, fmt.Sprintf("%s %d", st, n[st]))
		}
	}
	if len(parts) == 0 {
		return "none"
	}
	return strings.Join(parts, ", ")
}

// findKit is the nearest directory at or above card's directory holding retros/INDEX.md.
func findKit(card string) string {
	d, err := filepath.Abs(filepath.Dir(card))
	if err != nil {
		return ""
	}
	for {
		if st, err := os.Stat(filepath.Join(d, "retros", "INDEX.md")); err == nil && !st.IsDir() {
			return d
		}
		parent := filepath.Dir(d)
		if parent == d {
			return ""
		}
		d = parent
	}
}

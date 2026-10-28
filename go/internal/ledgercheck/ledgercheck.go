// Package ledgercheck is ledger-check: a client's VERSION-LEDGER.md checked against the kit's
// templates/VERSION-LEDGER.md (audit 2026-10-03 A9; the 2026-09-26 version-defect-ledger retro, Δ1). The contract is read
// from the template itself, never restated here: its title, its `##` sections, the Versions table header and the
// Status vocabulary of its "`Status` is one of ..." sentence. Run it through toolbelt/ledger-check.sh, which points
// LEDGER_CHECK_TEMPLATE at the kit's template.
//
//	LC1  the first non-empty line is not a `# VERSION-LEDGER` title                    FAIL
//	LC2  a `##` section of the template is missing                                      FAIL
//	LC3  the Versions table header is not the template's, cell for cell                 FAIL
//	LC4  a Versions row with another cell count, or an empty Module or Version cell     FAIL
//	LC5  a Status outside the template's vocabulary                                     FAIL
//	LC6  a Date that is not a real YYYY-MM-DD date                                      FAIL
//	LC7  a Class that is neither — nor a class of the ledger's Root-cause classes table FAIL
//	LC8  a Defect found with Root cause or Class left —                                 FAIL
//	LC9  a template placeholder (<...>) left in a Versions row                          WARN
//
// Rows: "<FAIL|WARN>  ledger-check  <file>[:<line>]  LC<n>: <reason>". Exit 0 clean or WARN-only, 1 any FAIL, 3
// usage, an unreadable ledger, or a template that cannot be read or holds no Versions table or Status sentence.
package ledgercheck

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

// TemplateEnv names the template path the toolbelt wrapper sets; --template overrides it.
const TemplateEnv = "LEDGER_CHECK_TEMPLATE"

const usage = "usage: ledger-check.sh [--template <VERSION-LEDGER template>] <VERSION-LEDGER.md>\n"

const none = "—"

// contract is what the template fixes for every client ledger.
type contract struct {
	sections []string // the template's `## ` headings, in order
	header   []string // the Versions table header cells
	statuses []string // the Status vocabulary
}

var (
	placeholder = regexp.MustCompile(`<[^<>|]+>`)
	backticked  = regexp.MustCompile("`([^`]+)`")
	isoDate     = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
)

// Main runs ledger-check.
func Main(args []string, s cli.Stdio) int {
	tpl := s.Env(TemplateEnv)
	var file string
	for i := 0; i < len(args); i++ {
		switch a := args[i]; {
		case a == "--template" && i+1 < len(args):
			i++
			tpl = args[i]
		case strings.HasPrefix(a, "-") || file != "":
			fmt.Fprint(s.Err, usage)
			return 3
		default:
			file = a
		}
	}
	if file == "" {
		fmt.Fprint(s.Err, usage)
		return 3
	}
	if tpl == "" {
		fmt.Fprintf(s.Err, "ledger-check: no template (set %s or pass --template)\n", TemplateEnv)
		return 3
	}
	c, err := readContract(tpl)
	if err != nil {
		fmt.Fprintf(s.Err, "ledger-check: %v\n", err)
		return 3
	}
	b, err := os.ReadFile(file)
	if err != nil {
		fmt.Fprintf(s.Err, "ledger-check: cannot read %s: %v\n", file, err)
		return 3
	}
	rows := check(c, file, strings.Split(strings.ReplaceAll(string(b), "\r\n", "\n"), "\n"))
	code := 0
	for _, r := range rows {
		fmt.Fprintln(s.Out, r)
		if strings.HasPrefix(r, "FAIL") {
			code = 1
		}
	}
	return code
}

// readContract reads the template's sections, Versions header and Status vocabulary.
func readContract(path string) (contract, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return contract{}, fmt.Errorf("cannot read template %s: %v", path, err)
	}
	text := string(b)
	lines := strings.Split(text, "\n")
	var c contract
	for _, l := range lines {
		if strings.HasPrefix(l, "## ") {
			c.sections = append(c.sections, strings.TrimSpace(l))
		}
	}
	if at := tableAfter(lines, "## Versions"); at >= 0 {
		c.header = cells(lines[at])
	}
	if len(c.header) == 0 {
		return contract{}, errors.New("template has no Versions table: " + path)
	}
	const lead = "`Status` is one of"
	if i := strings.Index(text, lead); i >= 0 {
		rest := text[i+len(lead):]
		if end := strings.Index(rest, "."); end >= 0 {
			for _, m := range backticked.FindAllStringSubmatch(rest[:end], -1) {
				c.statuses = append(c.statuses, m[1])
			}
		}
	}
	if len(c.statuses) == 0 {
		return contract{}, errors.New("template has no \"`Status` is one of\" sentence: " + path)
	}
	return c, nil
}

// tableAfter is the index of the first table line after the heading line h (-1 when there is none before the next
// heading).
func tableAfter(lines []string, h string) int {
	in := false
	for i, l := range lines {
		switch {
		case strings.TrimSpace(l) == h:
			in = true
		case in && strings.HasPrefix(l, "#"):
			return -1
		case in && strings.HasPrefix(strings.TrimSpace(l), "|"):
			return i
		}
	}
	return -1
}

// cells splits a Markdown table line into trimmed cells.
func cells(l string) []string {
	l = strings.TrimSpace(l)
	l = strings.TrimSuffix(strings.TrimPrefix(l, "|"), "|")
	out := strings.Split(l, "|")
	for i := range out {
		out[i] = strings.TrimSpace(out[i])
	}
	return out
}

// separator reports a table's |---|---| line.
func separator(c []string) bool {
	for _, x := range c {
		if strings.Trim(x, "-: ") != "" {
			return false
		}
	}
	return true
}

// check is every row of the ledger, in line order (the section rows, which have no line, first).
func check(c contract, file string, lines []string) []string {
	var out []string
	add := func(sev string, line int, msg string) {
		at := file
		if line > 0 {
			at += ":" + strconv.Itoa(line)
		}
		out = append(out, sev+"  ledger-check  "+at+"  "+msg)
	}
	first := 0
	for first < len(lines) && strings.TrimSpace(lines[first]) == "" {
		first++
	}
	// Mutation: LC1-title -- a file with another title is not a version ledger
	if first >= len(lines) || !strings.HasPrefix(lines[first], "# VERSION-LEDGER") {
		add("FAIL", first+1, "LC1: the first line is not a `# VERSION-LEDGER` title")
	}
	have := map[string]bool{}
	for _, l := range lines {
		if strings.HasPrefix(l, "## ") {
			have[strings.TrimSpace(l)] = true
		}
	}
	for _, sec := range c.sections {
		// Mutation: LC2-section -- a ledger without its Open items section loses the per-client aggregate
		if !have[sec] {
			add("FAIL", 0, "LC2: no `"+sec+"` section (templates/VERSION-LEDGER.md)")
		}
	}
	classes := map[string]bool{}
	if at := tableAfter(lines, "## Root-cause classes"); at >= 0 {
		for _, l := range lines[at+1:] {
			if !strings.HasPrefix(strings.TrimSpace(l), "|") {
				break
			}
			if cl := cells(l); !separator(cl) {
				classes[cl[0]] = true
			}
		}
	}
	at := tableAfter(lines, "## Versions")
	if at < 0 {
		return out // LC2 named the missing section; with the heading there and no table, there is no row to read
	}
	// Mutation: LC3-header -- a reordered header reads every row's cells into the wrong columns
	if strings.Join(cells(lines[at]), "|") != strings.Join(c.header, "|") {
		add("FAIL", at+1, "LC3: the Versions table header differs from the template's (| "+strings.Join(c.header, " | ")+" |)")
		return out
	}
	col := map[string]int{}
	for i, h := range c.header {
		col[h] = i
	}
	for i := at + 1; i < len(lines) && strings.HasPrefix(strings.TrimSpace(lines[i]), "|"); i++ {
		r := cells(lines[i])
		if separator(r) {
			continue
		}
		n := i + 1
		if len(r) != len(c.header) {
			add("FAIL", n, fmt.Sprintf("LC4: row has %d cells, want %d", len(r), len(c.header)))
			continue
		}
		get := func(h string) string { return r[col[h]] }
		if placeholder.MatchString(lines[i]) {
			// Mutation: LC9-placeholder -- the template's sample row left in place reads as a real version
			add("WARN", n, "LC9: template placeholder left in the row (replace or delete it)")
			continue
		}
		if get("Module") == "" || get("Version") == "" {
			add("FAIL", n, "LC4: empty Module or Version cell")
			continue
		}
		// Mutation: LC5-status -- a free-text status hides a jar that was never deployed
		if st := get("Status"); !contains(c.statuses, st) {
			add("FAIL", n, "LC5: status "+st+" is not one of "+strings.Join(c.statuses, ", "))
		}
		if d := get("Date"); !isoDate.MatchString(d) || !realDate(d) {
			add("FAIL", n, "LC6: date "+d+" is not a YYYY-MM-DD date")
		}
		// Mutation: LC7-class -- a class the ledger does not declare escapes the repeated-class escalation
		if cl := get("Class"); cl != none && !classes[cl] {
			add("FAIL", n, "LC7: class "+cl+" is not in the Root-cause classes table")
		}
		// Mutation: LC8-defect -- a defect with no class never counts toward a repeated class
		if get("Defect found") != none && (get("Root cause") == none || get("Class") == none) {
			add("FAIL", n, "LC8: a defect found with no Root cause and Class")
		}
	}
	return out
}

func contains(xs []string, x string) bool {
	for _, y := range xs {
		if y == x {
			return true
		}
	}
	return false
}

// realDate: d (already YYYY-MM-DD shaped) names a calendar day.
func realDate(d string) bool {
	_, err := time.Parse("2006-01-02", d)
	return err == nil
}

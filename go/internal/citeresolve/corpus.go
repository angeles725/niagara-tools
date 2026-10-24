package citeresolve

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/citeparse"
)

var (
	datedRetro = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}-(.+)\.md$`)
	gapID      = regexp.MustCompile(`^B\d+-G\d+$`)
	numeric    = regexp.MustCompile(`^\d+(?:\.\d+)*[a-z]?$`)
)

type retroResolver struct{ e *Engine }

// Resolve finds retros/<slug>.md or retros/<date>-<slug>.md and, for Δn, the line that defines it.
func (r retroResolver) Resolve(p citeparse.Pointer, _ *Card) Result {
	dir := filepath.Join(r.e.roots.Kit, "retros")
	names, err := r.e.fs.names(dir)
	if err != nil {
		return toolError(err)
	}
	var hit []string
	for _, n := range names {
		if m := datedRetro.FindStringSubmatch(n); n == p.Target+".md" || (m != nil && m[1] == p.Target) {
			hit = append(hit, n)
		}
	}
	if len(hit) == 0 {
		return Result{Status: MissingFile, Detail: "no retros/" + p.Target + ".md or retros/<date>-" + p.Target + ".md"}
	}
	file := filepath.Join(dir, hit[0])
	lines, err := r.e.fs.lines(file)
	if err != nil {
		return toolError(err)
	}
	res := Result{Status: Resolved, File: file, Detail: firstHeading(lines)}
	if len(hit) > 1 {
		res.Detail += fmt.Sprintf(" [%d retros match, used %s]", len(hit), hit[0])
	}
	if p.Delta > 0 {
		i := deltaLine(lines, p.Delta)
		if i < 0 {
			return Result{Status: MissingDelta, File: file, Detail: fmt.Sprintf("%s defines no Δ%d", hit[0], p.Delta)}
		}
		res.Detail = excerpt(lines[i])
	}
	if len(p.Lines) > 0 {
		lr, ok := checkLines(p, lines)
		if !ok {
			return lr
		}
		res.Detail = lr.Detail
	}
	return res
}

type corpusResolver struct{ e *Engine }

// Resolve finds niagara-mental-model-bloque<n>.md and each cited section, row, claim and rule.
func (r corpusResolver) Resolve(p citeparse.Pointer, _ *Card) Result {
	n := strings.TrimPrefix(p.Target, "B")
	file := filepath.Join(r.e.roots.Research, "niagara-mental-model-bloque"+n+".md")
	k, err := kind(file)
	if err != nil {
		return toolError(err)
	}
	if k != "file" {
		return Result{Status: MissingFile, Detail: "no niagara-mental-model-bloque" + n + ".md"}
	}
	lines, err := r.e.fs.lines(file)
	if err != nil {
		return toolError(err)
	}
	var ex []string
	look := func(match func(string) bool) bool {
		i := find(lines, match)
		if i >= 0 {
			ex = append(ex, excerpt(lines[i]))
		}
		return i >= 0
	}
	for _, s := range p.Sections {
		if !look(sectionMatcher(s)) {
			return Result{Status: MissingSection, File: file, Detail: fmt.Sprintf("B%s has no §%s", n, s)}
		}
	}
	for _, rg := range p.Rows {
		for row := rg.From; row <= rg.To; row++ {
			if !look(tableRow(fmt.Sprint(row))) {
				return Result{Status: MissingSection, File: file, Detail: fmt.Sprintf("B%s has no table row %d", n, row)}
			}
		}
	}
	for _, c := range p.Claims {
		row, word := tableRow(c), wordMatch(c)
		if !look(row) && !look(word) {
			return Result{Status: MissingSection, File: file, Detail: fmt.Sprintf("B%s has no %s", n, c)}
		}
	}
	for _, rule := range p.Rules {
		if !look(wordMatch("(?i:rule) " + regexp.QuoteMeta(rule))) {
			return Result{Status: MissingSection, File: file, Detail: fmt.Sprintf("B%s has no rule %s", n, rule)}
		}
	}
	if len(ex) == 0 {
		ex = []string{firstHeading(lines)}
	}
	return Result{Status: Resolved, File: file, Detail: strings.Join(ex, " | ")}
}

// sectionMatcher matches the line of a § token: a heading numbered s (`## §1255.4 —`, `## 1023.1 —`),
// a gap ID anywhere, or any other token as a heading prefix or a literal `§token`.
func sectionMatcher(s string) func(string) bool {
	if gapID.MatchString(s) {
		return wordMatch(regexp.QuoteMeta(s))
	}
	return func(l string) bool {
		h, ok := headingText(l)
		if numeric.MatchString(s) {
			return ok && numbered(h, s)
		}
		return (ok && strings.HasPrefix(h, s)) || strings.Contains(l, "§"+s)
	}
}

// headingText is a Markdown heading's text with any leading `§` dropped.
func headingText(l string) (string, bool) {
	t := strings.TrimLeft(l, "#")
	if t == l || !strings.HasPrefix(t, " ") {
		return "", false
	}
	return strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(t), "§")), true
}

// numbered reports whether heading text h is numbered n: `n` then the end, a space, `)` or a `.` that
// does not open a deeper number.
func numbered(h, n string) bool {
	if !strings.HasPrefix(h, n) {
		return false
	}
	rest := h[len(n):]
	if rest == "" {
		return true
	}
	if rest[0] == '.' {
		return len(rest) == 1 || rest[1] < '0' || rest[1] > '9'
	}
	return rest[0] == ' ' || rest[0] == ')' || rest[0] == ':'
}

func tableRow(id string) func(string) bool {
	re := regexp.MustCompile(`^\|\s*(?:\*\*)?` + regexp.QuoteMeta(id) + `(?:\*\*)?\s*\|`)
	return re.MatchString
}

func wordMatch(pattern string) func(string) bool {
	return regexp.MustCompile(`(?:^|[^\w-])` + pattern + `(?:[^\w-]|$)`).MatchString
}

func find(lines []string, match func(string) bool) int {
	for i, l := range lines {
		if match(l) {
			return i
		}
	}
	return -1
}

func firstHeading(lines []string) string {
	if i := find(lines, func(l string) bool { _, ok := headingText(l); return ok }); i >= 0 {
		return excerpt(lines[i])
	}
	return ""
}

var deltaHeader = regexp.MustCompile(`^\|\s*Δ\s*\|`)

// deltaLine is the line that defines Δn: one opening with Δn (table row, heading, bullet), or row n
// (`| n |`, `| Dn |`) of a table whose header cell is Δ; -1 when none.
func deltaLine(lines []string, n int) int {
	def := regexp.MustCompile(fmt.Sprintf(`^\s*(?:#+\s*|\|\s*|[-*]\s+)?(?:\*\*)?Δ%d(?:[^0-9]|$)`, n))
	row := tableRow(fmt.Sprint(n))
	rowD := tableRow(fmt.Sprintf("D%d", n))
	inTable := false
	for i, l := range lines {
		switch {
		case def.MatchString(l):
			return i
		case deltaHeader.MatchString(l):
			inTable = true
		case !strings.HasPrefix(strings.TrimSpace(l), "|"):
			inTable = false
		case inTable && (row(l) || rowD(l)):
			return i
		}
	}
	return -1
}

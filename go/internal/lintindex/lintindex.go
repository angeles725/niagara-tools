// Package lintindex generates build-n4-module-kit/toolbelt/INDEX.md, the lint index, from the
// header of each toolbelt/lint-*.sh (port of the awk in the former toolbelt/gen-lint-index.sh).
//
// Header contract (the header of every lint-*.sh: its leading run of '#' and blank lines, lintheader.Lines):
//   - line 2 is `# <script-name> — <what it checks>` (also `--` or `-`); continuation lines are
//     joined until a blank '#', a non-comment or keyword line, or line 5 (the description spans
//     header lines 2-5 at most);
//   - every `# Usage:` line and every `# Exit:`/`# Exits:`/`# Exit <code>` line of the header (so a
//     code comment below the first non-comment line never joins) is kept, in order, joined with
//     <br>; a '#' line whose text starts at or right of the
//     value column of the Usage/Exit line above it is a wrapped continuation, unless it is a
//     `Word:` line;
//   - every `[ev: ...]` tag in the header is collected once, in first-seen order.
//
// Bytes are compared as in the C locale: [[:space:]] is ASCII whitespace.
package lintindex

import (
	"regexp"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/lintheader"
)

// Row is one lint of the index.
type Row struct {
	Name  string // script base name, e.g. lint-timers.sh
	Desc  string // description, [ev:] tags removed
	Usage string // Usage lines joined with <br>
	Exit  string // Exit lines joined with <br>
	Ev    string // [ev: ...] tags, space-separated
	Auto  bool   // report-module.sh invokes the lint
}

var (
	evTag        = regexp.MustCompile(`\[ev: [^]]*\]`)
	separator    = regexp.MustCompile(`^[[:space:]]*(—|--|-)[[:space:]]*`)
	blankHash    = regexp.MustCompile(`^#?[[:space:]]*$`)
	keywordLine  = regexp.MustCompile(`^#[[:space:]]*(Usage|Exits?|Scope|Mutation|Row)`)
	hashLead     = regexp.MustCompile(`^#[[:space:]]*`)
	textLine     = regexp.MustCompile(`^#[[:space:]]*[^[:space:]]`)
	wordColon    = regexp.MustCompile(`^#[[:space:]]*[A-Za-z][A-Za-z-]*:`)
	usageLineRe  = regexp.MustCompile(`^#[[:space:]]*Usage:`)
	usageValueRe = regexp.MustCompile(`^#[[:space:]]*Usage:[[:space:]]*`)
	exitLine     = regexp.MustCompile(`^#[[:space:]]*Exits?[:[:space:]]`)
	exitValue    = regexp.MustCompile(`^#[[:space:]]*Exits?:?[[:space:]]*`)
	spaces       = regexp.MustCompile(`[[:space:]]+`)
)

// Extract reads the header of lint script name from its text. It returns the row (Name set, Auto
// false), or a non-empty contract problem when line 2, a Usage line or an Exit line is missing.
func Extract(name, text string) (Row, string) {
	var desc, usage, exitc string
	var ev []string
	seen := map[string]bool{}
	joining := false
	cont, vcol := byte(0), 0

	for i, line := range lintheader.Lines(text) {
		nr := i + 1
		for _, tag := range evTag.FindAllString(line, -1) {
			if !seen[tag] {
				seen[tag] = true
				ev = append(ev, tag)
			}
		}
		if nr == 2 {
			if pre := "# " + name; strings.HasPrefix(line, pre) {
				rest := line[len(pre):]
				if m := separator.FindStringIndex(rest); m != nil {
					desc, joining = rest[m[1]:], true
				}
			}
			continue
		}
		// Description cap: header lines 3-5 only.
		if joining && nr <= 5 {
			if blankHash.MatchString(line) || keywordLine.MatchString(line) {
				joining = false
			} else {
				desc += " " + line[len(hashLead.FindString(line)):]
			}
		}
		// Usage/Exit lines are read from the header only: lintheader.Lines stops at the first code line.
		// Wrapped continuation of the Usage/Exit line above: text at or right of its value column.
		// Mutation: GLI-wrap-exit -- dropping the continuation join truncates a wrapped exit contract
		if cont != 0 {
			if textLine.MatchString(line) && !wordColon.MatchString(line) && len(hashLead.FindString(line))+1 >= vcol {
				t := line[len(hashLead.FindString(line)):]
				if cont == 'u' {
					usage += " " + t
				} else {
					exitc += " " + t
				}
				continue
			}
			cont = 0
		}
		// Mutation: GLI-multi -- keeping only the first Usage/Exit line drops a second contract line
		if usageLineRe.MatchString(line) {
			n := len(usageValueRe.FindString(line))
			vcol, cont = n+1, 'u'
			usage = addLine(usage, line[n:])
			continue
		}
		if exitLine.MatchString(line) {
			n := len(exitValue.FindString(line))
			vcol, cont = n+1, 'x'
			exitc = addLine(exitc, line[n:])
			continue
		}
	}
	switch {
	case desc == "":
		return Row{}, `line 2 is not "# ` + name + ` — <what it checks>"`
	case usage == "":
		return Row{}, `no "# Usage:" line in the header (the leading comment block)`
	case exitc == "":
		return Row{}, `no "# Exit:"/"# Exits:" line in the header (the leading comment block)`
	}
	return Row{Name: name, Desc: clean(evTag.ReplaceAllString(desc, "")), Usage: clean(usage),
		Exit: clean(exitc), Ev: clean(strings.Join(ev, " "))}, ""
}

func addLine(cur, v string) string {
	if cur == "" {
		return v
	}
	return cur + "<br>" + v
}

// clean escapes | for the Markdown table and collapses whitespace runs to one space, trimmed.
func clean(s string) string {
	s = strings.ReplaceAll(s, "|", `\|`)
	s = spaces.ReplaceAllString(s, " ")
	s = strings.TrimPrefix(s, " ")
	return strings.TrimSuffix(s, " ")
}

// InvokedByReport reports whether a non-comment line of report-module.sh (its text) invokes
// "$TOOLBELT/<name>" or ${TOOLBELT}/<name>: the reference, followed by a quote, whitespace or the
// line end, must start a command word -- at the line start or after `$(`, `(`, a backtick, `;`,
// `&`, `|`, `{`, `!` or a command keyword (then, do, else, if, while, until, exec, time, command,
// bash, sh), with an optional opening quote. A comment, a message string (`echo "run $TOOLBELT/x"`),
// a printf argument, an assignment or a longer script name does not count.
func InvokedByReport(report, name string) bool {
	// Mutation: GLI-auto -- a plain substring match marks a lint named only in a comment or a message as Auto yes
	call := regexp.MustCompile(`[$][{]?TOOLBELT[}]?/` + regexp.QuoteMeta(name) + `(["'[:space:]]|$)`)
	comment := regexp.MustCompile(`^[[:space:]]*#`)
	for _, line := range strings.Split(report, "\n") {
		if comment.MatchString(line) {
			continue
		}
		for _, m := range call.FindAllStringIndex(line, -1) {
			if commandPosition(line[:m[0]]) {
				return true
			}
		}
	}
	return false
}

var commandWords = map[string]bool{"then": true, "do": true, "else": true, "if": true, "while": true, "until": true,
	"exec": true, "time": true, "command": true, "bash": true, "sh": true, "!": true}

// commandPosition reports whether a word starting right after prefix is in command position.
func commandPosition(prefix string) bool {
	p := strings.TrimRight(prefix, `"'`)
	if len(p) < len(prefix)-1 {
		return false // more than one quote: not the opening quote of this word
	}
	p = strings.TrimRight(p, " \t")
	if p == "" {
		return true
	}
	switch p[len(p)-1] {
	case '(', '`', ';', '&', '|', '{':
		return true
	}
	if i := strings.LastIndexAny(p, " \t;&|(`"); i >= 0 {
		p = p[i+1:]
	}
	return commandWords[p]
}

// Markdown is the row's line of the index table. Each usage line is its own code span; a usage
// that already holds a backtick is kept as written.
func (r Row) Markdown() string {
	ucell := r.Usage
	if !strings.Contains(ucell, "`") {
		ucell = "`" + strings.ReplaceAll(ucell, "<br>", "`<br>`") + "`"
	}
	auto, ev := "no", r.Ev
	if r.Auto {
		auto = "yes"
	}
	if ev == "" {
		ev = "—"
	}
	return "| `toolbelt/" + r.Name + "` | " + r.Desc + " | " + ucell + " | " + r.Exit + " | " + auto + " | " + ev + " |"
}

const preamble = "<!-- GENERATED by toolbelt/gen-lint-index.sh from each toolbelt/lint-*.sh header. Do not edit by hand. -->\n" +
	"<!-- Regenerate: bash toolbelt/gen-lint-index.sh · CI freshness gate: bash toolbelt/gen-lint-index.sh --check -->\n" +
	"# Toolbelt lint index\n" +
	"\n" +
	"One row per `toolbelt/lint-*.sh`, derived from the script's own header: the description (line 2,\n" +
	"`# <name> — <what it checks>`, up to header line 5), every `Usage:` line, every `Exit:`/`Exits:` line\n" +
	"(wrapped lines joined; several lines separated by a line break) and any `[ev: ...]` tags. **Auto** says\n" +
	"whether `report-module.sh` (run by `build.sh` after every successful build) invokes the lint; a `no`\n" +
	"lint is run by hand on the profiles its usage names.\n" +
	"\n" +
	"To add a lint: write its header to this contract, regenerate, commit both. Never edit this file and\n" +
	"never enumerate or count lints in BUILD-LOOP.md prose. [ev: retro kit-meta-hygiene-2026-10-01 Δ3]\n" +
	"\n" +
	"| Lint | What it checks | Usage | Exit contract | Auto | Evidence |\n" +
	"|---|---|---|---|---|---|"

// Render is the whole index for rows in order, without a trailing newline.
func Render(rows []Row) string {
	var b strings.Builder
	b.WriteString(preamble)
	for _, r := range rows {
		b.WriteString("\n")
		b.WriteString(r.Markdown())
	}
	return b.String()
}

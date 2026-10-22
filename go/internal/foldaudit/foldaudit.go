// Package foldaudit audits folded retros for [ev: retro] citation coverage: every folded row of
// retros/INDEX.md must be cited somewhere in the kit corpus, and (with --deltas-since) every Δk of
// a folded row dated on or after the cutoff must be cited or deferred. It is the Go port of
// build-n4-module-kit/toolbelt/sweep-fold-audit.sh (r8 T17); stdout, stderr and the exit code are
// byte-identical to the bash tool under the C.UTF-8 locale the repository and CI run in, except
// four fail-open cases it fixes: a leading-zero deltas cell is decimal (bash read it as octal, and
// 08/09 skipped the Δ check), an unterminated last INDEX row is audited, and an unreadable INDEX
// exits 3.
// VCS-free by design (kit-links L2).
package foldaudit

import (
	"bytes"
	"os"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"
)

var (
	// The token harvest of the bash tool: grep -ohE over every corpus file.
	tokenRe = regexp.MustCompile(`\[ev: retro ([A-Za-z0-9][A-Za-z0-9._-]*)`)
	deltaRe = regexp.MustCompile(`\[(?:ev|deferred): retro ([A-Za-z0-9][A-Za-z0-9._-]* Δ[0-9]+)\]`)
	// INDEX row filters.
	datedRe = regexp.MustCompile(`[0-9]{4}-[0-9]{2}-[0-9]{2}[^|]*\.md`)
	fileRe  = regexp.MustCompile(`[A-Za-z0-9._-]+\.md`)
	stemRe  = regexp.MustCompile(`^[0-9]{4}-[0-9]{2}-[0-9]{2}-`)
	dateArg = regexp.MustCompile(`^[0-9]{4}-[0-9]{2}-[0-9]{2}$`)
)

func init() { fileRe.Longest() } // grep -o is leftmost-longest

// minToken is the length floor below which a harvested stem token never credits a row; it removes
// template placeholders such as [ev: retro X].
const minToken = 6

// harvest collects the citation tokens of the corpus: stem tokens for the stem check and, when
// enabled, "<stem> Δ<k>" entries from [ev: ...] and [deferred: ...] markers for the Δ check.
type harvest struct {
	tokens   map[string]bool
	deltas   map[string]bool
	withDelt bool
}

func newHarvest(withDeltas bool) *harvest {
	return &harvest{tokens: map[string]bool{}, deltas: map[string]bool{}, withDelt: withDeltas}
}

// add harvests one corpus file. A file holding a NUL byte is binary to grep, which then prints no
// matching text (only a "binary file matches" notice on stderr, discarded by the bash tool).
func (h *harvest) add(b []byte) {
	if bytes.IndexByte(b, 0) >= 0 {
		return
	}
	for _, m := range tokenRe.FindAllSubmatch(b, -1) {
		h.tokens[string(m[1])] = true
	}
	if h.withDelt {
		for _, m := range deltaRe.FindAllSubmatch(b, -1) {
			h.deltas[string(m[1])] = true
		}
	}
}

// sortedTokens is `sort -u` of the stem tokens (bytewise, as under C.UTF-8); nil when empty.
func (h *harvest) sortedTokens() []string {
	var out []string
	for t := range h.tokens {
		out = append(out, t)
	}
	sort.Strings(out)
	return out
}

// harvestCorpus scans every *.md under root except INDEX.md and anything under a retros/ directory,
// reproducing `find "$root" -name '*.md' ! -path '*/retros/*' ! -name INDEX.md`: symlinks are not
// descended (a symlinked root without a trailing slash yields nothing), symlinked files are read
// through, and unreadable entries are skipped silently.
func harvestCorpus(root string, withDeltas bool) *harvest {
	h := newHarvest(withDeltas)
	st, err := os.Lstat(root)
	if err != nil || !st.IsDir() {
		return h
	}
	walk(root, strings.HasSuffix(root, "/"), h)
	return h
}

func walk(dir string, slashed bool, h *harvest) {
	entries, _ := os.ReadDir(dir) // a partial listing is still scanned, as find does
	for _, e := range entries {
		p := dir + "/" + e.Name()
		if slashed {
			p = dir + e.Name()
		}
		if e.IsDir() {
			walk(p, false, h)
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, ".md") || name == "INDEX.md" || strings.Contains(p, "/retros/") {
			continue
		}
		if b, err := os.ReadFile(p); err == nil {
			h.add(b)
		}
	}
}

// datedRow reports whether line holds a YYYY-MM-DD date followed, within one cell, by ".md".
// grep runs in a UTF-8 locale, where [^|] never matches an invalid byte, so each run of valid
// UTF-8 is matched on its own.
func datedRow(line string) bool {
	for _, run := range validRuns(line) {
		if datedRe.MatchString(run) {
			return true
		}
	}
	return false
}

func validRuns(s string) []string {
	var runs []string
	start := 0
	for i := 0; i < len(s); {
		r, n := utf8.DecodeRuneInString(s[i:])
		if r == utf8.RuneError && n == 1 {
			runs = append(runs, s[start:i])
			start = i + 1
		}
		i += n
	}
	return append(runs, s[start:])
}

// rowFile is the row's retro filename: the first [A-Za-z0-9._-]+.md token that is not INDEX.md.
func rowFile(line string) string {
	for _, m := range fileRe.FindAllString(line, -1) {
		if m != "INDEX.md" {
			return m
		}
	}
	return ""
}

// cells splits a row on "|" as awk -F'|' does.
func cells(line string) []string { return strings.Split(line, "|") }

func trimSpaces(s string) string { return strings.Trim(s, " ") }

// rowStatus is the first cell, trimmed of spaces, that reads "pending" or "folded" ("" if none).
func rowStatus(line string) string {
	for _, c := range cells(line) {
		if c = trimSpaces(c); c == "pending" || c == "folded" {
			return c
		}
	}
	return ""
}

// deltasCell is the cell right after the first "folded" cell, with every space removed (tabs kept);
// "" when there is no such cell.
func deltasCell(line string) string {
	cs := cells(line)
	for i, c := range cs {
		if trimSpaces(c) == "folded" {
			if i+1 < len(cs) {
				return strings.ReplaceAll(cs[i+1], " ", "")
			}
			return ""
		}
	}
	return ""
}

// stem is the filename without its leading YYYY-MM-DD- date and trailing .md.
func stem(file string) string {
	return stemRe.ReplaceAllString(strings.TrimSuffix(file, ".md"), "")
}

// citedBy returns the tokens (already sorted) that credit stem: a token T credits it when "-T-"
// is a hyphen-segment-aligned part of "-stem-" and T has at least minToken characters.
func citedBy(stem string, tokens []string) []string {
	var hits []string
	padded := "-" + stem + "-"
	for _, t := range tokens {
		if len(t) >= minToken && strings.Contains(padded, "-"+t+"-") {
			hits = append(hits, t)
		}
	}
	return hits
}

// options is the parsed command line.
type options struct {
	strict             bool
	since, index, root string
}

// parseArgs reads `[--strict] [--deltas-since <YYYY-MM-DD>] <INDEX.md> <kit-root>`; on a usage
// error it returns the exact stderr text of the bash tool.
func parseArgs(args []string) (options, string) {
	var o options
	i := 0
loop:
	for i < len(args) {
		a := args[i]
		switch {
		case a == "--strict":
			o.strict = true
			i++
		case a == "--deltas-since":
			if i+1 >= len(args) || !dateArg.MatchString(args[i+1]) {
				return options{}, "fold-audit: --deltas-since needs a YYYY-MM-DD date\n" + usage + "\n"
			}
			o.since = args[i+1]
			i += 2
		case a == "--":
			i++
			break loop
		case strings.HasPrefix(a, "-"):
			return options{}, "fold-audit: unknown flag: " + a + "\n" + usage + "\n"
		default:
			break loop
		}
	}
	if len(args)-i != 2 {
		return options{}, usage + "\n"
	}
	o.index, o.root = args[i], args[i+1]
	return o, ""
}

package citeparse

import (
	"regexp"
	"strconv"
	"strings"
)

// maxWrap is how many following lines a tag may wrap over before it counts as unterminated.
const maxWrap = 3

var (
	blockList = regexp.MustCompile(`^(B\d+(?:/B\d+)*)\b\s*(.*)$`)
	bareID    = regexp.MustCompile(`^(?:[A-Z]{2,5}-G?\d+|Δ\d+(?:\s+\S+)?)$`)
	deltaTok  = regexp.MustCompile(`^Δ(\d+)$`)
	slugRe    = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)
	claimRe   = regexp.MustCompile(`^C\d+$`)
	kindWords = map[string]bool{"retro": true, "corpus": true, "code": true, "devguide": true, "guide": true,
		"mem": true, "memory": true, "niagara-help": true, "vineflower": true, "javap": true, "course": true,
		"bitácora": true, "bitacora": true, "kit": true}
)

// Card returns every evidence pointer of a card, in line order. Lines inside ``` fences are skipped.
func Card(text string) []Pointer {
	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	fenced := fences(lines)
	sc := &scan{lines: lines, fenced: fenced, masked: append([]string(nil), lines...), aliases: aliases(lines, fenced)}
	var out []Pointer
	for i := range lines {
		if fenced[i] {
			continue
		}
		out = append(out, sc.tags(i)...)
		out = append(out, sc.inline(i)...)
	}
	for i := range out {
		refine(&out[i])
	}
	return out
}

func fences(lines []string) []bool {
	f := make([]bool, len(lines))
	in := false
	for i, l := range lines {
		if strings.HasPrefix(strings.TrimSpace(l), "```") {
			f[i], in = true, !in
			continue
		}
		f[i] = in
	}
	return f
}

type scan struct {
	lines, masked []string
	fenced        []bool
	aliases       map[string]string
}

// tags reads the tags that start on line i and blanks them in masked, so inline scanning skips them.
func (sc *scan) tags(i int) []Pointer {
	var out []Pointer
	for {
		l := sc.masked[i]
		at, deferred := strings.Index(l, "[ev:"), false
		if d := strings.Index(l, "[deferred:"); d >= 0 && (at < 0 || d < at) {
			at, deferred = d, true
		}
		if at < 0 {
			return out
		}
		if end := spanEnd(l, at); end > 0 && !strings.Contains(l[at:end], "]") {
			sc.masked[i] = l[:at] + strings.Repeat(" ", end-at) + l[end:] // literal code, not a tag
			continue
		}
		open := len("[ev:")
		if deferred {
			open = len("[deferred:")
		}
		body, ok := sc.body(i, at+open)
		if !ok {
			sc.masked[i] = l[:at]
			out = append(out, Pointer{Line: i + 1, Kind: Unparsed, Target: strings.TrimSpace(l[at:]), Raw: l[at:],
				Notes: []string{"unterminated tag"}, Deferred: deferred, Unterminated: true})
			continue
		}
		for _, p := range Clauses(body, sc.aliases) {
			p.Line, p.Deferred = i+1, deferred
			out = append(out, p)
		}
	}
}

// spanEnd is where the code span holding column at closes on line l (0 when at is outside a span, or the
// span runs on to the next line).
func spanEnd(l string, at int) int {
	if strings.Count(l[:at], "`")%2 == 0 {
		return 0
	}
	if end := strings.IndexByte(l[at:], '`'); end >= 0 {
		return at + end
	}
	return 0
}

// body returns the tag text after the opener at line i column col up to its `]`, joining wrapped lines
// (up to maxWrap, never across a blank or fenced line), and blanks what it read.
func (sc *scan) body(i, col int) (string, bool) {
	text := sc.masked[i][col:]
	if end := strings.IndexByte(text, ']'); end >= 0 {
		sc.masked[i] = sc.masked[i][:col-1] + strings.Repeat(" ", end+2) + text[end+1:]
		return text[:end], true
	}
	for j := i + 1; j < len(sc.lines) && j <= i+maxWrap; j++ {
		if sc.fenced[j] || strings.TrimSpace(sc.masked[j]) == "" {
			break
		}
		if end := strings.IndexByte(sc.masked[j], ']'); end >= 0 {
			text += " " + strings.TrimSpace(sc.masked[j][:end])
			sc.masked[i] = sc.masked[i][:col-1]
			for k := i + 1; k < j; k++ {
				sc.masked[k] = ""
			}
			sc.masked[j] = strings.Repeat(" ", end+1) + sc.masked[j][end+1:]
			return text, true
		}
		text += " " + strings.TrimSpace(sc.masked[j])
	}
	return "", false
}

// Clauses parses the text of one tag (without `[ev:` and `]`) into pointers (Line unset).
func Clauses(body string, aliases map[string]string) []Pointer {
	var out []Pointer
	for _, c := range splitClauses(body) {
		out = append(out, clause(c, aliases)...)
	}
	return out
}

// splitClauses splits on `;`, then on `, ` where the next part opens with a kind word.
func splitClauses(body string) []string {
	var out []string
	for _, semi := range strings.Split(body, ";") {
		parts := strings.Split(semi, ", ")
		cur := parts[0]
		for _, p := range parts[1:] {
			if kindWords[firstWord(p)] {
				out, cur = append(out, cur), p
			} else {
				cur += ", " + p
			}
		}
		out = append(out, cur)
	}
	var clean []string
	for _, c := range out {
		if c = strings.Trim(strings.TrimSpace(c), "`"); c != "" {
			clean = append(clean, c)
		}
	}
	return clean
}

func firstWord(s string) string {
	f := strings.Fields(s)
	if len(f) == 0 {
		return ""
	}
	return f[0]
}

func clause(c string, aliases map[string]string) []Pointer {
	word, rest, _ := strings.Cut(c, " ")
	rest = strings.TrimSpace(rest)
	switch {
	case word == "retro":
		return retroClause(c, rest)
	case word == "corpus":
		return corpusClause(c, rest)
	case blockList.MatchString(c):
		return corpusClause(c, c)
	case bareID.MatchString(c):
		return []Pointer{{Kind: Bare, Target: c, Raw: c}}
	case (word == "code" || word == "course") && zipEntry.MatchString(rest):
		return archiveClause(c, rest)
	case word == "course":
		return courseClause(c, rest)
	case word == "bitácora" || word == "bitacora":
		return bitacoraClause(c, rest)
	case word == "kit" && rest != "":
		return kitClause(c, rest, aliases)
	case word == "external" && firstWord(rest) == "retro":
		return externalRetro(c, strings.TrimSpace(strings.TrimPrefix(rest, "retro")))
	case strings.HasSuffix(c, " memory") && len(strings.Fields(c)) == 2:
		return []Pointer{{Kind: Mem, Target: strings.TrimSuffix(word, ".md"), Raw: c}}
	}
	if ps, ok := moduleRef(c, word, rest, aliases); ok {
		return ps
	}
	return sourceClause(c, word, rest, aliases)
}

func unparsed(c, why string) []Pointer {
	return []Pointer{{Kind: Unparsed, Target: c, Raw: c, Notes: []string{why}}}
}

// retroClause reads `retro <slug>[ Δn[, Δm]…][ note…]`.
func retroClause(c, rest string) []Pointer {
	f := strings.Fields(rest)
	var lines []Range
	if len(f) > 0 {
		f[0], lines = splitLines(f[0])
	}
	if len(f) == 0 || !slugRe.MatchString(strings.TrimSuffix(f[0], ".md")) {
		if len(f) > 0 && bareID.MatchString(rest) {
			return []Pointer{{Kind: Bare, Target: rest, Raw: c}}
		}
		return unparsed(c, "retro slug not readable")
	}
	base := Pointer{Kind: Retro, Target: strings.TrimSuffix(f[0], ".md"), Lines: lines, Raw: c}
	var deltas []int
	for _, w := range strings.FieldsFunc(strings.Join(f[1:], " "), func(r rune) bool { return r == ' ' || r == ',' || r == '/' }) {
		if m := deltaTok.FindStringSubmatch(w); m != nil {
			n, _ := strconv.Atoi(m[1])
			deltas = append(deltas, n)
		} else {
			base.Notes = append(base.Notes, w)
		}
	}
	if len(deltas) == 0 {
		return []Pointer{base}
	}
	out := make([]Pointer, len(deltas))
	for i, d := range deltas {
		out[i] = base
		out[i].Delta = d
	}
	return out
}

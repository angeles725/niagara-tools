package cisteps

import (
	"regexp"
	"strings"
)

// The here-document lint (retro coordination-stretch C8). bats preprocesses a .bats file line by line
// and reads every line that starts with the test keyword as a case of the file, inside a here-document
// too: a fixture written through `cat <<'SH'` with such lines adds cases to the file that writes it,
// and bats 1.10 (the CI pin) refuses the file with `Duplicate test name(s)` while bats 1.14 accepts it.
// Fix: spell the fixture's keyword as a placeholder and turn it back with sed
// (`sed 's/^%test /@test /' > f <<'SH'`).

var (
	// heredocOpen is one `<<` or `<<-` operator and its delimiter word, quoted or not; a herestring
	// (`<<<`) and an arithmetic shift (`1<<2`, no word) are filtered out by the caller and the regexp.
	heredocOpen = regexp.MustCompile(`<<(-?)[ \t]*(?:'([^']+)'|"([^"]+)"|\\?([A-Za-z_][A-Za-z0-9_]*))`)
	testKeyword = regexp.MustCompile(`^[ \t]*@test[ \t]`)
)

// pendingDoc is a here-document whose body has not started or not ended yet.
type pendingDoc struct {
	delim     string
	stripTabs bool
}

// HeredocTestLines returns the 1-based numbers of the here-document body lines that start with the
// test keyword. A comment line opens no here-document; several operators on one line are read in order.
func HeredocTestLines(data []byte) []int {
	var found []int
	lines := strings.Split(string(data), "\n")
	for i, body := range heredocBody(lines) {
		if body && testKeyword.MatchString(lines[i]) {
			found = append(found, i+1)
		}
	}
	return found
}

// heredocBody marks each line that is here-document text (a body line or its closing delimiter), not a
// command: the source scan of sourced.go skips them too.
func heredocBody(lines []string) []bool {
	body := make([]bool, len(lines))
	var queue []pendingDoc
	for i, line := range lines {
		if len(queue) > 0 {
			body[i] = true
			doc := queue[0]
			end := line
			if doc.stripTabs {
				end = strings.TrimLeft(end, "\t")
			}
			if end == doc.delim {
				queue = queue[1:]
			}
			continue
		}
		queue = append(queue, openedOn(line)...)
	}
	return body
}

// quotedAt reports whether byte i of a line sits inside a '…' or "…" string opened on that line
// (a backslash escapes the next byte outside single quotes).
func quotedAt(line string, i int) bool {
	var quote byte
	for j := 0; j < i; j++ {
		c := line[j]
		switch {
		case quote == '\'':
			if c == '\'' {
				quote = 0
			}
		case c == '\\':
			j++
		case quote == '"':
			if c == '"' {
				quote = 0
			}
		case c == '\'' || c == '"':
			quote = c
		}
	}
	return quote != 0
}

// commentAt is where an unquoted `#` that starts a word begins a comment (len(line) when none).
func commentAt(line string) int {
	for i := 0; i < len(line); i++ {
		if line[i] == '#' && (i == 0 || line[i-1] == ' ' || line[i-1] == '\t') && !quotedAt(line, i) {
			return i
		}
	}
	return len(line)
}

// openedOn lists the here-documents a command line opens.
func openedOn(line string) []pendingDoc {
	line = line[:commentAt(line)]
	var docs []pendingDoc
	for _, m := range heredocOpen.FindAllStringSubmatchIndex(line, -1) {
		if m[0] > 0 && line[m[0]-1] == '<' { // the tail of a `<<<` herestring
			continue
		}
		if m[0]+2 < len(line) && line[m[0]+2] == '<' {
			continue
		}
		if quotedAt(line, m[0]) { // `printf 'cat <<"TAP"\n'` writes a here-document, it opens none here
			continue
		}
		delim := ""
		for g := 2; g <= 4; g++ { // group 2 '…', 3 "…", 4 a bare (or \-escaped) word
			if m[2*g] >= 0 {
				delim = line[m[2*g]:m[2*g+1]]
			}
		}
		docs = append(docs, pendingDoc{delim: delim, stripTabs: m[3] > m[2]})
	}
	return docs
}

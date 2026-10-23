// Package guardpins is the guard-pin meta-check over the kit's own lints (port of the former
// toolbelt/lint-guard-pins.sh, C11 T4, B833): every toolbelt/lint-*.sh declares at least one
// header line `# Mutation: <fixture-id> -- <what it flips>` in its header (lintheader), and each declared id
// resolves to an `@test "<id>:"` or `@test "<id> ` entry in tests/*.bats.
package guardpins

import (
	"regexp"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/lintheader"
)

// Decl is one `# Mutation:` declaration: its 1-based line and fixture id.
type Decl struct {
	Line int
	ID   string
}

// declaration is the C11 §D4 grammar: column 0, ASCII `--`, an id starting with a letter and
// some text after the `--`.
var declaration = regexp.MustCompile(`^# Mutation: ([A-Za-z][A-Za-z0-9_-]*) -- .+$`)

// Declarations returns the mutation declarations of text, in line order: those in the header
// (lintheader.Lines, which replaced the former fixed 60-line window, D4c), and apart from them
// every declaration line below the header, which declares nothing and is reported, never skipped.
func Declarations(text string) (header, stray []Decl) {
	n := len(lintheader.Lines(text))
	for i, line := range lines(text) {
		if m := declaration.FindStringSubmatch(line); m != nil {
			if d := (Decl{i + 1, m[1]}); i < n {
				header = append(header, d)
			} else {
				stray = append(stray, d)
			}
		}
	}
	return header, stray
}

// lines splits text into lines; a final newline ends the last line rather than starting one.
func lines(text string) []string {
	if text == "" {
		return nil
	}
	return strings.Split(strings.TrimSuffix(text, "\n"), "\n")
}

// BatsFile is one tests/*.bats file: its path as printed and its text.
type BatsFile struct {
	Path, Text string
}

// Hit is where a fixture id is defined: the bats path and its 1-based line.
type Hit struct {
	Path string
	Line int
}

// IndexFixtures maps each fixture id to its first definition across files (in the given order,
// then line order). A line `@test "<rest>` defines the id that is the part of <rest> before its
// first ':' or ' ' -- exactly the ids the grep `^@test "<id>[: ]` would find, since an id holds
// neither character.
func IndexFixtures(files []BatsFile) map[string]Hit {
	idx := map[string]Hit{}
	const prefix = `@test "`
	for _, f := range files {
		for i, line := range lines(f.Text) {
			if !strings.HasPrefix(line, prefix) {
				continue
			}
			rest := line[len(prefix):]
			end := strings.IndexAny(rest, ": ")
			if end < 0 {
				continue
			}
			if id := rest[:end]; id != "" {
				if _, seen := idx[id]; !seen {
					idx[id] = Hit{f.Path, i + 1}
				}
			}
		}
	}
	return idx
}

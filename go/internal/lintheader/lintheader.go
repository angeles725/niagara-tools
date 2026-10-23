// Package lintheader defines the header of a kit lint script (build-n4-module-kit/toolbelt/lint-*.sh):
// its leading run of lines that start with '#' or are blank, the shebang included. The first other
// line (code, an indented comment) ends it. There is no line cap: the former fixed lines 1-60
// window silently dropped header lines of a longer header.
//
// Shared by guardpins (`# Mutation:` declarations) and lintindex (description, Usage, Exit, [ev:] tags).
package lintheader

import "strings"

// Lines returns the header lines of text, without their newlines (a CR of a CRLF line is kept).
// A final newline ends the last line rather than starting one; an unterminated last line is read.
func Lines(text string) []string {
	var out []string
	for text != "" {
		line, rest, _ := strings.Cut(text, "\n")
		// Mutation: GLI-exit-block -- reading on past the first code line joins a code comment's
		// `# Exit` or `# Mutation:` line to the header
		if !isHeader(line) {
			break
		}
		out = append(out, line)
		text = rest
	}
	return out
}

// isHeader reports whether line belongs to a header: blank (spaces, tabs or a lone CR) or
// starting with '#' in column 0.
func isHeader(line string) bool {
	t := strings.TrimRight(line, " \t\r")
	return t == "" || t[0] == '#'
}

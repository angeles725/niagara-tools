// Package lintheader defines the header of a kit lint script (build-n4-module-kit/toolbelt/lint-*.sh):
// its leading run of lines that start with '#', the shebang included. The first line that does not
// start with '#' (code, a blank line, an indented comment) ends it. There is no line cap: the former
// fixed lines 1-60 window silently dropped header lines of a longer header.
//
// Shared by guardpins (`# Mutation:` declarations) and lintindex (description, Usage, Exit, [ev:] tags).
package lintheader

import "strings"

// Lines returns the header lines of text, without their newlines (a CR of a CRLF line is kept).
// A final newline ends the last line rather than starting one; an unterminated last line is read.
func Lines(text string) []string {
	var out []string
	for len(text) > 0 && text[0] == '#' {
		line, rest, _ := strings.Cut(text, "\n")
		out = append(out, line)
		text = rest
	}
	return out
}

package envfile

import "strings"

// The reasons a line is refused. None carries text of the line.
const (
	reasonNotAssignment = "not a KEY=value assignment (a command, a function or another shell construct)"
	reasonExport        = "export needs KEY=value"
	reasonExpansion     = "a $ expansion or command substitution is not supported: write the value out"
	reasonBackquote     = "a `command` substitution is not supported: write the value out"
	reasonOperator      = "a shell operator (; & | < > ( ) { }) is not supported outside quotes"
	reasonTilde         = "a ~ expansion is not supported: write the path out"
	reasonQuote         = "a quote is not closed on its line"
	reasonContinuation  = "a line continuation (a final backslash) is not supported"
)

const (
	exportWord = "export"
	// operators are the bytes that end or group a shell command outside quotes.
	operators = ";&|<>(){}"
	// dquoteEscapes are the bytes a backslash escapes inside double quotes; before any other it stays.
	dquoteEscapes = "$`\"\\"
)

// word is one shell word of a line. key is the bare text before its first bare "="; it is empty when the
// word has no such "=" or when a quote or an escape touches that part (bash then runs a command).
type word struct {
	key, value string
	text       string // the whole word with its quoting removed
	quoted     bool   // a quote or an escape appears somewhere in the word
}

// isBare reports the word spelled exactly s, with no quoting.
func (w word) isBare(s string) bool { return !w.quoted && w.text == s }

// splitWords splits a line into its words the way bash does for the constructs this package supports.
// reason says why the line is refused; "" when it is not.
func splitWords(line string) (words []word, reason string) {
	for i := 0; i < len(line); {
		switch c := line[i]; {
		case c == ' ' || c == '\t':
			i++
		case c == '#':
			return words, "" // a comment runs to the end of the line
		default:
			var w word
			if w, i, reason = readWord(line, i); reason != "" {
				return nil, reason
			}
			words = append(words, w)
		}
	}
	return words, ""
}

// readWord reads the word that starts at line[i] and returns it with the index after it.
func readWord(line string, i int) (w word, next int, reason string) {
	var text strings.Builder
	eq := -1           // index in text of the first bare "="
	keyBare := true    // nothing before that "=" was quoted or escaped
	tildeHere := false // a bare "~" at this point would be expanded by bash
	for ; i < len(line) && line[i] != ' ' && line[i] != '\t'; i++ {
		c := line[i]
		expandsTilde := tildeHere
		tildeHere = false
		switch {
		case c == '\'' || c == '"':
			end, part, why := readQuoted(line, i)
			if why != "" {
				return w, i, why
			}
			text.WriteString(part)
			i, w.quoted = end, true
		case c == '\\':
			if i+1 == len(line) {
				return w, i, reasonContinuation
			}
			i++
			text.WriteByte(line[i])
			w.quoted = true
		case c == '$':
			return w, i, reasonExpansion
		case c == '`':
			return w, i, reasonBackquote
		case strings.IndexByte(operators, c) >= 0:
			return w, i, reasonOperator
		case c == '~' && expandsTilde:
			return w, i, reasonTilde
		default:
			if c == '=' && eq < 0 {
				eq, keyBare = text.Len(), !w.quoted
			}
			tildeHere = eq >= 0 && (c == '=' && eq == text.Len() || c == ':')
			text.WriteByte(c)
		}
	}
	w.text = text.String()
	if eq >= 0 && keyBare {
		w.key, w.value = w.text[:eq], w.text[eq+1:]
	}
	return w, i, ""
}

// readQuoted reads the quoted part that opens at line[i] and returns the index of its closing quote and
// its text.
func readQuoted(line string, i int) (end int, text, reason string) {
	quote := line[i]
	var part strings.Builder
	for j := i + 1; j < len(line); j++ {
		c := line[j]
		switch {
		case c == quote:
			return j, part.String(), ""
		case quote == '\'':
			part.WriteByte(c)
		case c == '$':
			return 0, "", reasonExpansion
		case c == '`':
			return 0, "", reasonBackquote
		case c == '\\' && j+1 < len(line) && strings.IndexByte(dquoteEscapes, line[j+1]) >= 0:
			j++
			part.WriteByte(line[j])
		default:
			part.WriteByte(c)
		}
	}
	return 0, "", reasonQuote
}

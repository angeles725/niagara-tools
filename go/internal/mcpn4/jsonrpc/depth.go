package jsonrpc

import "errors"

// MaxNesting is server.MAX_NESTING: the deepest request nesting (arrays and objects, the outer
// object counted) that is parsed; a deeper line gets the parse error reply, as in Python.
const MaxNesting = 1000

var errTooDeep = errors.New("nesting deeper than MaxNesting")

// NestsDeeper reports whether the JSON text opens more than limit arrays/objects at once,
// skipping string contents (with their escapes). On a valid document that is its nesting
// depth; on an invalid one either answer leads to the same parse error reply.
func NestsDeeper(text []byte, limit int) bool {
	depth, inString := 0, false
	for i := 0; i < len(text); i++ {
		c := text[i]
		switch {
		case inString && c == '\\':
			i++
		case c == '"':
			inString = !inString
		case inString:
		case c == '[' || c == '{':
			if depth++; depth > limit {
				return true
			}
		case c == ']' || c == '}':
			depth--
		}
	}
	return false
}

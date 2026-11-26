package buildstatefields

import (
	"regexp"
	"strings"
)

// signerEnv is what one build-state.v1 envelope says about expected_signer: its module, the value of every
// `expected_signer:` field at column 0 (comment removed, blanks trimmed) and every line that is the key in some
// other shape (strays).
type signerEnv struct {
	module string
	values []string
	strays []string
}

// strayKey matches a line that reads as the expected_signer key but is not the field form (the key at column
// 0, the colon right after it): blanks before the key or before the colon. Other keys have no such rule.
var strayKey = regexp.MustCompile(`^[ \t]*` + SignerKey + `[ \t]*:`)

// signerEnvelopes reads every envelope of a text, in order. Like the other envelope readers: keys start at
// column 0 and a `# comment` ends the value; a byte-order mark that opens the text is dropped and a CR ends a line.
func signerEnvelopes(text string) (found []signerEnv) {
	text = strings.ReplaceAll(strings.TrimPrefix(text, "\xef\xbb\xbf"), "\r", "\n")
	var cur *signerEnv
	for _, line := range Lines(text) {
		switch {
		case strings.HasPrefix(line, OpenMarker):
			found = append(found, signerEnv{})
			cur = &found[len(found)-1]
		case strings.HasPrefix(line, CloseMarker):
			cur = nil
		case cur != nil:
			key, val, ok := strings.Cut(line, ":")
			if i := strings.IndexByte(val, '#'); i >= 0 {
				val = val[:i]
			}
			switch {
			case ok && key == "module" && cur.module == "":
				cur.module = trimSpace(val)
			case ok && key == SignerKey:
				cur.values = append(cur.values, trimSpace(val))
			case strayKey.MatchString(line):
				cur.strays = append(cur.strays, line)
			}
		}
	}
	return found
}

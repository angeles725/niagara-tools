package prepush

import "strings"

// refLine is one line of the pre-push stdin.
type refLine struct {
	localRef, localSha, remoteRef, remoteSha string
}

// fieldSep is what separates the fields of a ref line (the shell's IFS without the newline).
const fieldSep = " \t"

// parseRefLine splits a ref line the way the shell's `read -r lref lsha rref rsha` did: three
// whitespace-separated fields, then the rest of the line, trimmed, as the fourth. A missing field is "".
func parseRefLine(line string) refLine {
	rest := line
	next := func() string {
		rest = strings.TrimLeft(rest, fieldSep)
		field := rest
		if i := strings.IndexAny(rest, fieldSep); i >= 0 {
			field = rest[:i]
		}
		rest = rest[len(field):]
		return field
	}
	return refLine{localRef: next(), localSha: next(), remoteRef: next(), remoteSha: strings.Trim(rest, fieldSep)}
}

// allZeros reports whether sha is made of zeros only; the empty string is.
func allZeros(sha string) bool { return strings.Trim(sha, "0") == "" }

// isDeletion reports whether a local sha is the all-zeros id of a branch deletion: nothing is pushed.
func isDeletion(localSha string) bool { return localSha != "" && allZeros(localSha) }

// isNewRemoteRef reports whether the remote sha says the ref does not exist yet on the remote.
func isNewRemoteRef(remoteSha string) bool { return allZeros(remoteSha) }

// orUnknown is the ref name printed in a refusal ("?" when the line had none).
func orUnknown(ref string) string {
	if ref == "" {
		return "?"
	}
	return ref
}

package checkclientsource

import (
	"bytes"
	"fmt"
	"strings"
	"unicode/utf16"
)

// Why a BUILD-STATE text cannot be read as what it declares (stderr, after the tool prefix; %s is the path).
const (
	utf16Odd    = "%s starts with a UTF-16 byte-order mark but holds an odd number of bytes: save it as UTF-8"
	nulBytes    = "%s holds NUL bytes (UTF-16 without a byte-order mark?): save it as UTF-8"
	lineOpenBOM = "%s:%d: a byte-order mark opens this line (only the file's first line may carry one): remove it"
)

// decodeBuildState is the text of a BUILD-STATE file's bytes: UTF-16 with its byte-order mark (either byte
// order) is decoded, anything else is read as UTF-8 (SourceOfTruth drops a UTF-8 mark that opens it). fault
// is why the bytes cannot be read as text, "" when they can: UTF-16 with an odd byte count, or NUL bytes,
// which UTF-8 text never holds (UTF-16 saved without its mark). Read as bytes either would name no module
// and the check would WARN "undeclared", exit 0. [ev: retro coordination-stretch-deltas K9]
func decodeBuildState(path string, b []byte) (text, fault string) {
	var big bool
	switch {
	case bytes.HasPrefix(b, []byte{0xff, 0xfe}):
	case bytes.HasPrefix(b, []byte{0xfe, 0xff}):
		big = true
	default:
		if bytes.IndexByte(b, 0) >= 0 {
			return "", fmt.Sprintf(nulBytes, path)
		}
		return string(b), ""
	}
	if len(b)%2 != 0 {
		return "", fmt.Sprintf(utf16Odd, path)
	}
	units := make([]uint16, 0, len(b)/2-1)
	for i := 2; i < len(b); i += 2 {
		if big {
			units = append(units, uint16(b[i])<<8|uint16(b[i+1]))
		} else {
			units = append(units, uint16(b[i+1])<<8|uint16(b[i]))
		}
	}
	// Mutation: CS-utf16 -- reading UTF-16 as bytes names no module (WARN undeclared, exit 0)
	return string(utf16.Decode(units)), ""
}

// strayBOM is the fault of a UTF-8 byte-order mark that opens a line other than the first (two files an
// editor joined): it would hide the key behind it, so the line would read as no field. "" when there is none.
func strayBOM(path, text string) string {
	// a line ends at an LF, a CR or a CRLF, as SourceOfTruth splits it; the number counts each once
	lines := strings.Split(strings.ReplaceAll(strings.ReplaceAll(text, "\r\n", "\n"), "\r", "\n"), "\n")
	for i, line := range lines {
		// Mutation: CS-mid-bom -- reading a line a mark opens as no key reports "undeclared", exit 0
		if i > 0 && strings.HasPrefix(line, bom) {
			return fmt.Sprintf(lineOpenBOM, path, i+1)
		}
	}
	return ""
}

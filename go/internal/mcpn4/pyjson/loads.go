package pyjson

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"
)

// ValueError is a Python ValueError whose str() is Msg: what json.loads(data.decode()) or
// bytes.decode() raises, worded as CPython words it.
type ValueError struct{ Msg string }

func (e *ValueError) Error() string { return e.Msg }

// Loads is json.loads(data.decode()): the bytes are decoded as strict UTF-8 first, and a text
// starting with a byte order mark is refused, as json.loads refuses a str starting with U+FEFF.
// Every failure is a *ValueError carrying CPython's message, position suffix included.
func Loads(data []byte) (any, error) {
	if _, err := DecodeUTF8(data, false); err != nil {
		return nil, err
	}
	if bytes.HasPrefix(data, []byte("\ufeff")) {
		return nil, &ValueError{"Unexpected UTF-8 BOM (decode using utf-8-sig): " + position(data, 0)}
	}
	v, err := Unmarshal(data)
	var se *SyntaxError
	if errors.As(err, &se) {
		return nil, &ValueError{se.PyText(data)}
	}
	return v, err
}

// PyText is str() of the JSONDecodeError CPython raises on doc (valid UTF-8): the message, then
// ": line L column C (char P)" with P counted in code points.
func (e *SyntaxError) PyText(doc []byte) string {
	if e.Plain {
		return e.Msg
	}
	return e.Msg + ": " + position(doc, e.Offset)
}

// position is JSONDecodeError's "line L column C (char P)" for byte offset off of doc.
func position(doc []byte, off int) string {
	head := doc[:off]
	pos := utf8.RuneCount(head)
	line := bytes.Count(head, []byte("\n")) + 1
	col := pos + 1
	if nl := bytes.LastIndexByte(head, '\n'); nl >= 0 {
		col = pos - utf8.RuneCount(head[:nl])
	}
	return fmt.Sprintf("line %d column %d (char %d)", line, col, pos)
}

// DecodeUTF8 is bytes.decode("utf-8", errors): with replace false a malformed sequence fails
// with CPython's UnicodeDecodeError text; with replace true each maximal invalid subpart
// becomes one U+FFFD, as CPython (and the Unicode recommendation) count them.
func DecodeUTF8(b []byte, replace bool) (string, error) {
	var out strings.Builder
	for i := 0; i < len(b); {
		if b[i] < utf8.RuneSelf {
			out.WriteByte(b[i])
			i++
			continue
		}
		n, end := badSequence(b, i)
		if n == 0 {
			r, size := utf8.DecodeRune(b[i:])
			out.WriteRune(r)
			i += size
			continue
		}
		if replace {
			out.WriteRune(utf8.RuneError)
			i += n
			continue
		}
		return "", decodeError(b, i, n, end)
	}
	return out.String(), nil
}

// decodeError is the UnicodeDecodeError text of the n-byte invalid subpart at b[i] (end: it is
// cut short by the end of b).
func decodeError(b []byte, i, n int, end bool) *ValueError {
	reason := "invalid start byte"
	switch {
	case end:
		reason = "unexpected end of data"
	case n > 1 || b[i] >= 0xc2 && b[i] <= 0xf4:
		reason = "invalid continuation byte"
	}
	where := fmt.Sprintf("byte 0x%02x in position %d", b[i], i)
	if n > 1 {
		where = fmt.Sprintf("bytes in position %d-%d", i, i+n-1)
	}
	return &ValueError{"'utf-8' codec can't decode " + where + ": " + reason}
}

// badSequence returns the length of the maximal invalid subpart at b[i] (0 when a valid
// sequence starts there) and whether it is cut short by the end of b.
func badSequence(b []byte, i int) (int, bool) {
	c := b[i]
	var need int
	lo, hi := byte(0x80), byte(0xbf) // the allowed range of the second byte
	switch {
	case c >= 0xc2 && c <= 0xdf:
		need = 1
	case c >= 0xe0 && c <= 0xef:
		need = 2
		if c == 0xe0 {
			lo = 0xa0
		} else if c == 0xed {
			hi = 0x9f
		}
	case c >= 0xf0 && c <= 0xf4:
		need = 3
		if c == 0xf0 {
			lo = 0x90
		} else if c == 0xf4 {
			hi = 0x8f
		}
	default:
		return 1, false
	}
	for k := 1; k <= need; k++ {
		if i+k >= len(b) {
			return k, true
		}
		if x := b[i+k]; x < lo || x > hi {
			return k, false
		}
		lo, hi = 0x80, 0xbf
	}
	return 0, false
}

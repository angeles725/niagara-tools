package pyjson

import (
	"strings"
	"unicode/utf8"
)

// textChunk is TextIOWrapper._CHUNK_SIZE: a text file is decoded 8192 bytes at a time.
const textChunk = 8192

// ReadLines is open(path).readlines() in UTF-8 mode on the file's bytes data: strict UTF-8
// decoded chunk by chunk, so a failure is the UnicodeDecodeError of the chunk where it occurs
// (its positions count from that chunk, after any bytes of a character the previous chunk cut),
// then universal newlines ("\r\n" and "\r" read as "\n"), each line keeping its "\n".
func ReadLines(data []byte) ([]string, error) {
	var text strings.Builder
	var pending []byte
	for off := 0; off < len(data); off += textChunk {
		buf := append(pending, data[off:min(off+textChunk, len(data))]...)
		pending = nil
		for i := 0; i < len(buf); {
			if buf[i] < utf8.RuneSelf {
				text.WriteByte(buf[i])
				i++
				continue
			}
			n, end := badSequence(buf, i)
			// CPython keeps a cut 0xED lead pending whatever its continuation byte (a surrogate
			// is refused only once the character is complete).
			end = end || n == 1 && buf[i] == 0xed && i+2 == len(buf) && buf[i+1]&0xc0 == 0x80
			switch {
			case n == 0:
				_, size := utf8.DecodeRune(buf[i:])
				text.Write(buf[i : i+size])
				i += size
			case end: // a character the next chunk may complete
				pending = append([]byte(nil), buf[i:]...)
				i = len(buf)
			default:
				return nil, decodeError(buf, i, n, false)
			}
		}
	}
	if len(pending) > 0 { // the final decode of what the last chunk left
		n, end := badSequence(pending, 0)
		return nil, decodeError(pending, 0, n, end)
	}
	s := strings.ReplaceAll(text.String(), "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	var lines []string
	for s != "" {
		line, rest, found := strings.Cut(s, "\n")
		if found {
			line += "\n"
		}
		lines = append(lines, line)
		s = rest
	}
	return lines, nil
}

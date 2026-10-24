package xmlsrc

import (
	"io"
	"regexp"
	"strings"
	"unicode/utf8"
)

// declEncoding finds the encoding a leading XML declaration names.
var declEncoding = regexp.MustCompile(`^(?:\xef\xbb\xbf)?<\?xml\s[^?>]*?encoding\s*=\s*["']([^"']*)["']`)

// Niagara writes UTF-8, ISO-8859-1 (Tridium's doc and module XML) and US-ASCII; other labels are
// refused rather than guessed.
var (
	latin1Labels = []string{"iso-8859-1", "iso8859-1", "iso_8859-1", "latin1", "latin-1", "l1"}
	asciiLabels  = []string{"us-ascii", "ascii"}
)

func hasLabel(labels []string, label string) bool {
	for _, l := range labels {
		if strings.EqualFold(l, label) {
			return true
		}
	}
	return false
}

// toUTF8 returns data as UTF-8 according to its declared encoding. Positions are then counted in
// the UTF-8 text: same lines, and Col counts its bytes (a Latin-1 'é' is 2).
func (p *parser) toUTF8(data []byte) ([]byte, error) {
	m := declEncoding.FindSubmatch(data)
	if m == nil || strings.EqualFold(string(m[1]), "utf-8") {
		return data, nil
	}
	label := string(m[1])
	switch {
	case hasLabel(latin1Labels, label):
		out := make([]byte, 0, len(data)+len(data)/8)
		for _, b := range data {
			out = utf8.AppendRune(out, rune(b))
		}
		return out, nil
	case hasLabel(asciiLabels, label):
		for i, b := range data {
			if b >= 0x80 {
				return nil, p.fail(int64(i), "byte 0x%02x is not US-ASCII", b)
			}
		}
		return data, nil
	}
	return nil, p.fail(0, "encoding %q is not supported (UTF-8, ISO-8859-1 and US-ASCII are)", label)
}

// sameReader is the Decoder's CharsetReader: toUTF8 has already decoded the text.
func sameReader(_ string, r io.Reader) (io.Reader, error) { return r, nil }

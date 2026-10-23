package xmlsrc

import (
	"bytes"
	"encoding/xml"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

// The checks below close the gaps between encoding/xml and expat that the parity test found;
// each works on the raw bytes of one token, so the document is still read once.

var bom = []byte("\xef\xbb\xbf")

// xmlDecl is the XMLDecl production: version, then optional encoding and standalone, in order.
var xmlDecl = regexp.MustCompile(`^<\?xml\s+version\s*=\s*("1\.[0-9]+"|'1\.[0-9]+')` +
	`(\s+encoding\s*=\s*("[A-Za-z][A-Za-z0-9._-]*"|'[A-Za-z][A-Za-z0-9._-]*'))?` +
	`(\s+standalone\s*=\s*("(yes|no)"|'(yes|no)'))?\s*\?>$`)

func isWhite(b byte) bool { return b == ' ' || b == '\t' || b == '\r' || b == '\n' }

// procInst checks one <?target ...?> whose raw bytes start at off.
func (p *parser) procInst(t xml.ProcInst, raw []byte, off, begin int64) error {
	after := len("<?") + len(t.Target)
	if after < len(raw) && !isWhite(raw[after]) && string(raw[after:]) != "?>" {
		return p.fail(off+int64(after), "processing-instruction target %q not followed by white space", t.Target)
	}
	if t.Target == "xml" {
		if off != begin {
			return p.fail(off, "XML declaration not at the start of the document")
		}
		if !xmlDecl.Match(raw) {
			return p.fail(off, "XML declaration not well-formed")
		}
	} else if strings.EqualFold(t.Target, "xml") {
		return p.fail(off, "reserved processing-instruction target %q", t.Target)
	}
	if !validChars(t.Inst) {
		return p.fail(off, "invalid character in processing instruction")
	}
	return nil
}

// validChars reports whether b is UTF-8 made only of XML characters; encoding/xml checks this in
// text and attribute values but not in comments or processing instructions.
func validChars(b []byte) bool {
	for i := 0; i < len(b); {
		r, n := utf8.DecodeRune(b[i:])
		if (r == utf8.RuneError && n == 1) || !isXMLChar(r) {
			return false
		}
		i += n
	}
	return true
}

// isXMLChar is the Char production of XML 1.0.
func isXMLChar(r rune) bool {
	return r == 0x09 || r == 0x0A || r == 0x0D || r >= 0x20 && r <= 0xD7FF ||
		r >= 0xE000 && r <= 0xFFFD || r >= 0x10000 && r <= 0x10FFFF
}

// attrSpacing returns the index in a raw start tag of a byte that follows an attribute's closing
// quote without white space, '/' or '>' (expat: invalid token), or -1.
func attrSpacing(raw []byte) int {
	var quote byte
	for i, b := range raw {
		switch {
		case quote != 0:
			if b == quote {
				quote = 0
				if i+1 < len(raw) && !isWhite(raw[i+1]) && raw[i+1] != '/' && raw[i+1] != '>' {
					return i + 1
				}
			}
		case b == '"' || b == '\'':
			quote = b
		}
	}
	return -1
}

// surrogateRef returns the index of a character reference to a surrogate (U+D800-U+DFFF) in raw
// markup, or -1. encoding/xml turns such a reference into U+FFFD instead of rejecting it.
func surrogateRef(raw []byte) int {
	for i := 0; ; {
		j := bytes.Index(raw[i:], []byte("&#"))
		if j < 0 {
			return -1
		}
		i += j
		end := bytes.IndexByte(raw[i:], ';')
		if end < 0 {
			return -1
		}
		ref, base := string(raw[i+2:i+end]), 10
		if strings.HasPrefix(ref, "x") {
			ref, base = ref[1:], 16
		}
		if n, err := strconv.ParseUint(ref, base, 32); err == nil && n >= 0xD800 && n <= 0xDFFF {
			return i
		}
		i += 2
	}
}

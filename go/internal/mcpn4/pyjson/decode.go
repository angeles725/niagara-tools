package pyjson

import (
	"fmt"
	"strconv"
	"unicode/utf8"
)

// maxIntDigits is CPython's default int_max_str_digits: json.loads refuses an int literal with
// more digits with a ValueError.
const maxIntDigits = 4300

// SyntaxError is a json.loads ValueError: the input is not a JSON document. Offset is the byte
// where decoding stopped.
type SyntaxError struct {
	Offset int
	Msg    string
}

func (e *SyntaxError) Error() string {
	return fmt.Sprintf("pyjson: %s at offset %d", e.Msg, e.Offset)
}

// Unmarshal returns json.loads(data) as nil, bool, string, Number, []any or *Object.
//
// Input bytes that are not valid UTF-8 are read as Python's stdin reads them in UTF-8 mode
// (surrogateescape): inside a string each such byte b becomes the lone surrogate U+DC00+b,
// stored as WTF-8; outside a string it is a syntax error. Nesting depth is not limited, where
// CPython would raise RecursionError (not a ValueError) on very deep input.
func Unmarshal(data []byte) (any, error) {
	d := decoder{s: data}
	d.skipWS()
	v, err := d.value()
	if err != nil {
		return nil, err
	}
	d.skipWS()
	if d.i != len(d.s) {
		return nil, d.fail("extra data")
	}
	return v, nil
}

type decoder struct {
	s []byte
	i int
}

func (d *decoder) fail(msg string) error { return &SyntaxError{Offset: d.i, Msg: msg} }

// skipWS skips JSON whitespace only: space, tab, LF, CR (not FF, VT or Unicode spaces).
func (d *decoder) skipWS() {
	for d.i < len(d.s) {
		switch d.s[d.i] {
		case ' ', '\t', '\n', '\r':
			d.i++
		default:
			return
		}
	}
}

func (d *decoder) peek(off int) byte {
	if d.i+off < len(d.s) {
		return d.s[d.i+off]
	}
	return 0
}

// literal consumes word when the input continues with it.
func (d *decoder) literal(word string) bool {
	if len(d.s)-d.i >= len(word) && string(d.s[d.i:d.i+len(word)]) == word {
		d.i += len(word)
		return true
	}
	return false
}

func (d *decoder) value() (any, error) {
	switch c := d.peek(0); {
	case c == '{':
		return d.object()
	case c == '[':
		return d.array()
	case c == '"':
		return d.str()
	case d.literal("null"):
		return nil, nil
	case d.literal("true"):
		return true, nil
	case d.literal("false"):
		return false, nil
	case d.literal("NaN"):
		return FloatNumber(nan), nil
	case d.literal("Infinity"):
		return FloatNumber(posInf), nil
	case d.literal("-Infinity"):
		return FloatNumber(negInf), nil
	case c == '-' || c >= '0' && c <= '9':
		return d.number()
	}
	return nil, d.fail("expecting value")
}

func (d *decoder) object() (any, error) {
	o := NewObject()
	d.i++
	d.skipWS()
	if d.peek(0) == '}' {
		d.i++
		return o, nil
	}
	for {
		if d.peek(0) != '"' {
			return nil, d.fail("expecting property name enclosed in double quotes")
		}
		k, err := d.str()
		if err != nil {
			return nil, err
		}
		d.skipWS()
		if d.peek(0) != ':' {
			return nil, d.fail("expecting ':' delimiter")
		}
		d.i++
		d.skipWS()
		v, err := d.value()
		if err != nil {
			return nil, err
		}
		o.Set(k.(string), v)
		d.skipWS()
		switch d.peek(0) {
		case '}':
			d.i++
			return o, nil
		case ',':
			d.i++
			d.skipWS()
		default:
			return nil, d.fail("expecting ',' delimiter")
		}
	}
}

func (d *decoder) array() (any, error) {
	items := []any{}
	d.i++
	d.skipWS()
	if d.peek(0) == ']' {
		d.i++
		return items, nil
	}
	for {
		v, err := d.value()
		if err != nil {
			return nil, err
		}
		items = append(items, v)
		d.skipWS()
		switch d.peek(0) {
		case ']':
			d.i++
			return items, nil
		case ',':
			d.i++
			d.skipWS()
		default:
			return nil, d.fail("expecting ',' delimiter")
		}
	}
}

func isDigit(c byte) bool { return c >= '0' && c <= '9' }

func (d *decoder) digits() {
	for isDigit(d.peek(0)) {
		d.i++
	}
}

// number matches Python's NUMBER_RE -?(0|[1-9]\d*)(\.\d+)?([eE][-+]?\d+)?: an optional part
// that is incomplete is left unconsumed (so "1." stops before the dot).
func (d *decoder) number() (any, error) {
	start := d.i
	if d.peek(0) == '-' {
		d.i++
	}
	switch c := d.peek(0); {
	case c == '0':
		d.i++
	case isDigit(c):
		d.digits()
	default:
		d.i = start
		return nil, d.fail("expecting value")
	}
	isFloat := false
	if d.peek(0) == '.' && isDigit(d.peek(1)) {
		d.i++
		d.digits()
		isFloat = true
	}
	if c := d.peek(0); c == 'e' || c == 'E' {
		n := 1
		if s := d.peek(1); s == '+' || s == '-' {
			n = 2
		}
		if isDigit(d.peek(n)) {
			d.i += n
			d.digits()
			isFloat = true
		}
	}
	lit := string(d.s[start:d.i])
	if isFloat {
		// A range error still yields Python's result: +-inf past the top, +-0 below the bottom.
		f, _ := strconv.ParseFloat(lit, 64)
		return FloatNumber(f), nil
	}
	n := len(lit)
	if lit[0] == '-' {
		n--
	}
	if n > maxIntDigits {
		d.i = start
		return nil, d.fail("int literal exceeds the 4300 digit limit")
	}
	return IntLiteral(lit), nil
}

func hexVal(s []byte) (rune, bool) {
	if len(s) < 4 {
		return 0, false
	}
	var r rune
	for _, c := range s[:4] {
		switch {
		case c >= '0' && c <= '9':
			r = r<<4 | rune(c-'0')
		case c >= 'a' && c <= 'f':
			r = r<<4 | rune(c-'a'+10)
		case c >= 'A' && c <= 'F':
			r = r<<4 | rune(c-'A'+10)
		default:
			return 0, false
		}
	}
	return r, true
}

var unescape = map[byte]byte{'"': '"', '\\': '\\', '/': '/', 'b': '\b', 'f': '\f', 'n': '\n', 'r': '\r', 't': '\t'}

// str decodes a string (strict: a raw control character is an error). A high surrogate escape
// followed by a low one joins into one code point; any other surrogate stays lone (WTF-8).
func (d *decoder) str() (any, error) {
	d.i++
	var b []byte
	for {
		if d.i >= len(d.s) {
			return nil, d.fail("unterminated string")
		}
		c := d.s[d.i]
		switch {
		case c == '"':
			d.i++
			return string(b), nil
		case c == '\\':
			r, err := d.escape()
			if err != nil {
				return nil, err
			}
			b = appendWTF8(b, r)
		case c < 0x20:
			return nil, d.fail("invalid control character")
		case c < utf8.RuneSelf:
			b = append(b, c)
			d.i++
		default:
			r, size := utf8.DecodeRune(d.s[d.i:])
			if r == utf8.RuneError && size == 1 {
				r = 0xdc00 + rune(c) // surrogateescape
			}
			b = appendWTF8(b, r)
			d.i += size
		}
	}
}

// escape decodes the escape at d.i (a backslash) and advances past it.
func (d *decoder) escape() (rune, error) {
	c := d.peek(1)
	if c != 'u' {
		r, ok := unescape[c]
		if !ok {
			d.i++
			return 0, d.fail("invalid escape")
		}
		d.i += 2
		return rune(r), nil
	}
	r, ok := hexVal(d.s[d.i+2:])
	if !ok {
		return 0, d.fail("invalid \\uXXXX escape")
	}
	d.i += 6
	if r >= 0xd800 && r <= 0xdbff && d.peek(0) == '\\' && d.peek(1) == 'u' {
		lo, ok := hexVal(d.s[d.i+2:])
		if !ok {
			return 0, d.fail("invalid \\uXXXX escape")
		}
		if lo >= 0xdc00 && lo <= 0xdfff {
			d.i += 6
			return 0x10000 + (r-0xd800)<<10 + (lo - 0xdc00), nil
		}
	}
	return r, nil
}

// appendWTF8 appends r as UTF-8, or a surrogate as its three generalized UTF-8 bytes.
func appendWTF8(b []byte, r rune) []byte {
	if r >= 0xd800 && r <= 0xdfff {
		return append(b, 0xe0|byte(r>>12), 0x80|byte(r>>6)&0x3f, 0x80|byte(r)&0x3f)
	}
	return utf8.AppendRune(b, r)
}

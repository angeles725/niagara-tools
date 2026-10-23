package pyjson

import (
	"bytes"
	"fmt"
	"sort"
	"strconv"
	"unicode/utf8"
)

// Options selects the json.dumps variant. The zero value is json.dumps(obj): ", " and ": "
// separators, insertion order. Non-ASCII is always escaped (ensure_ascii) and nothing is
// HTML-escaped.
type Options struct {
	Compact  bool // separators=(",", ":")
	SortKeys bool // sort_keys=True: keys in code point order
}

// Canonical is safety.canonical: json.dumps(obj, sort_keys=True, separators=(",", ":")).
var Canonical = Options{Compact: true, SortKeys: true}

// Marshal returns json.dumps(v) under opt. Accepted values: nil, bool, string, int, int64,
// float64, Number, []any, []string and *Object; anything else is refused (Python's TypeError),
// including map[string]any, whose iteration order is random.
func Marshal(v any, opt Options) ([]byte, error) {
	e := encoder{opt: opt, item: ", ", key: ": "}
	if opt.Compact {
		e.item, e.key = ",", ":"
	}
	if err := e.value(v); err != nil {
		return nil, err
	}
	return e.buf.Bytes(), nil
}

type encoder struct {
	buf       bytes.Buffer
	opt       Options
	item, key string
}

func (e *encoder) value(v any) error {
	switch x := v.(type) {
	case nil:
		e.buf.WriteString("null")
	case bool:
		e.buf.WriteString(strconv.FormatBool(x))
	case string:
		return e.str(x)
	case int:
		e.buf.WriteString(strconv.Itoa(x))
	case int64:
		e.buf.WriteString(strconv.FormatInt(x, 10))
	case float64:
		e.buf.WriteString(floatJSON(x))
	case Number:
		e.buf.WriteString(x.String())
	case []string:
		items := make([]any, len(x))
		for i, s := range x {
			items[i] = s
		}
		return e.array(items)
	case []any:
		return e.array(x)
	case *Object:
		return e.object(x)
	default:
		return fmt.Errorf("pyjson: object of type %T is not JSON serializable", v)
	}
	return nil
}

func (e *encoder) array(items []any) error {
	e.buf.WriteByte('[')
	for i, it := range items {
		if i > 0 {
			e.buf.WriteString(e.item)
		}
		if err := e.value(it); err != nil {
			return err
		}
	}
	e.buf.WriteByte(']')
	return nil
}

func (e *encoder) object(o *Object) error {
	keys := o.keys
	if e.opt.SortKeys {
		// Byte order of UTF-8 (and of WTF-8 surrogates, 0xED < 0xEE) is code point order,
		// which is how Python sorts str keys.
		keys = o.Keys()
		sort.Strings(keys)
	}
	e.buf.WriteByte('{')
	for i, k := range keys {
		if i > 0 {
			e.buf.WriteString(e.item)
		}
		if err := e.str(k); err != nil {
			return err
		}
		e.buf.WriteString(e.key)
		if err := e.value(o.vals[k]); err != nil {
			return err
		}
	}
	e.buf.WriteByte('}')
	return nil
}

var shortEscapes = map[byte]string{
	'"': `\"`, '\\': `\\`, '\n': `\n`, '\r': `\r`, '\t': `\t`, '\b': `\b`, '\f': `\f`,
}

// str writes s with Python's ensure_ascii escaping: short escapes, then \u%04x (lowercase) for
// controls, DEL and every non-ASCII code point, non-BMP ones as a surrogate pair.
func (e *encoder) str(s string) error {
	e.buf.WriteByte('"')
	for i := 0; i < len(s); {
		c := s[i]
		if c < utf8.RuneSelf {
			if esc, ok := shortEscapes[c]; ok {
				e.buf.WriteString(esc)
			} else if c < 0x20 || c == 0x7f {
				e.u16(rune(c))
			} else {
				e.buf.WriteByte(c)
			}
			i++
			continue
		}
		r, size := utf8.DecodeRuneInString(s[i:])
		if r == utf8.RuneError && size == 1 {
			sur, ok := wtf8Surrogate(s[i:])
			if !ok {
				return fmt.Errorf("pyjson: invalid UTF-8 at byte %d of a string", i)
			}
			r, size = sur, 3
		}
		if r >= 0x10000 {
			r -= 0x10000
			e.u16(0xd800 + r>>10)
			e.u16(0xdc00 + r&0x3ff)
		} else {
			e.u16(r)
		}
		i += size
	}
	e.buf.WriteByte('"')
	return nil
}

func (e *encoder) u16(r rune) {
	fmt.Fprintf(&e.buf, `\u%04x`, r)
}

// wtf8Surrogate decodes a surrogate code point U+D800..U+DFFF encoded as three generalized
// UTF-8 bytes (ED A0..BF 80..BF), which UTF-8 forbids but WTF-8 uses for lone surrogates.
func wtf8Surrogate(s string) (rune, bool) {
	if len(s) < 3 || s[0] != 0xed || s[1] < 0xa0 || s[1] > 0xbf || s[2] < 0x80 || s[2] > 0xbf {
		return 0, false
	}
	return 0xd000 | rune(s[1]&0x3f)<<6 | rune(s[2]&0x3f), true
}

package pyjson

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Str returns Python's str(v) for a decoded value: a string as itself, every other value as
// Repr. It is what "%s" % v prints in the server's messages.
func Str(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return Repr(v)
}

// Repr returns Python's repr(v) for a decoded value (nil, bool, string, Number, int, int64,
// float64, []any, *Object): None, True/False, a quoted string, an int's digits, repr(float),
// and lists and dicts as "[a, b]" and "{'k': v}" in insertion order.
func Repr(v any) string {
	var b strings.Builder
	repr(&b, v)
	return b.String()
}

func repr(b *strings.Builder, v any) {
	switch x := v.(type) {
	case nil:
		b.WriteString("None")
	case bool:
		if x {
			b.WriteString("True")
		} else {
			b.WriteString("False")
		}
	case string:
		b.WriteString(StrRepr(x))
	case Number:
		if x.isFloat {
			b.WriteString(FloatRepr(x.f))
		} else {
			b.WriteString(x.lit)
		}
	case int, int64:
		fmt.Fprint(b, x)
	case float64:
		b.WriteString(FloatRepr(x))
	case []any:
		b.WriteByte('[')
		for i, it := range x {
			if i > 0 {
				b.WriteString(", ")
			}
			repr(b, it)
		}
		b.WriteByte(']')
	case *Object:
		b.WriteByte('{')
		for i, k := range x.keys {
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString(StrRepr(k))
			b.WriteString(": ")
			repr(b, x.vals[k])
		}
		b.WriteByte('}')
	default:
		fmt.Fprintf(b, "<%T>", v)
	}
}

// StrRepr returns repr(s) for a Python str held as UTF-8 or WTF-8: single quotes unless s holds a
// ' and no ", the quote and backslash escaped, \t \n \r, then \xhh, \uhhhh or \Uhhhhhhhh for
// every code point that is not printable (str.isprintable, approximated by unicode.IsPrint),
// lone surrogates included.
func StrRepr(s string) string {
	quote := byte('\'')
	if strings.IndexByte(s, '\'') >= 0 && strings.IndexByte(s, '"') < 0 {
		quote = '"'
	}
	var b strings.Builder
	b.WriteByte(quote)
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		if r == utf8.RuneError && size == 1 {
			if sur, ok := wtf8Surrogate(s[i:]); ok {
				r, size = sur, 3
			} else {
				r = rune(s[i]) // not reachable for decoded strings; keep the byte visible
			}
		}
		i += size
		switch {
		case r == rune(quote) || r == '\\':
			b.WriteByte('\\')
			b.WriteRune(r)
		case r == '\t':
			b.WriteString(`\t`)
		case r == '\n':
			b.WriteString(`\n`)
		case r == '\r':
			b.WriteString(`\r`)
		case r >= 0xd800 && r <= 0xdfff || !unicode.IsPrint(r):
			switch {
			case r < 0x100:
				fmt.Fprintf(&b, `\x%02x`, r)
			case r < 0x10000:
				fmt.Fprintf(&b, `\u%04x`, r)
			default:
				fmt.Fprintf(&b, `\U%08x`, r)
			}
		default:
			b.WriteRune(r)
		}
	}
	b.WriteByte(quote)
	return b.String()
}

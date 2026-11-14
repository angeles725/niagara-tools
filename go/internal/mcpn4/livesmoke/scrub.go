package livesmoke

import (
	"fmt"
	"sort"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// Mask replaces every credential value in output.
const Mask = "***"

// Scrub replaces every non-empty secret in text with Mask, the longest secret first.
func Scrub(text string, secrets []string) string {
	var live []string
	for _, s := range secrets {
		if s != "" {
			live = append(live, s)
		}
	}
	sort.SliceStable(live, func(i, j int) bool { return len(live[i]) > len(live[j]) })
	for _, s := range live {
		text = strings.ReplaceAll(text, s, Mask)
	}
	return text
}

// ScrubValue scrubs every string (dict keys included) in a decoded JSON value, before encoding.
// A scrubbed key that collides with an earlier one replaces its value in place, as a Python dict
// comprehension does.
func ScrubValue(v any, secrets []string) any {
	switch x := v.(type) {
	case string:
		return Scrub(x, secrets)
	case *pyjson.Object:
		out := pyjson.NewObject()
		for _, k := range x.Keys() {
			val, _ := x.Get(k)
			out.Set(Scrub(k, secrets), ScrubValue(val, secrets))
		}
		return out
	case []any:
		out := make([]any, len(x))
		for i, val := range x {
			out[i] = ScrubValue(val, secrets)
		}
		return out
	case []string:
		out := make([]any, len(x))
		for i, val := range x {
			out[i] = Scrub(val, secrets)
		}
		return out
	}
	return v
}

// RenderReport is the JSON text of the report (json.dumps indent=2) with no credential in it, raw
// or JSON-escaped: the structure is scrubbed first (raw values), encoded, then the text is
// scrubbed again with the escaped spellings as a second line of defense.
func RenderReport(report *pyjson.Object, secrets []string) (string, error) {
	compact, err := pyjson.Marshal(ScrubValue(report, secrets), pyjson.Options{Compact: true})
	if err != nil {
		return "", err
	}
	all := append([]string(nil), secrets...)
	for _, s := range secrets {
		if s != "" {
			ascii, _ := pyjson.Marshal(s, pyjson.Options{})
			all = append(all, string(ascii[1:len(ascii)-1]), escapeKeepUnicode(s))
		}
	}
	return Scrub(indent(compact), all), nil
}

// escapeKeepUnicode is json.dumps(s, ensure_ascii=False)[1:-1]: only the quote, the backslash and
// control characters are escaped.
func escapeKeepUnicode(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch r {
		case '"':
			b.WriteString(`\"`)
		case '\\':
			b.WriteString(`\\`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		case '\b':
			b.WriteString(`\b`)
		case '\f':
			b.WriteString(`\f`)
		default:
			if r < 0x20 {
				fmt.Fprintf(&b, `\u%04x`, r)
			} else {
				b.WriteRune(r)
			}
		}
	}
	return b.String()
}

// indent re-lays compact JSON as json.dumps(indent=2) does: one item per line, ": " after keys,
// empty containers kept as [] and {}.
func indent(compact []byte) string {
	var b strings.Builder
	depth, inStr, esc := 0, false, false
	newline := func() { b.WriteString("\n" + strings.Repeat("  ", depth)) }
	for i := 0; i < len(compact); i++ {
		c := compact[i]
		if inStr {
			b.WriteByte(c)
			if esc {
				esc = false
			} else if c == '\\' {
				esc = true
			} else if c == '"' {
				inStr = false
			}
			continue
		}
		switch c {
		case '"':
			inStr = true
			b.WriteByte(c)
		case '{', '[':
			b.WriteByte(c)
			if i+1 < len(compact) && (compact[i+1] == '}' || compact[i+1] == ']') {
				b.WriteByte(compact[i+1])
				i++
				continue
			}
			depth++
			newline()
		case '}', ']':
			depth--
			newline()
			b.WriteByte(c)
		case ',':
			b.WriteByte(c)
			newline()
		case ':':
			b.WriteString(": ")
		default:
			b.WriteByte(c)
		}
	}
	return b.String()
}

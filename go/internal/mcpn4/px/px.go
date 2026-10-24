// Package px summarizes one Px view document as files.parse_px does (mcp-n4-kit/mcp_n4/files.py):
// media, imports, canvases, layers, widget and binding counts, bound ORDs grouped by component,
// images, included Px files, charts, other ORDs and labels, every list bounded. The XML is read
// by pyxml, ElementTree's parser.
package px

import (
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyxml"
)

// valueSlots is VALUE_SLOTS: slot names that hold a value, not a component.
var valueSlots = func() map[string]bool {
	m := map[string]bool{}
	for _, s := range strings.Fields("out in fallback value status set facets proxyExt override emergencyOverride") {
		m[s] = true
	}
	for i := 1; i <= 16; i++ {
		m["in"+strconv.Itoa(i)] = true
	}
	return m
}()

// slotHop is _SLOT_HOP; Python's $ also matches before a final newline.
var slotHop = regexp.MustCompile(`\|slot:[A-Za-z0-9_$]+\n?$`)

func hasAnyPrefix(s string, prefixes ...string) bool {
	for _, p := range prefixes {
		if strings.HasPrefix(s, p) {
			return true
		}
	}
	return false
}

func hasAnySuffix(s string, suffixes ...string) bool {
	for _, x := range suffixes {
		if strings.HasSuffix(s, x) {
			return true
		}
	}
	return false
}

func isOrd(v string) bool {
	return hasAnyPrefix(v, "station:|", "slot:", "h:", "local:|", "history:", "ip:", "file:", "module:", "bql:", "service:")
}

func isSlotOrd(v string) bool { return hasAnyPrefix(v, "station:|slot:", "slot:", "local:|slot:") }

func isImage(v string) bool {
	head := pyjson.Lower(v)
	if i := strings.LastIndexByte(head, '?'); i >= 0 {
		head = head[:i]
	}
	return hasAnySuffix(head, ".png", ".gif", ".jpg", ".jpeg", ".svg", ".bmp", ".ico")
}

// BoundComponent is bound_component: the component ORD a binding addresses, a trailing slot hop
// or known value slot removed (a heuristic: any other last segment is the component itself).
func BoundComponent(ord string) string {
	if loc := slotHop.FindStringIndex(ord); loc != nil {
		return ord[:loc[0]]
	}
	if i := strings.LastIndexByte(ord, '/'); i >= 0 && valueSlots[ord[i+1:]] && !strings.HasSuffix(ord[:i], ":") {
		return ord[:i]
	}
	return ord
}

// declaration is re.search(r"<!(DOCTYPE|ENTITY)", text, re.I): under re.I a letter matches any
// character whose simple lowercase it is, and "i" also matches U+0131 (dotless i).
func declaration(text string) bool {
	for i := strings.Index(text, "<!"); i >= 0; {
		rest := text[i+2:]
		if foldPrefix(rest, "doctype") || foldPrefix(rest, "entity") {
			return true
		}
		j := strings.Index(rest, "<!")
		if j < 0 {
			return false
		}
		i += 2 + j
	}
	return false
}

func foldPrefix(s, word string) bool {
	for _, w := range word {
		if s == "" {
			return false
		}
		r, size := pyjson.DecodeRune(s)
		if unicode.ToLower(r) != w && !(w == 'i' && r == 0x131) {
			return false
		}
		s = s[size:]
	}
	return true
}

// bounded is _Bounded: an ordered, de-duplicated list capped at limit; truncated once a value
// is dropped. Values compare by their JSON (str, None and lists of them).
type bounded struct {
	limit     int
	items     []any
	seen      map[string]bool
	truncated bool
}

func newBounded(limit int) *bounded {
	return &bounded{limit: limit, items: []any{}, seen: map[string]bool{}}
}

func (b *bounded) add(v any) {
	key, _ := pyjson.Marshal(v, pyjson.Options{})
	if b.seen[string(key)] {
		return
	}
	if len(b.items) >= b.limit {
		b.truncated = true
		return
	}
	b.seen[string(key)] = true
	b.items = append(b.items, v)
}

func obj(kv ...any) *pyjson.Object {
	o := pyjson.NewObject()
	for i := 0; i < len(kv); i += 2 {
		o.Set(kv[i].(string), kv[i+1])
	}
	return o
}

// attr is elem.get(name): the value, or None.
func attr(el *pyxml.Element, name string) any {
	if v, ok := el.Get(name); ok {
		return v
	}
	return nil
}

func count(c *pyjson.Object, tag string) {
	n, _ := c.Get(tag)
	k, _ := n.(int)
	c.Set(tag, k+1)
}

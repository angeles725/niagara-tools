package bogaudit

import (
	"regexp"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/htmltext"
	"github.com/angeles725/niagara-tools/go/internal/xmlsrc"
)

// omap is a Python dict of string keys: iteration in first-insertion order, a later set replaces the value.
type omap[V any] struct {
	keys []string
	m    map[string]V
}

func (o *omap[V]) set(k string, v V) {
	if o.m == nil {
		o.m = map[string]V{}
	}
	if _, ok := o.m[k]; !ok {
		o.keys = append(o.keys, k)
	}
	o.m[k] = v
}

func (o *omap[V]) get(k string) (V, bool) { v, ok := o.m[k]; return v, ok }

// slot is a value property written as <p n=... v=... f=.../>; hasV false is Python's None value.
type slot struct {
	typ, value, flags string
	hasV              bool
}

// comp is a component: a <p> start tag with an h= attribute that is not an empty-element tag.
type comp struct {
	name, handle, typ, module, path string
	slots                           omap[slot]
	actions                         omap[string] // <a n=... f=.../>: action -> bog flags
	hasFallback, writable           bool
}

func (c *comp) simpleType() string { return c.typ[strings.LastIndexByte(c.typ, ':')+1:] }
func (c *comp) className() string  { return "B" + c.simpleType() }

// link is one b:Link: the handle of the component holding it (ok false: none) and its three children.
// "" is Python's None for the children (an empty value was never recorded).
type link struct {
	container            string
	hasContainer         bool
	srcH, srcSlot, tgtSl string
}

// graph is what the scan of the bog builds: components by handle (dict order), the count of each handle
// and every link in document order.
type graph struct {
	comps  omap[*comp]
	counts omap[int]
	links  []link
}

func (g *graph) comp(h string) *comp { c, _ := g.comps.get(h); return c }

type frameKind int

const (
	other frameKind = iota
	compFrame
	linkFrame
)

type frame struct {
	kind frameKind
	path string
	c    *comp
}

// build replays the start and end tags of doc through the bash script's line-scanner state machine:
// only </p> and </a> pop a frame, an <a> pushes none, an empty-element tag sees no end tag.
func build(doc *xmlsrc.Doc) *graph {
	b := &builder{g: &graph{}, prefixes: map[string]string{}}
	elems := doc.Elements()
	var open []int // elements whose end tag is still to come
	closeTo := func(depth int) {
		for len(open) > 0 && elems[open[len(open)-1]].Depth >= depth {
			b.end(elems[open[len(open)-1]].Name)
			open = open[:len(open)-1]
		}
	}
	for i, e := range elems {
		closeTo(e.Depth)
		b.start(e)
		if !e.Empty {
			open = append(open, i)
		}
	}
	closeTo(0)
	return b.g
}

type builder struct {
	g        *graph
	prefixes map[string]string
	stack    []frame
	inLink   bool
	cur      link
}

func (b *builder) top() frame {
	if len(b.stack) == 0 {
		return frame{}
	}
	return b.stack[len(b.stack)-1]
}

func (b *builder) nearest() *comp {
	for i := len(b.stack) - 1; i >= 0; i-- {
		if b.stack[i].kind == compFrame {
			return b.stack[i].c
		}
	}
	return nil
}

func (b *builder) end(name string) {
	if (name != "p" && name != "a") || len(b.stack) == 0 {
		return
	}
	popped := b.stack[len(b.stack)-1]
	b.stack = b.stack[:len(b.stack)-1]
	if popped.kind == linkFrame {
		b.g.links = append(b.g.links, b.cur)
		b.inLink, b.cur = false, link{}
	}
}

func attr(e xmlsrc.Element, name string) string { v, _ := e.Attr(name); return v }

func (b *builder) start(e xmlsrc.Element) {
	if e.Name == "a" {
		if c := b.nearest(); c != nil {
			c.actions.set(attr(e, "n"), attr(e, "f"))
		}
		return
	}
	if e.Name != "p" {
		return
	}
	n, t, f := attr(e, "n"), attr(e, "t"), attr(e, "f")
	h, hasH := e.Attr("h")
	v, hasV := e.Attr("v")
	for _, part := range strings.FieldsFunc(attr(e, "m"), htmltext.IsSpace) {
		if k, mod, ok := strings.Cut(part, "="); ok {
			b.prefixes[k] = mod
		}
	}
	parent := b.top()
	switch {
	case b.inLink:
		switch {
		case e.Empty && n == "sourceOrd" && strings.HasPrefix(v, "h:"):
			b.cur.srcH = v[2:]
		case e.Empty && n == "sourceSlotName" && v != "":
			b.cur.srcSlot = v
		case e.Empty && n == "targetSlotName" && v != "":
			b.cur.tgtSl = v
		case !e.Empty && !hasH:
			b.stack = append(b.stack, frame{kind: other, path: parent.path})
		}
	case isLinkType(t) && !e.Empty:
		b.cur = link{}
		if c := b.nearest(); c != nil {
			b.cur.container, b.cur.hasContainer = c.handle, true
		}
		b.inLink = true
		b.stack = append(b.stack, frame{kind: linkFrame, path: parent.path})
	case hasH && !e.Empty:
		pfx := ""
		if i := strings.IndexByte(t, ':'); i >= 0 {
			pfx = t[:i]
		}
		c := &comp{name: n, handle: h, typ: t, module: b.prefixes[pfx], path: strings.TrimLeft(parent.path+"/"+n, "/")}
		c.writable = writableType.MatchString(c.simpleType())
		cnt, _ := b.g.counts.get(h)
		b.g.counts.set(h, cnt+1)
		b.g.comps.set(h, c)
		b.stack = append(b.stack, frame{kind: compFrame, path: c.path, c: c})
	case n == "fallback":
		if c := b.nearest(); c != nil {
			c.hasFallback = true
		}
		if !e.Empty {
			b.stack = append(b.stack, frame{kind: other, path: parent.path + "/fallback"})
		}
	case e.Empty:
		if (hasV || f != "") && parent.kind == compFrame && !platformSlots[n] {
			parent.c.slots.set(n, slot{typ: t, value: v, flags: f, hasV: hasV})
		}
	default:
		path := parent.path
		if n != "" {
			path = strings.TrimLeft(path+"/"+n, "/")
		}
		b.stack = append(b.stack, frame{kind: other, path: path})
	}
}

// isLinkType is a link record: a b:Link or a b:ConversionLink (a link with a converter child, e.g.
// EnumToStatusBoolean). The python scanner knew only t='b:Link' and lost every conversion link.
func isLinkType(t string) bool {
	return strings.HasSuffix(t, ":Link") || strings.HasSuffix(t, ":ConversionLink")
}

// writableType is a proxy point type the audit treats as a writable (CHECK8, CHECK11).
var writableType = regexp.MustCompile(`^(Boolean|Numeric|Float|Integer)Writable$`)

// platformSlots are the platform-managed names never recorded as a component's slot.
var platformSlots = map[string]bool{"wsAnnotation": true, "value": true, "status": true, "displayName": true}

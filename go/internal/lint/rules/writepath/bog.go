package writepath

import (
	"regexp"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/lint"
	"github.com/angeles725/niagara-tools/go/internal/xmlsrc"
)

var (
	dashboard = regexp.MustCompile(`(?i)Dashboard|RoomPanel|DashPanel`)
	profileOf = regexp.MustCompile(`-(rt|wb|ux|se)$`)
)

// frame is one open <p> (or b:Link) element of the bog walk: the module of its type, if it has a handle.
type frame struct {
	mod  string
	link bool
}

// bogSlots are the --bog target slots: every b:Link whose source component (sourceOrd h:<handle>) has a
// type from a Dashboard/RoomPanel/DashPanel module and whose nearest enclosing typed component is of the
// root's own module (the root's name, a -rt/-wb/-ux/-se suffix dropped). The bog is read through xmlsrc
// (a zipped config.bog unzipped in memory); a bog that cannot be read or is malformed is an ERROR row.
//
// The walk replays the bash lint's python helper over the element tree in document order: a module
// prefix (m="pfx=module ...") is known from the element that declares it on; only an element with
// children opens a frame (the helper read a childless <p .../> as self-closing: its handle is never a
// link source); a closing </a> pops a frame as the helper's did.
func (s *scan) bogSlots(path string) (map[string]bool, error) {
	if path == "" {
		return nil, nil
	}
	doc, err := xmlsrc.Open(path)
	if err != nil {
		return nil, s.errorRow("--bog: " + err.Error())
	}
	for _, w := range lint.XMLWarnings(doc) {
		w.Path = path
		s.bogWarns = append(s.bogWarns, w)
	}
	own := profileOf.ReplaceAllString(lint.Basename(strings.TrimRight(s.root, "/")), "")
	if strings.Trim(s.root, "/") == "" { // python's basename of "" (a root of only slashes)
		own = ""
	}
	elems := doc.Elements()
	prefix, handleMod, slots := map[string]string{}, map[string]string{}, map[string]bool{}
	var stack []frame
	var open []int // the elements with children not yet closed, innermost last
	inLink := false
	var srcH, tgt string
	closeTo := func(depth int) {
		for len(open) > 0 && elems[open[len(open)-1]].Depth >= depth {
			e := elems[open[len(open)-1]]
			open = open[:len(open)-1]
			if (e.Name != "p" && e.Name != "a") || len(stack) == 0 {
				continue
			}
			top := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			if !top.link {
				continue
			}
			if srcH != "" && tgt != "" && dashboard.MatchString(handleMod[srcH]) && enclosing(stack) == own {
				slots[tgt] = true
			}
			inLink, srcH, tgt = false, "", ""
		}
	}
	for i, e := range elems {
		closeTo(e.Depth)
		leaf := i+1 == len(elems) || elems[i+1].Depth <= e.Depth
		if !leaf {
			open = append(open, i)
		}
		if m, _ := e.Attr("m"); m != "" {
			for _, part := range strings.Fields(m) {
				if k, v, ok := strings.Cut(part, "="); ok {
					prefix[k] = v
				}
			}
		}
		if e.Name != "p" {
			continue
		}
		n, _ := e.Attr("n")
		v, _ := e.Attr("v")
		t, _ := e.Attr("t")
		switch {
		case inLink && leaf:
			if n == "sourceOrd" && strings.HasPrefix(v, "h:") {
				srcH = v[2:]
			} else if n == "targetSlotName" && v != "" {
				tgt = v
			}
		case t == "b:Link" && !leaf:
			inLink, srcH, tgt = true, "", ""
			stack = append(stack, frame{link: true})
		case !leaf:
			f := frame{}
			if h, ok := e.Attr("h"); ok {
				pfx, _, typed := strings.Cut(t, ":")
				if !typed {
					pfx = ""
				}
				if f.mod = prefix[pfx]; f.mod != "" {
					handleMod[h] = f.mod
				}
			}
			stack = append(stack, f)
		}
	}
	closeTo(0)
	return slots, nil
}

// enclosing is the module of the innermost open frame that has one.
func enclosing(stack []frame) string {
	for i := len(stack) - 1; i >= 0; i-- {
		if stack[i].mod != "" {
			return stack[i].mod
		}
	}
	return ""
}

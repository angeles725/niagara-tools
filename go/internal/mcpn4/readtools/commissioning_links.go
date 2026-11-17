package readtools

import (
	"fmt"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/bql"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// The links section, over one link scan of the scope (the scan of n4_list_links): the links
// whose source names nothing, the writable inputs several links feed at one priority level,
// and the outputs the dangling-output check reports.

// linksSection is the links section.
func (r *commRun) linksSection() (*commSection, error) {
	s := r.section("links")
	names := []string{"unresolved_source", "inputs_with_several_links", "dangling_outputs"}
	r.requests++
	ord, depth := r.scope, r.depth
	nodes, links, err := r.t.scanOrd(ord, depth)
	if err != nil {
		why, err := r.failure("the link scan of "+r.scope+" failed", err)
		s.notChecked(why, names...)
		return s, err
	}
	components := 0
	for _, p := range nodes.Keys() {
		if h, _ := box.Node(nodes, p).Get("h"); h != nil && h != "" {
			components++
		}
	}
	s.extra = []any{"components", components, "links", len(links), "note", fmt.Sprintf(noteLinks, depth)}
	if components > r.maxRows { // the cap bounds what one report judges
		s.notChecked(fmt.Sprintf(whyLinkCap, r.scope, depth, components, r.maxRows), names...)
		if s.truncated = r.maxRows; r.truncated == 0 {
			r.truncated = r.maxRows
		}
		return s, nil
	}
	r.linkSources(s.check(names[0]), nodes, links, depth)
	severalLinks(s.check(names[1]), links)
	dangling, unknown, _, _, err := danglingOutputs(ord, nodes, links, depth) // an extension consumer is no finding
	if err != nil {
		return nil, err
	}
	c := s.check(names[2])
	for _, item := range dangling {
		path, _ := lookup(item.(*pyjson.Object), "path").(string)
		c.add(slotPrefix+path, bql.UnescapeSlot(lastSegment(path)), fmt.Sprintf(factDangling, pyjson.Str(lookup(item.(*pyjson.Object), "type"))))
	}
	if len(unknown) > 0 { // a link with an unresolved source may read them
		c.extra = []any{"unknown", len(unknown)}
	}
	return s, nil
}

// linkSources lists in c the links whose source names nothing: a source ORD that is neither a
// handle nor an absolute slot path, and a slot path inside the loaded levels of the scope that
// no component has. A source outside what was loaded is counted in source_outside_scan.
func (r *commRun) linkSources(c *commCheck, nodes *pyjson.Object, links []link, depth int) {
	outside := 0
	for _, l := range links {
		target, into := slotPrefix+l.targetPath, pyjson.Str(l.targetSlot)
		source, _ := l.sourcePath.(string)
		switch ord, _ := l.sourceOrd.(string); {
		case !l.resolved:
			c.add(target, l.linkName, fmt.Sprintf(factSourceForm, l.linkName, into, pyjson.Repr(l.sourceOrd)))
		case l.sourcePath == nil:
			outside++
		case !strings.HasPrefix(ord[strings.LastIndex(ord, "|")+1:], slotPrefix): // by handle: a loaded component
		case !r.loaded(source, depth):
			outside++
		case !hasNode(nodes, r.relative(source)):
			c.add(target, l.linkName, fmt.Sprintf(factSourceMissing, l.linkName, into, source))
		}
	}
	c.extra = []any{"source_outside_scan", outside}
}

// relative is path below the scope as the scan keys it ("" for the scope itself), nil when
// path is not below the scope.
func (r *commRun) relative(path string) any {
	base := strings.TrimRight(r.slot, "/")
	switch {
	case path == r.slot:
		return ""
	case strings.HasPrefix(path, base+"/"):
		return path[len(base)+1:]
	}
	return nil
}

// loaded reports whether path is inside the levels the scan loaded: at most depth+2 below the
// scope, one above the deepest slots.
func (r *commRun) loaded(path string, depth int) bool {
	rel, ok := r.relative(path).(string)
	return ok && level(rel) <= depth+2
}

// hasNode reports whether the scan holds a node at rel (a relative path).
func hasNode(nodes *pyjson.Object, rel any) bool {
	key, _ := rel.(string)
	_, has := nodes.Get(key)
	return has
}

// severalLinks lists in c the priority inputs (priorityInputRe) that more than one link
// targets: the station shows whichever wrote last.
func severalLinks(c *commCheck, links []link) {
	sources, order := map[string][]string{}, []string{}
	for _, l := range links {
		slot, _ := l.targetSlot.(string)
		if !priorityInputRe.MatchString(slot) {
			continue
		}
		key := l.targetPath + "\x00" + slot
		if _, seen := sources[key]; !seen {
			order = append(order, key)
		}
		from := pyjson.Str(l.sourceOrd)
		if p, ok := l.sourcePath.(string); ok {
			from = p
		}
		sources[key] = append(sources[key], from+"."+pyjson.Str(l.sourceSlot))
	}
	for _, key := range order {
		if from := sources[key]; len(from) > 1 {
			path, slot, _ := strings.Cut(key, "\x00")
			c.add(slotPrefix+path, bql.UnescapeSlot(lastSegment(path)), fmt.Sprintf(factSeveralLinks, slot, len(from), strings.Join(from, ", ")))
		}
	}
}

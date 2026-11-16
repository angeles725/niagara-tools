package boxstation

import "strings"

// dropLinksFrom is what a station does when a component is removed: every link stored elsewhere
// whose source is the removed component or one under it is removed with it (bench retro
// 2026-11-15 F2: the links of a removed constant block into other blocks were gone).
func (st *Station) dropLinksFrom(removed *Node) {
	gone := map[string]bool{}
	var mark func(n *Node)
	mark = func(n *Node) {
		if n.Handle != "" {
			gone["h:"+n.Handle] = true
		}
		for _, c := range n.Children {
			mark(c)
		}
	}
	mark(removed)
	var sweep func(n *Node)
	sweep = func(n *Node) {
		kept := n.Children[:0]
		for _, c := range n.Children {
			if c.Type == "baja:Link" && gone[linkSource(c)] {
				continue
			}
			sweep(c)
			kept = append(kept, c)
		}
		n.Children = kept
	}
	sweep(st.root)
}

// linkSource is the `h:<handle>` tail of a link's sourceOrd, "" when it has none.
func linkSource(link *Node) string {
	src := link.Child("sourceOrd")
	if src == nil {
		return ""
	}
	s, _ := src.Value.(string)
	return s[strings.LastIndex(s, "|")+1:]
}

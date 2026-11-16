package boxstation

import (
	"fmt"
	"strings"
)

// resolve is the node at a station:|slot:/a/b ORD; a double slash is refused like N4.14 does.
func (st *Station) resolve(ord string) (*Node, error) {
	if strings.Contains(ord, "//") {
		return nil, fmt.Errorf("ValueError: Illegal double slashes")
	}
	_, path, _ := strings.Cut(ord, "slot:")
	n := st.root
	for _, part := range strings.Split(path, "/") {
		if part == "" {
			continue
		}
		if n = n.Child(part); n == nil {
			return nil, fmt.Errorf("KeyError: '%s'", ord)
		}
	}
	return n, nil
}

// resolveComponent is what a load answers for ord: like N4, the nearest component at or above
// the slot it names (a struct or simple property loads its component; live retro row M3).
func (st *Station) resolveComponent(ord string) (*Node, error) {
	if _, err := st.resolve(ord); err != nil {
		return nil, err
	}
	_, path, _ := strings.Cut(ord, "slot:")
	n, found := st.root, st.root
	for _, part := range strings.Split(path, "/") {
		if part != "" {
			if n = n.Child(part); n.Handle != "" {
				found = n
			}
		}
	}
	return found, nil
}

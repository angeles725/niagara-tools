package boxstation

import "fmt"

// SetRegistry describes types on the registry channel (reg/loadTypes) from now on: a spec maps to
// its entry ({c: [declared slots {st, n, f?, v?}], p: supertype}), or to nil for a type the
// station does not know (left out of the reply, as a live station leaves out a type it cannot
// describe). A spec not in entries keeps the default answer {} (every slot dynamic).
func (st *Station) SetRegistry(entries map[string]any) {
	st.Do(func(*Station) { st.registry = entries })
}

// loadTypes answers reg/loadTypes; the station is locked.
func (st *Station) loadTypes(body map[string]any) map[string]any {
	specs, _ := body["t"].([]any)
	out := map[string]any{}
	for _, s := range specs {
		spec := fmt.Sprint(s)
		entry, described := st.registry[spec]
		switch {
		case !described:
			out[spec] = map[string]any{}
		case entry != nil:
			out[spec] = entry
		}
	}
	return out
}

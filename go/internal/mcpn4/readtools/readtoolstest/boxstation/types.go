package boxstation

import "fmt"

// SetSupertype makes the type registry answer super as the supertype of spec ("" forgets it). A
// type without one is answered {}: every slot of the fake is dynamic.
func (st *Station) SetSupertype(spec, super string) {
	st.Do(func(*Station) {
		if st.supertypes == nil {
			st.supertypes = map[string]string{}
		}
		if st.supertypes[spec] = super; super == "" {
			delete(st.supertypes, spec)
		}
	})
}

// loadTypes is the registry channel's loadTypes: an entry for each requested spec and, like a
// station, for every supertype it depends on ({p: supertype} when the type has one).
func (st *Station) loadTypes(body map[string]any) map[string]any {
	specs, _ := body["t"].([]any)
	out := map[string]any{}
	for _, s := range specs {
		for spec := fmt.Sprint(s); spec != "" && out[spec] == nil; spec = st.supertypes[spec] {
			entry := map[string]any{}
			if super := st.supertypes[spec]; super != "" {
				entry["p"] = super
			}
			out[spec] = entry
		}
	}
	return out
}

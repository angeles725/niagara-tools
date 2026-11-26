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
// station, for every supertype it depends on ({p: supertype} when the type has one). A spec
// SetRegistry describes answers its entry instead (nil: left out); the station is locked.
func (st *Station) loadTypes(body map[string]any) map[string]any {
	specs, _ := body["t"].([]any)
	out := map[string]any{}
	for _, s := range specs {
		for spec := fmt.Sprint(s); spec != "" && out[spec] == nil; spec = st.supertypes[spec] {
			if entry, described := st.registry[spec]; described {
				if entry != nil {
					out[spec] = entry
				}
				break
			}
			entry := map[string]any{}
			if super := st.supertypes[spec]; super != "" {
				entry["p"] = super
			}
			out[spec] = entry
		}
	}
	return out
}

// isComponent is the write tools' rule (box.IsComponentType) except for a type whose registry
// supertype chain reaches baja:Struct (SetSupertype): a struct is a slot value, as on a station (B1200-G3).
func (st *Station) isComponent(t any) bool {
	spec, _ := t.(string)
	for i := 0; i < 8 && spec != ""; i++ {
		if spec == "baja:Struct" {
			return false
		}
		spec = st.supertypes[spec]
	}
	return isComponentType(t)
}

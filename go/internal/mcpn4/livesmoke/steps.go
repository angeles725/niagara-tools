package livesmoke

import (
	"sort"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// children is the names of the station root's children.
func (s *Scenario) children() ([]any, error) {
	out, err := s.read("n4_navigate", kwargs("ord", RootOrd, "depth", 1))
	if err != nil {
		return nil, err
	}
	list, err := item(out, "children")
	if err != nil {
		return nil, err
	}
	kids, err := iterate(list)
	if err != nil {
		return nil, err
	}
	names := make([]any, 0, len(kids))
	for _, c := range kids {
		n, err := item(c, "name")
		if err != nil {
			return nil, err
		}
		names = append(names, n)
	}
	return names, nil
}

// scratchPresent is `SCRATCH in self.children()`.
func (s *Scenario) scratchPresent() (bool, error) {
	names, err := s.children()
	if err != nil {
		return false, err
	}
	for _, n := range names {
		if n == Scratch {
			return true, nil
		}
	}
	return false, nil
}

// slotValue is the value of comp's out slot, nil when the component has none.
func (s *Scenario) slotValue(comp string) (any, error) {
	out, err := s.read("n4_read_slots", kwargs("ord", inFolder(comp)))
	if err != nil {
		return nil, err
	}
	list, err := item(out, "slots")
	if err != nil {
		return nil, err
	}
	slots, err := iterate(list)
	if err != nil {
		return nil, err
	}
	var found []any
	for _, sl := range slots {
		name, err := item(sl, "name")
		if err != nil {
			return nil, err
		}
		if name == "out" {
			found = append(found, sl)
		}
	}
	if len(found) == 0 {
		return nil, nil
	}
	return item(found[0], "value")
}

func (s *Scenario) connect(*pyjson.Object) (any, any, error) {
	out, err := s.read("n4_connect", kwargs("station", s.station, "expected_station", s.station))
	if err != nil {
		return nil, nil, err
	}
	name, err := get(out, "station_name")
	if err != nil {
		return nil, nil, err
	}
	verdict := "mismatch"
	if name == s.station {
		verdict = "verified"
	}
	version, err := get(out, "version")
	if err != nil {
		return nil, nil, err
	}
	tier, err := get(out, "tier")
	if err != nil {
		return nil, nil, err
	}
	// version and tier make the report the PoC file an --allow-tier-b opt-in can point to (#985 Δ3)
	return verdict, kwargs("station_name", name, "version", version, "tier", tier), nil
}

func (s *Scenario) absentBefore(*pyjson.Object) (any, any, error) {
	present, err := s.scratchPresent()
	if err != nil {
		return nil, nil, err
	}
	if present {
		return "failed", kwargs("detail", Scratch+" already exists: refusing to touch it"), nil
	}
	return "verified", kwargs("detail", "absent"), nil
}

func (s *Scenario) create(name, typ string, ws *component, parent string) stepFn {
	return func(entry *pyjson.Object) (any, any, error) {
		args := kwargs("parent_ord", parent, "name", name, "type", typ)
		if ws != nil {
			args.Set("wire_sheet", kwargs("x", ws.X, "y", ws.Y, "w", 8))
		}
		// The folder may exist on the station from the moment the execute call is sent, even if
		// its reply never arrives: mark it first so cleanup always looks.
		var mark func()
		if name == Scratch {
			mark = func() { s.created = true }
		}
		out, err := s.Write(entry, "n4_create_component", mark, args)
		if err != nil {
			return nil, nil, err
		}
		return verdictOf(out)
	}
}

func (s *Scenario) link(l link) stepFn {
	return func(entry *pyjson.Object) (any, any, error) {
		out, err := s.Write(entry, "n4_create_link", nil, kwargs("source_ord", inFolder(l.Src),
			"source_slot", l.SrcSlot, "target_ord", inFolder(l.Dst), "target_slot", l.DstSlot))
		if err != nil {
			return nil, nil, err
		}
		return verdictOf(out)
	}
}

func (s *Scenario) dangling(*pyjson.Object) (any, any, error) {
	out, err := s.read("n4_find_dangling_outputs", kwargs("ord", FolderOrd, "depth", 1))
	if err != nil {
		return nil, nil, err
	}
	list, err := item(out, "dangling")
	if err != nil {
		return nil, nil, err
	}
	items, err := iterate(list)
	if err != nil {
		return nil, nil, err
	}
	names := []string{}
	for _, d := range items {
		p, err := item(d, "path")
		if err != nil {
			return nil, nil, err
		}
		ps, ok := p.(string)
		if !ok {
			return nil, nil, &pyError{"AttributeError", "'" + typeName(p) + "' object has no attribute 'rsplit'"}
		}
		names = append(names, ps[strings.LastIndex(ps, "/")+1:])
	}
	sort.Strings(names)
	verdict := "mismatch"
	if len(names) == 1 && names[0] == "Cooling" {
		verdict = "verified"
	}
	return verdict, kwargs("dangling", names), nil
}

func (s *Scenario) setValue(comp string, value int) stepFn {
	return func(entry *pyjson.Object) (any, any, error) {
		out, err := s.Write(entry, "n4_invoke_action", nil, kwargs("ord", inFolder(comp),
			"action", "set", "arg", value, "arg_type", "baja:Double"))
		if err != nil {
			return nil, nil, err
		}
		return verdictOf(out)
	}
}

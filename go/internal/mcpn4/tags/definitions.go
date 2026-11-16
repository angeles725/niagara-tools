package tags

import (
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// classDefaults are the value types of tag definitions whose defValue a load omits (it is at the
// class default): BSimpleTagInfo defaults to a Marker, BDynamicEnumTagInfo to a DynamicEnum.
var classDefaults = map[string]string{
	"tagdictionary:SimpleTagInfo":      "baja:Marker",
	"tagdictionary:DynamicEnumTagInfo": "baja:DynamicEnum",
}

// TagDef is one tag definition: its id, info type, the value type and default it gives a direct
// tag (ValueType "" for a smart tag the station computes), and its validity condition.
type TagDef struct {
	ID, Name, Info, ValueType string
	Default                   any
	Validity                  *pyjson.Object // {type, display}, nil when the load has none
}

// GroupDef is one tag group definition: its id, the slot ORD a tag group relation points at
// (the form Workbench writes), and the ids of the tags it implies.
type GroupDef struct {
	ID, Name, Ord string
	Tags          []string
}

// TagDefs are the tag definitions of d (one load of its tagDefinitions, depth 1).
func TagDefs(l Loader, d Dictionary) ([]TagDef, error) {
	nodes, err := l.LoadTree(d.Ord+"/tagDefinitions", 1, box.LoadOptions{})
	if err != nil {
		return nil, err
	}
	ns := d.Namespace
	var out []TagDef
	kids := box.Children(nodes, "")
	for _, name := range kids.Keys() {
		info, _ := get(box.Node(kids, name), "t").(string)
		def := TagDef{ID: ns + ":" + name, Name: name, Info: info, ValueType: classDefaults[info]}
		if dv := box.Node(nodes, name+"/defValue"); dv != nil {
			def.ValueType, _ = get(dv, "t").(string)
			def.Default = get(dv, "v")
		}
		if v := box.Node(nodes, name+"/validity"); v != nil {
			def.Validity = pyjson.NewObject()
			def.Validity.Set("type", get(v, "t"))
			def.Validity.Set("display", get(v, "d"))
		}
		out = append(out, def)
	}
	return out, nil
}

// GroupDefs are the tag group definitions of d (one load of its tagGroupDefinitions, depth 2).
func GroupDefs(l Loader, d Dictionary) ([]GroupDef, error) {
	nodes, err := l.LoadTree(d.Ord+"/tagGroupDefinitions", 2, box.LoadOptions{})
	if err != nil {
		return nil, err
	}
	ns := d.Namespace
	var out []GroupDef
	for _, name := range box.Children(nodes, "").Keys() {
		g := GroupDef{ID: ns + ":" + name, Name: name, Ord: SlotOrd(d.Ord + "/tagGroupDefinitions/" + name), Tags: []string{}}
		for _, tag := range box.Children(nodes, name+"/tagList").Keys() {
			g.Tags = append(g.Tags, ns+":"+tag)
		}
		out = append(out, g)
	}
	return out, nil
}

// RelationDefs are the relation ids d defines (one load of its relationDefinitions).
func RelationDefs(l Loader, d Dictionary) ([]string, error) {
	nodes, err := l.LoadTree(d.Ord+"/relationDefinitions", 1, box.LoadOptions{})
	if err != nil {
		return nil, err
	}
	ns := d.Namespace
	out := []string{}
	for _, name := range box.Children(nodes, "").Keys() {
		out = append(out, ns+":"+name)
	}
	return out, nil
}

// SlotOrd is the slot: ORD of a station:|slot: ORD (a relation's sourceOrd form).
func SlotOrd(stationOrd string) string {
	const prefix = "station:|"
	if len(stationOrd) > len(prefix) && stationOrd[:len(prefix)] == prefix {
		return stationOrd[len(prefix):]
	}
	return stationOrd
}

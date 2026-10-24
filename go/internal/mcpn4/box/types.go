package box

import "strings"

// componentTypes is COMPONENT_TYPES (B1200-G3): type spec -> true when the station creates it
// with its own add op, false when it is a slot value. An entry always wins over the naming rule.
var componentTypes = map[string]bool{
	"baja:Component": true, "baja:Folder": true, "baja:UnrestrictedFolder": true,
	"control:NullProxyExt": true, "control:PriorityLevel": false, "baja:Link": false,
	"baja:WsAnnotation": false, "modbusCore:FlexAddress": false, "bacnet:BacnetAddress": false,
}

// IsComponentType is is_component_type: an entry of the table decides, else every type outside
// the baja module is a component. A value that is not a "module:Type" string is not.
func IsComponentType(t any) bool {
	s, ok := t.(string)
	if !ok || !strings.Contains(s, ":") {
		return false
	}
	if c, ok := componentTypes[s]; ok {
		return c
	}
	module, _, _ := strings.Cut(s, ":")
	return module != "baja"
}

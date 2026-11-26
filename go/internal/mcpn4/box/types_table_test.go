package box

import "testing"

// TableComponentType reports the static table's entry only, and IsComponentType still lets an entry
// win over the naming rule and decides the rest by it.
func TestTableComponentTypeIsTheStaticTableAlone(t *testing.T) {
	for _, tc := range []struct {
		spec              any
		component, listed bool
	}{
		{"control:NullProxyExt", true, true}, {"control:PriorityLevel", false, true},
		{"alarm:LimitEnable", false, false}, {"baja:Double", false, false}, {7, false, false},
	} {
		if c, l := TableComponentType(tc.spec); c != tc.component || l != tc.listed {
			t.Errorf("%v: %v, %v; want %v, %v", tc.spec, c, l, tc.component, tc.listed)
		}
	}
	if !IsComponentType("alarm:LimitEnable") || IsComponentType("control:PriorityLevel") {
		t.Error("IsComponentType changed")
	}
}

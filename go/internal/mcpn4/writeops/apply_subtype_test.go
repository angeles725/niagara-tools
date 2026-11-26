package writeops_test

import "testing"

// A binding of another type (a vendor subclass) is not rewritten; one that still names a target by
// handle is refused, never copied onto the golden (independent verify, #934).
func TestApplyRefusesABindingLikeChildStillNamingAHandle(t *testing.T) {
	tree := apGolden().tree()
	apKid(apKid(tree, "templateConfig"), "B3").Set("t", "template:MyBinding")
	c := apStationOf(tree, apGolden().pt())
	c.types.(interface{ Set(string, any) }).Set("template:MyBinding", obj("c", []any{}, "p", "baja:Component"))
	apRefusal(t, c, apArgs(apRow("R1")), "'B3'", "template:MyBinding", "keep pointing at the golden")
}

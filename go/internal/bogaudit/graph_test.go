package bogaudit

import (
	"reflect"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/xmlsrc"
)

const graphDoc = `<?xml version='1.0' encoding='UTF-8'?>
<bajaObjectGraph version='4.0'>
 <p h='1' m='b=baja CRP=ColdRoomPan' t='b:Station'>
  <p n='Room$20A' h='c1' t='CRP:ColdRoom'>
   <p n="sp" t="b:double" v="2"/>
   <p n="mode" f="o"/>
   <p n="wsAnnotation" v="1,1,8"/>
   <p n="noValue" t="b:int"/>
   <a n="act" f="h"/>
   <p n='Box' t='b:Folder'><p n="inner" v="3"/>
    <p n='U_1' h='u1' t='CRP:EvaporatorUnit'></p>
   </p>
   <p n='L' t='b:Link'><p n="sourceOrd" v="h:u1"/><p n="sourceSlotName" v="out"/><p n="sourceSlotName" v=""/><p n="targetSlotName" v="sp"/></p>
  </p>
  <p n='w' h='w1' m='c=control' t='c:BooleanWritable'><p n='fallback' t='b:StatusBoolean'/></p>
  <p n='Self' h='s1' t='CRP:X'/>
  <p n='c1dup' h='c1' t='CRP:Y'></p>
 </p>
</bajaObjectGraph>
`

// TestBuild pins the python scanner's model: dict order and last-wins handles, paths through folders,
// value slots only as direct empty-element children with v or f, actions, links and fallbacks.
func TestBuild(t *testing.T) {
	doc, err := xmlsrc.Parse("g.xml", []byte(graphDoc))
	if err != nil {
		t.Fatal(err)
	}
	g := build(doc)
	if want := []string{"1", "c1", "u1", "w1"}; !reflect.DeepEqual(g.comps.keys, want) {
		t.Fatalf("handles = %v, want %v", g.comps.keys, want)
	}
	if n, _ := g.counts.get("c1"); n != 2 || g.comp("c1").path != "c1dup" {
		t.Errorf("c1: count %d path %q, want 2 and the last component", n, g.comp("c1").path)
	}
	u := g.comp("u1")
	if u.path != "Room$20A/Box/U_1" || u.module != "ColdRoomPan" || u.className() != "BEvaporatorUnit" {
		t.Errorf("u1 = %q %q %q", u.path, u.module, u.className())
	}
	if w := g.comp("w1"); !w.writable || !w.hasFallback || w.module != "control" {
		t.Errorf("w1 writable %v fallback %v module %q", w.writable, w.hasFallback, w.module)
	}
	links := []link{{container: "c1", hasContainer: true, srcH: "u1", srcSlot: "out", tgtSl: "sp"}}
	if !reflect.DeepEqual(g.links, links) {
		t.Errorf("links = %+v, want %+v", g.links, links)
	}
}

func TestBuildSlots(t *testing.T) {
	doc, err := xmlsrc.Parse("g.xml", []byte(graphDoc))
	if err != nil {
		t.Fatal(err)
	}
	b := &builder{g: &graph{}, prefixes: map[string]string{}}
	for _, e := range doc.Elements()[:9] { // the station and the first c1, up to its <a>
		b.start(e)
	}
	room := b.nearest()
	if want := []string{"sp", "mode"}; room == nil || !reflect.DeepEqual(room.slots.keys, want) {
		t.Fatalf("slots of %v, want %v", room, want)
	}
	if s, _ := room.slots.get("mode"); s.hasV || s.flags != "o" {
		t.Errorf("mode = %+v, want no value, flags o", s)
	}
	if f, _ := room.actions.get("act"); f != "h" {
		t.Errorf("action act = %q", f)
	}
}

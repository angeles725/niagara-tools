package xmlsrc

import (
	"reflect"
	"testing"
)

const queryDoc = `<?xml version="1.0" encoding="UTF-8"?>
<!-- header -->
<bajaObjectGraph version="4.0">
 <p n="station" t="b:Station" h="1">
  <p n="a&amp;b" v="&lt;&#x41;&#66;&gt;" t="b:Folder"/>
  <?pi skipped?>
  <p n="text">one <![CDATA[<two>]]> <x/>three&#xe9;</p>
 </p>
</bajaObjectGraph>
`

func parseQuery(t *testing.T) *Doc {
	t.Helper()
	d, err := Parse("q.bog", []byte(queryDoc))
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func TestElements(t *testing.T) {
	type row struct {
		Name                     string
		Attrs                    []Attr
		Line, Col, Depth, Parent int
	}
	var got []row
	for _, e := range parseQuery(t).Elements() {
		got = append(got, row{e.Name, e.Attrs, e.Line, e.Col, e.Depth, e.Parent})
	}
	want := []row{
		{"bajaObjectGraph", []Attr{{"version", "4.0"}}, 3, 1, 0, -1},
		{"p", []Attr{{"n", "station"}, {"t", "b:Station"}, {"h", "1"}}, 4, 2, 1, 0},
		{"p", []Attr{{"n", "a&b"}, {"v", "<AB>"}, {"t", "b:Folder"}}, 5, 3, 2, 1},
		{"p", []Attr{{"n", "text"}}, 7, 3, 2, 1},
		{"x", nil, 7, 37, 3, 3},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Elements() =\n%+v\nwant\n%+v", got, want)
	}
}

func TestPrefixedNames(t *testing.T) {
	d, err := Parse("p.px", []byte(`<px:r xmlns:px="u"><px:a px:k="v"/></px:r>`))
	if err != nil {
		t.Fatal(err)
	}
	e := d.Elements()
	if e[0].Name != "px:r" || e[0].Attrs[0].Name != "xmlns:px" || e[1].Name != "px:a" || e[1].Attrs[0].Name != "px:k" {
		t.Fatalf("prefixes not kept: %+v", e)
	}
}

func TestAttr(t *testing.T) {
	e := parseQuery(t).Elements()[2]
	if v, ok := e.Attr("v"); !ok || v != "<AB>" {
		t.Errorf("Attr(v) = %q, %v", v, ok)
	}
	if v, ok := e.Attr("missing"); ok || v != "" {
		t.Errorf("Attr(missing) = %q, %v", v, ok)
	}
}

func TestFindAndChildren(t *testing.T) {
	d := parseQuery(t)
	for _, c := range []struct {
		name string
		got  []int
		want []int
	}{
		{"Find(p)", d.Find("p"), []int{1, 2, 3}},
		{"Find(x)", d.Find("x"), []int{4}},
		{"Find(none)", d.Find("none"), nil},
		{"Children(root)", d.Children(0), []int{1}},
		{"Children(station)", d.Children(1), []int{2, 3}},
		{"Children(text)", d.Children(3), []int{4}},
		{"Children(leaf)", d.Children(4), nil},
	} {
		if !reflect.DeepEqual(c.got, c.want) {
			t.Errorf("%s = %v, want %v", c.name, c.got, c.want)
		}
	}
}

func TestText(t *testing.T) {
	d := parseQuery(t)
	for i, want := range map[int]string{3: "one <two> threeé", 2: "", 4: "", 1: "\n  \n  \n  \n "} {
		if got := d.Text(i); got != want {
			t.Errorf("Text(%d) = %q, want %q", i, got, want)
		}
	}
}

// TestEmpty pins Element.Empty: true only for a tag written as <name/>, never for <name></name>.
func TestEmpty(t *testing.T) {
	d, err := Parse("e.xml", []byte(`<r><a/><b></b><c x="/>"  /><d>t</d></r>`))
	if err != nil {
		t.Fatal(err)
	}
	var got []bool
	for _, e := range d.Elements() {
		got = append(got, e.Empty)
	}
	if want := []bool{false, true, false, true, false}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Empty = %v, want %v", got, want)
	}
}

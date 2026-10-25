package agentonincludesync

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
	"github.com/angeles725/niagara-tools/go/internal/xmlsrc"
)

func parse(t *testing.T, path, text string) *xmlsrc.Doc {
	t.Helper()
	doc, err := xmlsrc.Parse(path, []byte(text))
	if err != nil {
		t.Fatal(err)
	}
	return doc
}

// TestTestClasses: binary names join the package and the enclosing types; a sibling after a nested type is not
// nested; a type named in a comment or a string is not one.
func TestTestClasses(t *testing.T) {
	cases := []struct{ src, want string }{
		{"package a.b; class C {}", "a.b.C"},
		{"class C {}", "C"},
		{"package a; class C { static class D { class E {} } class F {} } class G {}", "a.C a.C$D a.C$D$E a.C$F a.G"},
		{"package a; // class X {}\nclass C { String s = \"class Y {}\"; }", "a.C"},
	}
	for _, c := range cases {
		lines := strings.Split(c.src, "\n")
		got := strings.Join(TestClasses(javasrc.Tokens(javasrc.NewFile("T.java", lines).Stripped())), " ")
		if got != c.want {
			t.Errorf("%q: got %q, want %q", c.src, got, c.want)
		}
	}
}

// TestTestLeaks: a srcTest source wins over a moduleTest-include.xml listing; a production class is not a leak; a
// <type> with no class is skipped; no moduleTest-include.xml (nil) checks sources only.
func TestTestLeaks(t *testing.T) {
	prod := parse(t, "p/module-include.xml",
		"<types>\n<type name=\"A\" class=\"x.BA\"/>\n<type name=\"T\" class=\"x.BT\"/>\n<type name=\"N\"/>\n<type name=\"L\" class=\"x.BL\"/>\n</types>")
	test := parse(t, "p/moduleTest-include.xml", "<types><type class=\"x.BT\"/><type class=\"x.BL\"/></types>")
	src := map[string]string{"x.BT": "p/srcTest/x/BT.java"}
	cases := []struct {
		name string
		test *xmlsrc.Doc
		src  map[string]string
		want []TestLeak
	}{
		{"both", test, src, []TestLeak{{"x.BT", 3, "source p/srcTest/x/BT.java"}, {"x.BL", 5, "also listed in p/moduleTest-include.xml"}}},
		{"listed only", test, nil, []TestLeak{{"x.BT", 3, "also listed in p/moduleTest-include.xml"}, {"x.BL", 5, "also listed in p/moduleTest-include.xml"}}},
		{"sources only", nil, src, []TestLeak{{"x.BT", 3, "source p/srcTest/x/BT.java"}}},
		{"clean", nil, nil, nil},
	}
	for _, c := range cases {
		if got := TestLeaks(prod, c.test, c.src); !slices.Equal(got, c.want) {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
}

// TestPartTestLeaks: the file-system entry verify-module uses reads the part's three inputs; no module-include.xml is
// no leak, and a malformed moduleTest-include.xml is the error.
func TestPartTestLeaks(t *testing.T) {
	write := func(dir, rel, text string) {
		p := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	part := t.TempDir()
	if got, err := PartTestLeaks(part); err != nil || got != nil {
		t.Fatalf("no include: got %v, %v", got, err)
	}
	write(part, "module-include.xml", "<types>\n<type class=\"x.BT\"/>\n</types>\n")
	write(part, "srcTest/x/BT.java", "package x;\npublic class BT {}\n")
	got, err := PartTestLeaks(part)
	if want := []TestLeak{{"x.BT", 2, "source " + part + "/srcTest/x/BT.java"}}; err != nil || !slices.Equal(got, want) {
		t.Fatalf("srcTest leak: got %v, %v; want %v", got, err, want)
	}
	write(part, "moduleTest-include.xml", "<types>\n")
	if _, err := PartTestLeaks(part); err == nil || !strings.Contains(err.Error(), "moduleTest-include.xml:2:1") {
		t.Fatalf("malformed: got %v, want an xmlsrc position error", err)
	}
}

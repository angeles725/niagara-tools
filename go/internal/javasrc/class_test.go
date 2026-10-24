package javasrc

import (
	"fmt"
	"strings"
	"testing"
)

func toksOf(src string) []Token {
	lines, _ := Strip(strings.Split(src, "\n"))
	return Tokens(lines)
}

// TestClasses: each named type with its kind, superclass simple name and name line; a class literal, an
// anonymous class and a commented-out declaration are not one.
func TestClasses(t *testing.T) {
	for _, tc := range []struct{ name, src, want string }{
		{"extends simple", "public class BAcmeDevice\n  extends BNDevice\n  implements Runnable {\n}", "class BAcmeDevice:BNDevice@1"},
		{"qualified and generic superclass", "class A extends com.example.drv.BBase<String, Integer> {}", "class A:BBase@1"},
		{"generic class header", "class A<T extends Comparable<T>> extends B<T> {}", "class A:B@1"},
		{"no superclass", "final class A implements I {}", "class A:@1"},
		{"interface extends is not a superclass", "interface I extends J, K {}", "interface I:@1"},
		{"nested and enum", "class A extends X {\n  static class B extends A { }\n  enum E { P, Q }\n}", "class A:X@1 class B:A@2 enum E:@3"},
		{"class literal and anonymous class", "class A {\n  Class<?> c = A.class;\n  Object o = new Object() { };\n}", "class A:@1"},
		{"declaration in a comment or a string", "// class B extends C {}\nclass A { String s = \"class D extends E {}\"; }", "class A:@2"},
		{"record header", "record R(int a) implements I { }", "record R:@1"},
	} {
		var got []string
		for _, c := range Classes(toksOf(tc.src)) {
			got = append(got, fmt.Sprintf("%s %s:%s@%d", c.Kind, c.Name, c.Extends, c.Line+1))
		}
		if g := strings.Join(got, " "); g != tc.want {
			t.Errorf("%s:\n got %s\nwant %s", tc.name, g, tc.want)
		}
	}
}

// TestClassBody: the body range holds the tokens inside the braces only.
func TestClassBody(t *testing.T) {
	toks := toksOf("class A { int x; }")
	c := Classes(toks)[0]
	var got []string
	for _, tk := range toks[c.Body[0]:c.Body[1]] {
		got = append(got, tk.Text)
	}
	if g := strings.Join(got, " "); g != "int x ;" {
		t.Errorf("body tokens %q", g)
	}
}

// TestFields: the fields declared directly in a class body, with their modifiers, type and initializer;
// methods, constructors, initializer blocks and nested types are not fields.
func TestFields(t *testing.T) {
	for _, tc := range []struct{ name, body, want string }{
		{"static generic with initializer", "private static final Set<String> names = new HashSet<>();",
			"static Set<String> names=new HashSet<>()@1"},
		{"instance field", "private final Map<String, BObject> m = new HashMap<>();", "Map<String, BObject> m=new HashMap<>()@1"},
		{"no initializer, two declarators", "static int a, b;", "static int a@1"},
		{"annotation and qualified type", "@Deprecated(\"x\") protected static java.util.List<String> q;", "static java.util.List<String> q@1"},
		{"array type", "static String[] names = {\"a\", \"b\"};", "static String[] names={\"a\",\"b\"}@1"},
		{"methods, constructor and blocks", "A() { }\nstatic { init(); }\nvoid f() { int x = 1; }\nabstract int g();\nstatic int n = 3;",
			"static int n=3@5"},
		{"anonymous class initializer", "static final Map<String, Integer> m = new HashMap<String, Integer>() {{ put(\"a\", 1); }};\nint z;",
			"static Map<String, Integer> m=new HashMap<String,Integer>(){{put(\"a\",1);}}@1 int z@2"},
		{"nested class fields are its own", "static class B { static int y; }\nint x;", "int x@2"},
		{"generic method is not a field", "static <T> List<T> f(T t) { return null; }\nstatic <T> List<T> g(T t);", ""},
	} {
		toks := toksOf("class A {\n" + tc.body + "\n}")
		var got []string
		for _, f := range Fields(toks, Classes(toks)[0].Body) {
			s := ""
			if f.Static {
				s = "static "
			}
			s += f.Type + " " + f.Name
			if f.Init[1] > f.Init[0] {
				var b strings.Builder
				for k, tk := range toks[f.Init[0]:f.Init[1]] {
					if k > 0 && tk.Kind == TokIdent && toks[f.Init[0]+k-1].Kind == TokIdent {
						b.WriteString(" ")
					}
					b.WriteString(tk.Text)
				}
				s += "=" + b.String()
			}
			got = append(got, fmt.Sprintf("%s@%d", s, f.Line))
		}
		if g := strings.Join(got, " "); g != tc.want {
			t.Errorf("%s:\n got %s\nwant %s", tc.name, g, tc.want)
		}
	}
}

// TestDescendants: direct and module-local chains reach a base; an unrelated class or a cycle does not.
func TestDescendants(t *testing.T) {
	toks := toksOf("class A extends BNDevice {}\nclass B extends A {}\nclass C extends B {}\nclass D extends Object {}\n" +
		"class E extends F {}\nclass F extends E {}")
	got := Descendants(Classes(toks), "BNDevice", "BNNetwork")
	if len(got) != 3 || !got["A"] || !got["B"] || !got["C"] {
		t.Errorf("Descendants = %v, want A B C", got)
	}
}

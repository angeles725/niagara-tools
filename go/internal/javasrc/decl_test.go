package javasrc

import (
	"fmt"
	"strings"
	"testing"
)

func showDecls(src string) string {
	lines, _ := Strip(strings.Split(src, "\n"))
	toks := Tokens(lines)
	var b []string
	for _, d := range Decls(toks) {
		o := ""
		if d.Override {
			o = "@"
		}
		if d.Anonymous {
			o += "~"
		}
		b = append(b, fmt.Sprintf("%s%s.%s(%s)@%d#%d", o, d.Class, d.Name, strings.Join(d.Params, ","), d.Line+1, Statements(toks[d.Body[0]:d.Body[1]])))
	}
	return strings.Join(b, " ")
}

// TestDecls: method declarations with their class, parameter types, @Override and body statement count.
func TestDecls(t *testing.T) {
	for _, tc := range []struct{ name, src, want string }{
		{"method and constructor", "class A {\n  A() { x = 1; }\n  private static int f(byte[] b, final int n) { a(); b(); return 1; }\n}",
			"A.A()@2#1 A.f(byte[],int)@3#3"},
		{"generics, varargs, c-style array, annotations", "class A {\n  <T> void g(@Nullable Map<String, List<T>> m, String... s, int a[]) throws IOException, X { }\n}",
			"A.g(Map<String,List<T>>,String...,int[])@2#0"},
		{"override", "class A {\n  @Override\n  public String toString() { return \"a\"; }\n  @SuppressWarnings(\"x\") @Override void h() {}\n}",
			"@A.toString()@3#1 @A.h()@4#0"},
		{"nested, inner and anonymous classes", "class A {\n  class B { void f() {} }\n  void g() { Runnable r = new Runnable() { public void run() { go(); } }; }\n}\nenum E { X; void k() {} }",
			"B.f()@2#0 A.g()@3#2 ~Runnable.run()@3#1 E.k()@5#0"},
		{"for header counts once", "class A {\n  void f() { for (int i = 0; i < n; i++) { a(); } }\n}", "A.f()@2#1"},
		{"control blocks and lambdas are not methods", "class A {\n  void f() { if (x) { y(); } while (z) { w(); } r(() -> { q(); }); synchronized (this) { s(); } }\n  static { init(); }\n}",
			"A.f()@2#5"},
		{"interface method without body", "interface I {\n  void f(int a);\n  default void g() { f(1); }\n}", "I.g()@3#1"},
		{"class literal is not a declaration", "class A {\n  void f() { Class<?> c = A.class; }\n}", "A.f()@2#1"},
		{"text block holding a method", "class A {\n  String s = \"\"\"\n    void f() { a(); }\n    \"\"\";\n}", ""},
		{"record", "record R(int a) {\n  int twice() { return a * 2; }\n}", "R.twice()@2#1"},
	} {
		if got := showDecls(tc.src); got != tc.want {
			t.Errorf("%s:\n got %s\nwant %s", tc.name, got, tc.want)
		}
	}
}

// TestShape: the normalized token sequence keeps identifiers and literals, drops layout and comments.
func TestShape(t *testing.T) {
	toks := func(s string) []Token { l, _ := Strip(strings.Split(s, "\n")); return Tokens(l) }
	a := Shape(toks("a(1);  // x\n  b = \"s\";"))
	if b := Shape(toks("a( 1 ); /* y */ b=\"s\";")); a != b {
		t.Errorf("layout and comments changed the shape: %q vs %q", a, b)
	}
	if c := Shape(toks("a(1); c = \"s\";")); a == c {
		t.Errorf("a renamed identifier kept the shape: %q", c)
	}
	if d := Shape(toks("a(1); b = \"t\";")); a == d {
		t.Errorf("a changed literal kept the shape: %q", d)
	}
	if e := Shape(toks("ab(1);")); e == Shape(toks("a b(1);")) {
		t.Errorf("token boundaries are lost: %q", e)
	}
}

// TestDeclModifiers: private and static are read from the header, never from an annotation's arguments.
func TestDeclModifiers(t *testing.T) {
	src := "class A {\n  private static int a() { return 1; }\n  static int b() { return 1; }\n  private int c() { return 1; }\n" +
		"  public int d() { return 1; }\n  @Tag(kind = \"static\", v = private_) int e() { return 1; }\n}"
	lines, _ := Strip(strings.Split(src, "\n"))
	var got []string
	for _, d := range Decls(Tokens(lines)) {
		got = append(got, fmt.Sprintf("%s:%v%v", d.Name, d.Private, d.Static))
	}
	if want := "a:truetrue b:falsetrue c:truefalse d:falsefalse e:falsefalse"; strings.Join(got, " ") != want {
		t.Errorf("got %s\nwant %s", strings.Join(got, " "), want)
	}
}

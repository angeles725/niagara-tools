package javasrc

import (
	"reflect"
	"strings"
	"testing"
)

// The cases port tests/method-boundary.bats (MB1-MB9) and add edge cases; every input here is also
// run through the awk lib by TestDifferentialAwk, so the expected values are awk's, not a guess.

type stripCase struct {
	name     string
	in, want []string
}

var stripCases = []stripCase{
	{"MB1 // inside a string is code",
		[]string{`String u = "tcp://host"; doWork();`},
		[]string{`String u = "tcp://host"; doWork();`}},
	{"MB2 /* inside a string opens no comment",
		[]string{`String s = "/*"; a();`, `b();`, `c(); // x */`},
		[]string{`String s = "/*"; a();`, `b();`, `c(); `}},
	{"MB3 char literals / \" and \\'",
		[]string{`char c = '/'; d(); // t`, `char q = '"'; e(); // t`, `char a = '\''; f(); // t`},
		[]string{`char c = '/'; d(); `, `char q = '"'; e(); `, `char a = '\''; f(); `}},
	{"MB4 escaped quotes and backslashes",
		[]string{`String s = "say \"hi\" // not"; g(); // real`, `String b = "a\\"; h(); // c`},
		[]string{`String s = "say \"hi\" // not"; g(); `, `String b = "a\\"; h(); `}},
	{"MB5 text block across lines",
		[]string{`String t = """`, `  http://x /* not a comment`, `  a \""" still inside`, `  """; k(); // c`, `m(); // d`},
		[]string{`String t = """`, `  http://x /* not a comment`, `  a \""" still inside`, `  """; k(); `, `m(); `}},
	{"MB6 real comments",
		[]string{`x(); // c`, `/* a`, `b */ y();`, `p(); /* q */ r();`, `/** "doc" // */ z();`},
		[]string{`x(); `, ``, ` y();`, `p();  r();`, ` z();`}},
	{"MB7 unterminated string ends at its line",
		[]string{`String s = "abc`, `n(); // c`, ``, `"x"; /* open`, `still comment "`, `*/ q();`},
		[]string{`String s = "abc`, `n(); `, ``, `"x"; `, ``, ` q();`}},
	{"one-line text block holding an escaped triple quote",
		[]string{`String t = """ a \""" b """; c(); // d`},
		[]string{`String t = """ a \""" b """; c(); `}},
	{"/*/ opens a comment that */ closes",
		[]string{`a(); /*/ b(); */ c();`},
		[]string{`a();  c();`}},
	{"backslash at end of an unterminated string",
		[]string{`String s = "a\`, `b(); // c`},
		[]string{`String s = "a\`, `b(); `}},
	{"comment right after a closing quote",
		[]string{`f("a")//x`, `g('b')/*y*/;`},
		[]string{`f("a")`, `g('b');`}},
	{"CR is kept like any other byte",
		[]string{"a(); // c\r", "b();\r"},
		[]string{"a(); ", "b();\r"}},
	{"non-ASCII text inside and outside literals",
		[]string{`String s = "año // x"; ñ(); // é`, `char c = 'é'; /* ü */ d();`},
		[]string{`String s = "año // x"; ñ(); `, `char c = 'é';  d();`}},
}

func TestStrip(t *testing.T) {
	for _, tc := range stripCases {
		t.Run(tc.name, func(t *testing.T) {
			got, _ := Strip(tc.in)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("Strip(%q)\n got %q\nwant %q", tc.in, got, tc.want)
			}
		})
	}
}

// MB8: the end state is 0 closed, 1 inside /* */, 2 inside a text block.
func TestStripEndState(t *testing.T) {
	for in, want := range map[string]EndState{
		`a(); "x"; // c`:  Closed,
		`b(); /* open`:    OpenComment,
		`String t = """`:  OpenTextBlock,
		`"""a""" /* x */`: Closed,
	} {
		if _, got := Strip([]string{in}); got != want {
			t.Errorf("Strip(%q) state = %d, want %d", in, got, want)
		}
	}
	if _, got := Strip(nil); got != Closed {
		t.Errorf("Strip(nil) state = %d, want Closed", got)
	}
}

type methodsCase struct {
	name string
	src  string
	want []Method
}

var methodsCases = []methodsCase{
	{"MB9 braces inside string and char literals", `class A {
  void a() {
    String s = "}";
    char c = '}';
    x();
  }
  void b() {
    y();
  }
}`, []Method{{"a", 2, 6}, {"b", 7, 9}}},
	{"keywords and initializer blocks are not methods", `class K {
  static {
    if (ready) {
      go();
    }
    try {
    } catch (Exception e) {
    }
    for (int i = 0; i < n; i++) {
    }
    while (y) { z(); }
    switch (v) {
    }
  }
  void after() { x(); }
}`, []Method{{"after", 15, 15}}},
	{"annotation stops the backward scan; brace on its own line", `class C {
  @Override
  public void multi(int a,
                    int b)
  {
    x();
  }
  @Deprecated
  {
    init();
  }
  int f;
  {
    g();
  }
}`, []Method{{"multi", 5, 7}}},
	{"one-liner accessors are skipped, multi-line ones kept", `class D {
  int getX() { return x; }
  boolean isOn() { return on; }
  void setY(int y) { this.y = y; }
  void tick() { n++; }
  boolean isDirty() {
    return d;
  }
  int get() { return 0; }
  int get_x() { return 1; }
  int getter() { return 1; }
}`, []Method{{"tick", 5, 5}, {"isDirty", 6, 8}, {"get", 9, 9}, {"getter", 11, 11}}},
	{"lambdas and anonymous classes stay inside their method", `class E {
  void schedule() {
    run(() -> {
      if (a) { b(); }
    });
    Object o = new Object() {
      public String toString() { return "}"; }
    };
  }
  void next() {}
}`, []Method{{"schedule", 2, 9}, {"next", 10, 10}}},
	{"generics and throws", `class F {
  public <T> List<T> load(String p) throws IOException, X {
    return null;
  }
}`, []Method{{"load", 2, 4}}},
	{"class-body guard: fields and array initializers", `class G {
  @A({"x"}) int f;
  int[] a = {1, 2};
  String t = "void z() {";
}`, nil},
	{"anonymous class field is named after its type (awk quirk kept)", `class H {
  Runnable r = new Runnable() {
    public void run() { go(); }
  };
}`, []Method{{"Runnable", 2, 4}}},
	{"a parameter annotation with parentheses hides the method (awk quirk kept)", `class P {
  void f(@Named("x") String s) {
    g();
  }
  void h(int[] a) {
  }
}`, []Method{{"h", 5, 6}}},
	{"text block with braces inside a method", `class T {
  String q() {
    return """
      { "a": "}" }
      """;
  }
}`, []Method{{"q", 2, 6}}},
	{"unterminated string and a stripped comment brace", `class U {
  void u() { // }
    String s = "abc {
    x();
  }
  void w() /* { */ {
  }
}`, []Method{{"u", 2, 5}, {"w", 6, 7}}},
	{"backward scan reaches 20 lines above a lone brace", "class S {\n  void far(\n" +
		strings.Repeat("      int p,\n", 19) + "  {\n  }\n}", []Method{{"far", 22, 23}}},
	{"backward scan stops after 20 lines", "class S {\n  void tooFar(\n" +
		strings.Repeat("      int p,\n", 20) + "  {\n  }\n}", nil},
	{"nested class methods", `class O {
  static class In {
    void deep() {
      d();
    }
  }
}`, []Method{{"deep", 3, 5}}},
}

// goOnlyMethodsCases are the declared differences of Methods from mb_parse (package doc): TestMethods runs
// them, TestDifferentialAwk does not feed them to the awk lib.
var goOnlyMethodsCases = []methodsCase{
	{"a body ends where its own brace closes, not the deepest block of its first line", `class B {
  void x() { if (a) {
      y();
    }
    z();
  }
  void w(String s) {	try	{ p(s); }
    catch (Exception e) { q(); }
  }
}`, []Method{{"x", 2, 6}, {"w", 7, 9}}},
	{"wrapped signature closing on the brace line", `class W {
  void m(int a,
      int b) {
    x();
  }
  public void t(int a,
      int b)
      throws Exception {
    y();
  }
  void z(int a, int b) {
  }
}`, []Method{{"m", 3, 5}, {"t", 8, 10}, {"z", 11, 12}}},
	{"a wrapped ) { after a statement is not a signature", `class S {
  int f = g(1,
      2);
  void a(int x,
      int y) {
  }
}`, []Method{{"a", 5, 6}}},
}

func methodsOf(src string) []Method {
	stripped, _ := Strip(strings.Split(src, "\n"))
	return Methods(stripped)
}

func TestMethods(t *testing.T) {
	for _, tc := range append(append([]methodsCase{}, methodsCases...), goOnlyMethodsCases...) {
		t.Run(tc.name, func(t *testing.T) {
			if got := methodsOf(tc.src); !reflect.DeepEqual(got, tc.want) {
				t.Errorf("Methods\n got %v\nwant %v", got, tc.want)
			}
		})
	}
}

func TestLines(t *testing.T) {
	for in, want := range map[string][]string{
		"":           nil,
		"\n":         {""},
		"a\nb":       {"a", "b"},
		"a\n\n":      {"a", ""},
		"a\r\nb\r\n": {"a\r", "b\r"},
	} {
		got, err := Lines(strings.NewReader(in))
		if err != nil || !reflect.DeepEqual(got, want) {
			t.Errorf("Lines(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
}

// TestScopes: Scopes is Methods plus the lambda bodies opened outside a method; a lambda inside a method
// stays part of the method's body, and Enclosing finds the scope of a line.
func TestScopes(t *testing.T) {
	src := `class L {
  Runnable r = () -> {
    a();
  };
  Runnable s = () ->
  {
    b();
  };
  void m() {
    Runnable q = () -> {
      c();
    };
  }
  int f = 1;
  Object o = new Object() {
    public String toString() {
      return "x";
    }
  };
}`
	stripped, _ := Strip(strings.Split(src, "\n"))
	want := []Method{{"lambda", 2, 4}, {"lambda", 6, 8}, {"m", 9, 13}, {"Object", 15, 19}}
	got := Scopes(stripped)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Scopes\n got %v\nwant %v", got, want)
	}
	if ms := Methods(stripped); !reflect.DeepEqual(ms, []Method{{"m", 9, 13}, {"Object", 15, 19}}) {
		t.Errorf("Methods changed: %v", ms)
	}
	for line, w := range map[int]string{3: "lambda", 7: "lambda", 11: "m", 17: "Object", 14: "", 1: ""} {
		m, ok := Enclosing(got, line)
		if ok != (w != "") || m.Name != w {
			t.Errorf("Enclosing(%d) = %v %v, want %q", line, m, ok, w)
		}
	}
	f := NewFile("L.java", strings.Split(src, "\n"))
	if !reflect.DeepEqual(f.Scopes(), want) {
		t.Errorf("File.Scopes() = %v", f.Scopes())
	}
	open, _ := Strip([]string{"class U {", "  void a() {", "    try {", "      x();"})
	if got, ms := Scopes(open), Methods(open); !reflect.DeepEqual(got, []Method{{"a", 2, 4}}) || ms != nil {
		t.Errorf("unbalanced file: Scopes %v (want a 2-4), Methods %v (want none)", got, ms)
	}
}

package nullcontext

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
	"github.com/angeles725/niagara-tools/go/internal/lint"
)

const bqlMsg = "bql: ORD resolved without the caller's Context (pass get(base, cx)): "

// The NCW* cases port tests/lint-null-context-write.bats one to one; the I293 cases cover issue #293.
// want lists "<line>: <message>" in report order (every write row, then every bql row).
var cases = []struct {
	name, path string
	src        []string
	want       []string
}{
	{"NCW1 a write with a real Context is clean", "M/src/A.java",
		[]string{"class A { void f(Context cx){ parent.set(prop, val, cx); } }"}, nil},
	{"NCW2 set(...,null) WARNs", "M/src/A.java",
		[]string{"class A { void f(){ parent.set(prop, val, null); } }"},
		[]string{"1: null-Context write: class A { void f(){ parent.set(prop, val, null); } }"}},
	{"NCW3 invoke(...,null) WARNs", "M/src/A.java",
		[]string{"class A { void f(){ comp.invoke(act, arg, null); } }"},
		[]string{"1: null-Context write: class A { void f(){ comp.invoke(act, arg, null); } }"}},
	{"NCW write variants and leading whitespace trimmed", "M/src/A.java",
		[]string{"\t  c.setInt(p, 1,null );", "c.setBoolean(p, true, cx);", " c.setDouble(p, 1.0, null)"},
		[]string{"1: null-Context write: c.setInt(p, 1,null );", "3: null-Context write: c.setDouble(p, 1.0, null)"}},
	{"NCW-bql1 BOrd.make(\"bql:...\").get(base) WARNs", "M/src/A.java",
		[]string{"class A { BITable<?> q() {",
			`  return (BITable<?>) BOrd.make("station:|slot:/|bql:select * from control:NumericPoint").get(Sys.getStation());`, "} }"},
		[]string{"2: " + bqlMsg + `return (BITable<?>) BOrd.make("station:|slot:/|bql:select * from control:NumericPoint").get(Sys.getStation());`}},
	{"NCW-bql2 a multi-line .get(base, null) and a bare .get() WARN", "M/src/A.java",
		[]string{"class A { void q() {", "  Object t = BOrd.make(", `      "bql:select name from baja:Folder")`,
			"      .get(base, null);", `  Object u = BOrd.make("bql:select *").get();`, "} }"},
		[]string{"3: " + bqlMsg + `"bql:select name from baja:Folder")`, "5: " + bqlMsg + `Object u = BOrd.make("bql:select *").get();`}},
	{"NCW-bql-ok get(base, cx) and resolve(base, cx).get() are clean", "M/src/A.java",
		[]string{"class A { void q(Context cx) {", `  Object t = BOrd.make("bql:select *").get(base, cx);`,
			`  Object u = BOrd.make("bql:select *").resolve(base, cx).get();`, "} }"}, nil},
	{"NCW-bql-ok a -wb part is skipped by the bql rule", "M/src/Foo-wb/src/W.java",
		[]string{`class W { Object t = BOrd.make("bql:select *").get(base); }`}, nil},
	{"NCW-literal a // inside a string does not hide a write; a quoted write is not code", "M/src/A.java",
		[]string{`class A { void f(){ String u = "http://h"; parent.set(prop, val, null); }`, `  String s = "c.set(a, b, null)"; }`},
		[]string{`1: null-Context write: class A { void f(){ String u = "http://h"; parent.set(prop, val, null); }`}},
	{"NCW-comment a write in a comment is not reported", "M/src/A.java",
		[]string{"class A {", "  // parent.set(prop, val, null);", "  /* comp.invoke(act, arg, null); */", "}"}, nil},
	{"NCW-wb-write a -wb null-Context write still WARNs", "M/src/Foo-wb/src/W.java",
		[]string{"class W { void f(){ comp.invoke(act, arg, null); } }"},
		[]string{"1: null-Context write: class W { void f(){ comp.invoke(act, arg, null); } }"}},
	{"NCW-bql-unrelated a one-argument get outside the ORD chain is clean", "M/src/A.java",
		[]string{"class A { void q(Context cx) {", `  Object t = BOrd.make("bql:select " + "*").get(base, cx) == null ? params.get("x") : null;`, "} }"}, nil},
	{"NCW-bql text block and char literals are blanked", "M/src/A.java",
		[]string{`String t = """`, `  x.set(a, b, null)`, `  """; char c = ')'; Object o = BOrd.make("bql:x").get(base, cx);`}, nil},
	{"NCW CRLF lines keep their CR in the row", "M/src/A.java",
		[]string{"c.set(a, b, null);\r", `BOrd.make("bql:x").get();` + "\r"},
		[]string{"1: null-Context write: c.set(a, b, null);\r", "2: " + bqlMsg + `BOrd.make("bql:x").get();` + "\r"}},
	// Truth audit 2026-10-22 T2: the write is judged on its whole statement (its own argument list).
	{"T2 a multi-line write with a null last argument WARNs at the call's line", "M/src/A.java",
		[]string{"class A { void f() {", "  c.set(p,", "      BDouble.make(2),", "      null);", "} }"},
		[]string{"2: null-Context write: c.set(p,"}},
	{"T2 a (Context) null last argument WARNs", "M/src/A.java",
		[]string{"c.set(p, BDouble.make(2), (Context) null);", "c.invoke(a, v, (Context)null);"},
		[]string{"1: null-Context write: c.set(p, BDouble.make(2), (Context) null);", "2: null-Context write: c.invoke(a, v, (Context)null);"}},
	{"T2 a null inside a nested argument or in another call on the line is clean", "M/src/A.java",
		[]string{"c.set(p, foo(x, null));", "a.set(p, v); b.foo(x, null);", "c.set(p, v, nullValue);"}, nil},
	{"T2 two null writes on one line give one row", "M/src/A.java",
		[]string{"c.set(p, v, null); d.set(q, w, null);"},
		[]string{"1: null-Context write: c.set(p, v, null); d.set(q, w, null);"}},
	// Issue #293 (2): resolve is judged like get.
	{"I293-2 resolve(base) WARNs", "M/src/A.java",
		[]string{`OrdTarget t = BOrd.make("bql:select *").resolve(base);`},
		[]string{"1: " + bqlMsg + `OrdTarget t = BOrd.make("bql:select *").resolve(base);`}},
	{"I293-2 resolve() WARNs", "M/src/A.java",
		[]string{`OrdTarget t = BOrd.make("bql:select *").resolve();`},
		[]string{"1: " + bqlMsg + `OrdTarget t = BOrd.make("bql:select *").resolve();`}},
	{"I293-2 resolve(base, null) WARNs", "M/src/A.java",
		[]string{`OrdTarget t = BOrd.make("bql:select *").resolve(base, null);`},
		[]string{"1: " + bqlMsg + `OrdTarget t = BOrd.make("bql:select *").resolve(base, null);`}},
	// Issue #293 (1): the null-Context check is bounded to the get/resolve call's own parentheses.
	{"I293-1 a null argument after the call closes is clean", "M/src/A.java",
		[]string{`foo(BOrd.make("bql:select *").get(base, cx), null);`}, nil},
	{"I293-1 a null inside a nested call argument is clean", "M/src/A.java",
		[]string{`Object t = BOrd.make("bql:select *").get(base, ctx(a, null));`}, nil},
	{"I293-1 a null last argument after a nested call still WARNs", "M/src/A.java",
		[]string{`Object t = BOrd.make("bql:select *").get(base(a, b), null);`},
		[]string{"1: " + bqlMsg + `Object t = BOrd.make("bql:select *").get(base(a, b), null);`}},
}

func TestCheck(t *testing.T) {
	r := Spec().Rule
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fds, err := r.Check(javasrc.NewFile(tc.path, tc.src))
			if err != nil {
				t.Fatalf("Check error: %v", err)
			}
			var got []string
			for _, f := range fds {
				got = append(got, fmt.Sprintf("%d: %s", f.Line, f.Msg))
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("Check\n got %q\nwant %q", got, tc.want)
			}
		})
	}
}

func TestSpec(t *testing.T) {
	s := Spec()
	if s.Rule.ID() != "lint-null-context-write" || s.Usage != "usage: lint-null-context-write.sh <src-root>" ||
		!reflect.DeepEqual(s.Select.Names, []string{"*.java"}) || len(s.Select.Prune) != 0 ||
		s.Mode != lint.Raw || s.Rule.Severity() != lint.Warn {
		t.Errorf("Spec() = %+v", s)
	}
}

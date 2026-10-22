package servicetypes

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
	"github.com/angeles725/niagara-tools/go/internal/lint"
)

const (
	wantFailMsg  = "getServiceTypes() registers no type (empty array or null) -- the service is never registered; return new Type[] { TYPE } (METHODOLOGY.md § rt)"
	wantUnresMsg = "getServiceTypes() return not resolved in this file -- check it returns a non-empty array holding TYPE"
)

func wantOtherMsg(own string) string {
	return "getServiceTypes() does not return the service's own TYPE -- Sys.getService(" + own +
		".TYPE) will not find it; add TYPE unless that is intended"
}

// svc mirrors the bats helper: a minimal service whose getServiceTypes() body is one line, with an optional
// extra member line before it (the declaration is then on line 5, else line 4).
func svc(class, body string, member ...string) []string {
	src := []string{"package com.x;", "public class " + class + " extends BAbstractService {"}
	for _, m := range member {
		src = append(src, "  "+m)
	}
	return append(src, "  @Override", "  public Type[] getServiceTypes() {", "    "+body, "  }", "}")
}

// gst mirrors the bats helper: getServiceTypes() runs one statement, then returns { TYPE } (declaration line 3).
func gst(class, stmt string) []string {
	return []string{"package com.x;", "public class " + class + " extends BAbstractService {",
		"  public Type[] getServiceTypes() {", "    " + stmt, "    return new Type[] { TYPE };", "  }", "}"}
}

// method wraps body lines in a service class B (declaration line 3).
func method(body ...string) []string {
	src := []string{"package com.x;", "public class B extends BAbstractService {", "  public Type[] getServiceTypes() {"}
	src = append(src, body...)
	return append(src, "  }", "}")
}

func wantFail(line int) string  { return fmt.Sprintf("FAIL %d: %s", line, wantFailMsg) }
func wantUnres(line int) string { return fmt.Sprintf("WARN %d: %s", line, wantUnresMsg) }
func wantOther(line int, own string) string {
	return fmt.Sprintf("WARN %d: %s", line, wantOtherMsg(own))
}

// The ST* cases port tests/lint-service-types.bats file by file; TB* pin the literal blanker's text-block and
// escape handling (issue #286); Q* pin awk behaviours the port keeps byte-identical.
var cases = []struct {
	name string
	src  []string
	want []string
}{
	{"ST1 new Type[0] FAILs", svc("BZero", "return new Type[0];"), []string{wantFail(4)}},
	{"ST1 new Type[] {} FAILs", svc("BBraces", "return new Type[] {};"), []string{wantFail(4)}},
	{"ST1 null FAILs", svc("BNull", "return null;"), []string{wantFail(4)}},
	{"ST2 one-line { TYPE } is clean", svc("BOne", "return new Type[] { TYPE };"), nil},
	{"ST2 multi-line initializer holding TYPE is clean", []string{"package com.x;", "public class BMulti extends BAbstractService {",
		"  public Type[] getServiceTypes() {", "    return new Type[] {", "      TYPE,", "      BIMulti.TYPE", "    };", "  }", "}"}, nil},
	{"ST3 a field = { TYPE } is clean", svc("BField", "return serviceTypes;", "private static final Type[] serviceTypes = { TYPE };"), nil},
	{"ST3 a field = new Type[0] FAILs", svc("BFieldEmpty", "return serviceTypes;", "private static final Type[] serviceTypes = new Type[0];"), []string{wantFail(5)}},
	{"ST4 a foreign-only array WARNs", svc("BOther", "return new Type[] { BIOther.TYPE };"), []string{wantOther(4, "BOther")}},
	{"ST4 Own.TYPE counts as own", svc("BQual", "return new Type[] { BQual.TYPE, BIQual.TYPE };"), nil},
	{"ST5 a super call WARNs unresolved", svc("BSuper", "return super.getServiceTypes();"), []string{wantUnres(4)}},
	{"ST5 an inherited field WARNs unresolved", svc("BInh", "return INHERITED;"), []string{wantUnres(4)}},
	{"ST6 an interface declaration is out of scope", []string{"package com.x;", "public interface IFoo {", "  Type[] getServiceTypes();", "}"}, nil},
	{"ST6 a file without the method gives no row", []string{"package com.x;", "public class BPlain extends BComponent {", "}"}, nil},
	{"ST6 a commented-out stub gives no row", method("    // return new Type[0];", "    return new Type[] { TYPE };"), nil},
	{"ST8 this.getType() is own", svc("BGet", "return new Type[] { this.getType() };"), nil},
	{"ST8 a parenthesized array is own", svc("BPar", "return (new Type[] { TYPE });"), nil},
	{"ST8 an Own.field reference is resolved", svc("BRef", "return BRef.TYPES;", "public static final Type[] TYPES = { TYPE };"), nil},
	{"ST9 a static-block assignment is resolved", []string{"package com.x;", "public class BLate extends BAbstractService {",
		"  private static Type[] serviceTypes;", "  static { BLate.serviceTypes = new Type[0]; }",
		"  public Type[] getServiceTypes() {", "    return BLate.serviceTypes;", "  }", "}"}, []string{wantFail(5)}},
	{"ST9 a cloned field is resolved", svc("BClone", "return (Type[]) types.clone();", "private static final Type[] types = { TYPE };"), nil},
	{"ST10 a later null return FAILs", method("    if (ready) return new Type[] { TYPE };", "    return null;"), []string{wantFail(3)}},
	{"ST10 a later foreign-only return WARNs", method("    if (ready) return new Type[] { TYPE };", "    return new Type[] { BIMix.TYPE };"), []string{wantOther(3, "B")}},
	{"ST11 an assignment after another statement is read from the assignment", []string{"package com.x;",
		"public class BLine extends BAbstractService {", "  private static Type[] types;", "  static { int k = 0; types = new Type[0]; }",
		"  public Type[] getServiceTypes() {", "    return types;", "  }", "}"}, []string{wantFail(5)}},
	{"ST12 a block lambda's return is nested", method("    Supplier<Type[]> s = () -> { return null; };", "    return new Type[] { TYPE };"), nil},
	{"ST12 an anonymous class's return is nested", method("    Callable<Type[]> c = new Callable<Type[]>() {",
		"      public Type[] call() { return new Type[0]; }", "    };", "    return new Type[] { TYPE };"), nil},
	{"ST12 a local class's return is nested", method("    class Helper { Type[] none() { return null; } }", "    return new Type[] { TYPE };"), nil},
	{"ST12 the method's returns around a nested scope count", method("    Supplier<Type[]> s = () -> { return new Type[] { TYPE }; };",
		"    if (ready) { return null; }", "    return new Type[] { TYPE };"), []string{wantFail(3)}},
	{"ST12 a switch-rule block is the method's scope", method("    switch (m) { case 1 -> { return null; } default -> { } }",
		"    return new Type[] { TYPE };"), []string{wantFail(3)}},
	{"ST13 empty beats other", method("    if (a) return new Type[] { BIEo.TYPE };", "    return null;"), []string{wantFail(3)}},
	{"ST13 other beats unresolved", method("    if (a) return super.getServiceTypes();", "    return new Type[] { BIOu.TYPE };"), []string{wantOther(3, "B")}},
	{"ST13 unresolved beats own", method("    if (a) return new Type[] { TYPE };", "    return super.getServiceTypes();"), []string{wantUnres(3)}},
	{"ST14 case A, B -> { is a switch rule", gst("BMulti", "switch (m) { case 1, 2 -> { return null; } default -> { } }"), []string{wantFail(3)}},
	{"ST14 default -> { is a switch rule", gst("BDef", "switch (m) { case 1 -> { } default -> { return null; } }"), []string{wantFail(3)}},
	{"ST14 a guarded case is a switch rule", gst("BGuard", "switch (o) { case Integer i when (i > 0) -> { return null; } default -> { } }"), []string{wantFail(3)}},
	{"ST14 a lambda on a case line is nested", gst("BCaseLam", "switch (m) { case 1 -> run(() -> { return null; }); default -> { } }"), nil},
	{"ST14 a lambda assigned after case X: is nested", gst("BColonLam", "switch (m) { case 1: s = x -> { return null; }; break; }"), nil},
	{"ST14 a lambda argument after case X: is nested", gst("BColonRun", "switch (m) { case 1: run(() -> { return null; }); break; }"), nil},
	{"ST14 yield is not a return", gst("BYield", "Type[] t = switch (m) { case 1 -> { yield new Type[0]; } default -> { yield null; } };"), nil},
	{"ST15 \"class A\" in a literal opens no scope", gst("BLitCls", `if (n.equals("class A")) { return null; }`), []string{wantFail(3)}},
	{"ST15 braces in literals do not count", gst("BLitBrace", `String s = "}"; char c = '{'; if (n == null) return null;`), []string{wantFail(3)}},
	{"ST15 an arrow in a literal is not a lambda", gst("BLitArrow", `if (n.equals("a ->")) { return null; }`), []string{wantFail(3)}},
	{"ST15 a return in a literal is not a return", gst("BLitRet", `String r = "return null;";`), nil},

	// Issue #286: the literal blanker's text-block and escape handling. Each case flips on a port that drops
	// the matching rule (see the comment on each).
	{"TB1 an escaped \\\"\"\" does not close a text block", method(`    String s = """`, `      a \""" { return null; }`, `      """;`,
		"    return new Type[] { TYPE };"), nil}, // no escape handling: the block closes early, return null counts
	{"TB2 an escaped backslash then \"\"\" closes the text block", method(`    String s = """`, `      a \\"""; if (x) { return null; }`,
		"    return new Type[] { TYPE };"), []string{wantFail(3)}}, // \\ read as escaping the quote: return null hidden
	{"TB3 a line-continuation backslash escapes nothing on the next line", method(`    String s = """`, `      a \`, `""";`,
		"    if (x) { return null; }", "    return new Type[] { TYPE };"), []string{wantFail(3)}}, // escape carried across lines
	{"TB4 quotes and \"\" inside a text block do not close it", method(`    String s = """`, `      say "hi" and "" { return null; }`,
		`      """;`, "    return new Type[] { TYPE };"), nil}, // a single " closing the block: return null counts
	{"TB5 \\\" in a string and \\' in a char do not close them", method(`    String s = "a \" { return null; }"; char c = '\''; char d = '{';`,
		"    return new Type[] { TYPE };"), nil}, // no escape in "...": the string closes at \", return null counts
	{"TB6 code after a text block closing on its content line counts", method(`    String s = """`, `      x""" + ""; if (y) return null;`,
		"    return new Type[] { TYPE };"), []string{wantFail(3)}}, // closing only on a delimiter-only line hides it
	{"TB7 a text block on the declaration line is blanked", []string{"class B {", `  public Type[] getServiceTypes() { String s = """`,
		`    } return null; {`, `    """; return new Type[] { TYPE }; }`, "}"}, nil},

	// Awk behaviours kept byte-identical (see the retro).
	{"Q1 a one-line class: the body starts at the first brace on the declaration line", []string{
		"class B { public Type[] getServiceTypes() { return new Type[] { TYPE }; } Type[] x() { return null; } }"}, []string{wantFail(1)}},
	{"Q2 no class keyword: the own type is empty", []string{"enum E {", "  A;", "  public Type[] getServiceTypes() { return new Type[] { BI.TYPE }; }", "}"},
		[]string{wantOther(3, "")}},
	{"Q3 a ';' before the declaration on its line skips it", []string{"class B {", "  int x; public Type[] getServiceTypes() { return null; }", "}"}, nil},
	{"Q4 CRLF lines", []string{"class B {\r", "  public Type[] getServiceTypes() {\r", "    return new Type[0];\r", "  }\r", "}\r"}, []string{wantFail(2)}},
	{"Q5 a body with no return is unresolved", method("    throw new IllegalStateException();"), []string{wantUnres(3)}},
	{"Q6 two declarations report in line order", []string{"class B {", "  public Type[] getServiceTypes() { return null; }",
		"  Type[] getServiceTypes(int k) { return new Type[] { X.TYPE }; }", "}"}, []string{wantFail(2), wantOther(3, "B")}},
	{"Q7 a field assigned other and own resolves to other", svc("B", "return t;", "Type[] t = { TYPE };", "static { t = new Type[] { X.TYPE }; }"),
		[]string{wantOther(6, "B")}},
	{"Q8 a field assigned unresolved and own resolves to unresolved", svc("B", "return t;", "Type[] t = { TYPE };", "static { t = make(); }"),
		[]string{wantUnres(6)}},
	{"Q12 only the first assignment of a line is read", svc("B", "return t;", "Type[] t = { TYPE }; static { t = null; }"), nil},
	{"Q9 a multi-line field initializer is read up to its ';'", []string{"class B {", "  static final Type[] t =", "      new Type[]", "      { };",
		"  public Type[] getServiceTypes() { return t; }", "}"}, []string{wantFail(5)}},
	{"Q10 an unbalanced outer paren pair is stripped like awk", svc("B", "return (a)+(b);"), []string{wantUnres(4)}},
	{"Q11 a method that never closes reads to the end of the file", []string{"class B {", "  public Type[] getServiceTypes() {", "    return null;"},
		[]string{wantFail(2)}},
}

func TestCheck(t *testing.T) {
	r := Spec().Rule
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fds, err := r.Check(javasrc.NewFile("M/src/B.java", tc.src))
			if err != nil {
				t.Fatalf("Check error: %v", err)
			}
			var got []string
			for _, f := range fds {
				sev := f.Severity
				if sev == lint.Unset {
					sev = r.Severity()
				}
				got = append(got, fmt.Sprintf("%s %d: %s", sev, f.Line, f.Msg))
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("Check\n got %q\nwant %q\nsrc:\n%s", got, tc.want, strings.Join(tc.src, "\n"))
			}
		})
	}
}

func TestSpec(t *testing.T) {
	s := Spec()
	if s.Rule.ID() != "lint-service-types" || s.Usage != "usage: lint-service-types.sh [--strict] <src-root>" ||
		!reflect.DeepEqual(s.Select.Names, []string{"*.java"}) || len(s.Select.Prune) != 0 ||
		s.Mode != lint.One || s.Rule.Severity() != lint.Fail || s.UsageExit != 0 {
		t.Errorf("Spec() = %+v", s)
	}
}

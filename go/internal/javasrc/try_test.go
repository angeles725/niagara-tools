package javasrc

import (
	"reflect"
	"strings"
	"testing"
)

// tryView is a Try as test-readable text: its line, the lines of its body braces and, per catch, its line,
// types and body lines.
type tryView struct {
	Line, Open, Close int
	Catches           []catchView
}

type catchView struct {
	Line        int
	Types       []string
	Open, Close int
}

func viewTries(src string) []tryView {
	stripped, _ := Strip(strings.Split(src, "\n"))
	t := TryBlocks(stripped)
	var out []tryView
	for _, tr := range t.List {
		v := tryView{t.Line(tr.At), t.Line(tr.Body.Open), t.Line(tr.Body.Close), nil}
		for _, c := range tr.Catches {
			v.Catches = append(v.Catches, catchView{t.Line(c.At), c.Types, t.Line(c.Body.Open), t.Line(c.Body.Close)})
		}
		out = append(out, v)
	}
	return out
}

func TestTryBlocks(t *testing.T) {
	for _, tc := range []struct {
		name, src string
		want      []tryView
	}{
		{"one line", `try { a(); } catch (Exception e) { b(); }`,
			[]tryView{{1, 1, 1, []catchView{{1, []string{"Exception"}, 1, 1}}}}},
		{"catch on the closing line, a second catch, finally", "try {\n  a();\n} catch (IOException e) {\n  b();\n}\ncatch (final RuntimeException r) { c(); }\nfinally { d(); }",
			[]tryView{{1, 1, 3, []catchView{{3, []string{"IOException"}, 3, 5}, {6, []string{"RuntimeException"}, 6, 6}}}}},
		{"multi-catch", `try { a(); } catch (IOException | java.lang.NumberFormatException e) { b(); }`,
			[]tryView{{1, 1, 1, []catchView{{1, []string{"IOException", "java.lang.NumberFormatException"}, 1, 1}}}}},
		{"try-with-resources", "try (Reader r = open(\"{\");\n     Writer w = out()) {\n  a();\n} catch (Exception e) {}",
			[]tryView{{1, 2, 4, []catchView{{4, []string{"Exception"}, 4, 4}}}}},
		{"nested: each catch to its own try", "try {\n  try { a(); }\n  catch (NumberFormatException e) { b(); }\n} catch (Exception e) { c(); }",
			[]tryView{{1, 1, 4, []catchView{{4, []string{"Exception"}, 4, 4}}}, {2, 2, 2, []catchView{{3, []string{"NumberFormatException"}, 3, 3}}}}},
		{"a catch after an unrelated block is not attached", "try { a(); } finally { }\nif (x) { y(); }\ncatch (Exception e) { }",
			[]tryView{{1, 1, 1, nil}}},
		{"comments, literals and lookalike words", "// try { x(); } catch (E e) {}\nString s = \"try {\"; retry { r(); } trying();\ntry /* c */ { a(); } // catch (E e) {}\n catch (E e) { }",
			[]tryView{{3, 3, 3, []catchView{{4, []string{"E"}, 4, 4}}}}},
		{"no brace is no try, an unclosed one runs to the end", "try\nx();\ntry {\n  a();",
			[]tryView{{3, 3, 4, nil}}},
	} {
		if got := viewTries(tc.src); !reflect.DeepEqual(got, tc.want) {
			t.Errorf("%s:\n got %+v\nwant %+v", tc.name, got, tc.want)
		}
	}
}

func TestTriesOffsets(t *testing.T) {
	tr := TryBlocks([]string{"a", "try { b(); }", "c"})
	if got := tr.Offset(2); got != 2 || tr.Line(got) != 2 || tr.Line(len(tr.Text)) != 3 {
		t.Errorf("Offset(2) = %d, Line = %d, last line %d", got, tr.Line(got), tr.Line(len(tr.Text)))
	}
	if b := tr.List[0].Body; !b.Contains(b.Open+1) || b.Contains(b.Open) || b.Contains(b.Close) {
		t.Errorf("Contains on %+v", b)
	}
}

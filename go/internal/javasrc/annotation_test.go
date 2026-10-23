package javasrc

import (
	"reflect"
	"testing"
)

// step is one AnnotationSkip.Step result, with Open() after it.
type step struct {
	n           int
	ended, open bool
}

func TestAnnotationSkip(t *testing.T) {
	type at struct {
		line string
		k    int
	}
	for _, tc := range []struct {
		name  string
		steps []at
		want  []step
	}{
		{"marker annotation", []at{{`@Override public void f()`, 0}},
			[]step{{len(`@Override `), true, false}}},
		{"annotation at a line end", []at{{`@Deprecated`, 0}},
			[]step{{len(`@Deprecated`), true, false}}},
		{"qualified name, blanks before its (", []at{{`@a.b.C  (x)`, 0}, {`@a.b.C  (x)`, 9}, {`@a.b.C  (x)`, 10}},
			[]step{{len(`@a.b.C  (`), false, true}, {1, false, true}, {1, true, false}}},
		{"nested parentheses and annotations stay inside", []at{{`@A(f(@B(1)))`, 0}, {`@A(f(@B(1)))`, 3}, {`@A(f(@B(1)))`, 4},
			{`@A(f(@B(1)))`, 5}, {`@A(f(@B(1)))`, 7}, {`@A(f(@B(1)))`, 9}, {`@A(f(@B(1)))`, 10}, {`@A(f(@B(1)))`, 11}},
			[]step{{3, false, true}, {1, false, true}, {1, false, true}, {1, false, true}, {1, false, true}, {1, false, true}, {1, false, true}, {1, true, false}}},
		{"argument list across lines", []at{{`@Ann(`, 0}, {`  v = "")`, 0}, {`  v = "")`, 8}},
			[]step{{5, false, true}, {1, false, true}, {1, true, false}}},
		{"an @ inside code starts one too", []at{{`x@y z`, 1}}, []step{{len(`@y `), true, false}}},
		{"not an annotation", []at{{`@ x`, 0}, {`a(`, 1}}, []step{{0, false, false}, {0, false, false}}},
	} {
		var a AnnotationSkip
		var got []step
		for _, s := range tc.steps {
			n, ended := a.Step(s.line, s.k)
			got = append(got, step{n, ended, a.Open()})
		}
		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("%s: got %v, want %v", tc.name, got, tc.want)
		}
	}
}

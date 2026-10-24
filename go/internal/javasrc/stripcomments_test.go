package javasrc

import (
	"reflect"
	"strings"
	"testing"
)

// TestStripComments: the comment text of each line beside Strip's code; a comment marker inside a literal is code.
func TestStripComments(t *testing.T) {
	src := "a(); // why: broad on purpose\n/* justif\n   more */ b();\nString s = \"// not a comment\"; /* x */ c();\nd();"
	code, comments, state := StripComments(strings.Split(src, "\n"))
	want := []string{"// why: broad on purpose", "/* justif", "   more */", "/* x */", ""}
	if !reflect.DeepEqual(comments, want) || state != Closed {
		t.Errorf("comments %q state %v, want %q", comments, state, want)
	}
	if stripped, _ := Strip(strings.Split(src, "\n")); !reflect.DeepEqual(code, stripped) {
		t.Errorf("code %q differs from Strip %q", code, stripped)
	}
}

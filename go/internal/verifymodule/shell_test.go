package verifymodule

import (
	"reflect"
	"testing"
)

// TestGrepText pins the GNU grep 3.11 output rules under C.UTF-8 that only/count keep: a line with an
// encoding error matches and counts but is never printed (-o prints a valid match from it), and a file
// holding a NUL byte prints nothing at all. hits reads bytes (grep -a) and reports every such line.
func TestGrepText(t *testing.T) {
	re := rx(`station:[a-z]`)
	latin := newGrepText("a \"station:x\"\nb \xe9 \"station:y\"\nc \"station:z\"\n")
	if got := latin.hits(re); !reflect.DeepEqual(got, []string{`1:a "station:x"`, "2:b \uFFFD \"station:y\"", `3:c "station:z"`}) {
		t.Errorf("hits (bytes, encoding error shown as U+FFFD) %q", got)
	}
	if got := latin.only(re); !reflect.DeepEqual(got, []string{"station:x", "station:y", "station:z"}) {
		t.Errorf("only %q", got)
	}
	nul := newGrepText("a station:x\n\x00b\nc station:z")
	if nul.count(re) != 2 || len(nul.hits(re)) != 2 || nul.only(re) != nil {
		t.Errorf("binary file: count %d hits %q only %q", nul.count(re), nul.hits(re), nul.only(re))
	}
	if newGrepText("").lines != nil || len(newGrepText("\n").lines) != 1 {
		t.Error("an empty input has no line, a lone newline one")
	}
}

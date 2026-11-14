package prepush

import (
	"os"
	"regexp"
	"testing"
)

// The live doctrine set is defined for shell callers in scripts/lib/live-doctrine-set.sh; the Go
// patterns must be the same text, so the two definitions cannot drift.
func TestLiveDoctrinePatternsEqualTheShellDefinition(t *testing.T) {
	lib, err := os.ReadFile("../../../scripts/lib/live-doctrine-set.sh")
	if err != nil {
		t.Fatal(err)
	}
	for name, re := range map[string]*regexp.Regexp{"LIVE_DOCTRINE_RE": liveDoctrine, "LIVE_CITE_TOOL_RE": liveCiteTool} {
		m := regexp.MustCompile(`(?m)^` + name + `='([^']*)'$`).FindSubmatch(lib)
		if m == nil {
			t.Fatalf("%s is not a single-quoted assignment in the shell library", name)
		}
		if string(m[1]) != re.String() {
			t.Errorf("%s differs\nshell: %s\ngo:    %s", name, m[1], re)
		}
	}
}

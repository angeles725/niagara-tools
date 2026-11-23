package ngdeploy

import (
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/exitdoc"
)

// TestEveryExitCodeIsInTheUsageText (retro build-kit-bench-session Δ20): each Exit* constant of exit.go has a line
// in the "Exit codes:" list of usage.txt, alone or in a list of codes ("130, 143"), so a code a test pins cannot
// be missing from what the operator reads.
func TestEveryExitCodeIsInTheUsageText(t *testing.T) {
	codes, err := exitdoc.Codes("exit.go")
	if err != nil || len(codes) == 0 {
		t.Fatalf("exit.go: %v, %d codes", err, len(codes))
	}
	_, list, found := strings.Cut(usage, "\nExit codes:\n")
	if !found {
		t.Fatal(`usage.txt has no "Exit codes:" list`)
	}
	for _, c := range codes {
		line := regexp.MustCompile(`(?m)^  (?:[0-9]+, )*` + strconv.Itoa(c.Value) + `(?:, [0-9]+)*  `)
		if !line.MatchString(list) {
			t.Errorf("%s = %d has no line in the exit codes of usage.txt", c.Name, c.Value)
		}
	}
}

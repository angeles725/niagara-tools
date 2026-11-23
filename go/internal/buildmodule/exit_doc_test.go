package buildmodule

import (
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/exitdoc"
)

// TestEveryExitCodeIsInBuildVerify (retro build-kit-bench-session Δ20): each Exit* constant of exit.go is named
// as **N** in the build.sh paragraph of build-verify.md, the exit list the kit's documentation keeps.
func TestEveryExitCodeIsInBuildVerify(t *testing.T) {
	codes, err := exitdoc.Codes("exit.go")
	if err != nil || len(codes) == 0 {
		t.Fatalf("exit.go: %v, %d codes", err, len(codes))
	}
	doc, err := os.ReadFile("../../../build-n4-module-kit/build-verify.md")
	if err != nil {
		t.Fatal(err)
	}
	var paragraph string
	for _, line := range strings.Split(string(doc), "\n") {
		if strings.HasPrefix(line, "`build.sh` usage:") {
			paragraph = line
		}
	}
	if paragraph == "" {
		t.Fatal("build-verify.md has no `build.sh` usage paragraph")
	}
	for _, c := range codes {
		if !strings.Contains(paragraph, "**"+strconv.Itoa(c.Value)+"**") {
			t.Errorf("%s = %d is not named as **%d** in the build.sh paragraph of build-verify.md", c.Name, c.Value, c.Value)
		}
	}
}

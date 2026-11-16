package structure

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// TestStructureDocListsEveryRule: the rule table of types/structure.md has one row per rule the lint can report
// (every "L<n>: " row text in this package's sources), and no other row but those it marks "not run by this lint".
func TestStructureDocListsEveryRule(t *testing.T) {
	doc, err := os.ReadFile("../../../../../build-n4-module-kit/types/structure.md")
	if err != nil {
		t.Fatal(err)
	}
	var inDoc, inCode []string
	for _, m := range regexp.MustCompile(`(?m)^\| (L[0-9]+b?) \|[^|]*\| ([^|]*)\|`).FindAllStringSubmatch(string(doc), -1) {
		if !strings.HasPrefix(m[2], "not run by this lint") {
			inDoc = append(inDoc, m[1])
		}
	}
	seen := map[string]bool{}
	sources, _ := filepath.Glob("*.go")
	for _, f := range sources {
		src, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		for _, m := range regexp.MustCompile("[\"`](L[0-9]+b?): ").FindAllStringSubmatch(string(src), -1) {
			if !seen[m[1]] {
				seen[m[1]] = true
				inCode = append(inCode, m[1])
			}
		}
	}
	sort.Strings(inDoc)
	sort.Strings(inCode)
	if got, want := strings.Join(inDoc, " "), strings.Join(inCode, " "); got != want || len(inCode) == 0 {
		t.Errorf("types/structure.md rule rows:\n %s\nrules the lint reports:\n %s", got, want)
	}
}

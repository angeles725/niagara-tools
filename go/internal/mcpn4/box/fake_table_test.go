package box

import (
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"testing"
)

// TestTheFakeStationSharesTheComponentTypeTable keeps the black-box suite's FakeStation
// (mcp-n4-kit/tests/fake_station.py COMPONENT_TYPES) on the server's table: the fake decides
// which add ops nest a component the way the write tools do.
func TestTheFakeStationSharesTheComponentTypeTable(t *testing.T) {
	src, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "mcp-n4-kit", "tests", "fake_station.py"))
	if err != nil {
		t.Fatal(err)
	}
	block := regexp.MustCompile(`(?s)\nCOMPONENT_TYPES = \{\n(.*?)\n\}\n`).FindSubmatch(src)
	if block == nil {
		t.Fatal("fake_station.py has no COMPONENT_TYPES table")
	}
	fake := map[string]bool{}
	for _, m := range regexp.MustCompile(`"([^"]+)": (True|False),`).FindAllSubmatch(block[1], -1) {
		fake[string(m[1])] = string(m[2]) == "True"
	}
	if !reflect.DeepEqual(fake, componentTypes) {
		t.Errorf("fake_station.py COMPONENT_TYPES = %v, the server's = %v", fake, componentTypes)
	}
}

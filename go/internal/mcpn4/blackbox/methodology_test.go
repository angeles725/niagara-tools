package blackbox

// Every test METHODOLOGY.md cites exists and is a Go test (port of the kit's former Python
// methodology-names test).

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"testing"
)

var (
	pyCite = regexp.MustCompile(`\btest_[a-z0-9_]+(\.py)?\b`)
	goCite = regexp.MustCompile("`([a-z]+)\\.(Test[A-Za-z0-9]+)(?:/([a-z0-9_]+))?`")
)

// TestMethodologyCitesNoPythonTest: the kit's tests are Go since T9 retired the Python suite, so a `test_name` or
// `test_file.py` in METHODOLOGY.md can only be a stale citation; cite the Go twin
// (`pkg.TestName`, checked by TestEveryGoTestCitedInMethodologyExists) instead.
func TestMethodologyCitesNoPythonTest(t *testing.T) {
	seen := map[string]bool{}
	var cited []string
	for _, name := range pyCite.FindAllString(kitDoc(t, "METHODOLOGY.md"), -1) {
		if !seen[name] {
			seen[name] = true
			cited = append(cited, name)
		}
	}
	sort.Strings(cited)
	if len(cited) != 0 {
		t.Errorf("METHODOLOGY.md cites Python tests (the suite is Go; cite the Go twin): %v", cited)
	}
}

// TestEveryGoTestCitedInMethodologyExists: a cited `pkg.TestName` is a func TestName(t
// *testing.T) of go/internal/mcpn4/pkg; a cited `box.TestReplayRecordedFrames/<scenario>` is a
// scenario of box/testdata/replay.jsonl.
func TestEveryGoTestCitedInMethodologyExists(t *testing.T) {
	cites := goCite.FindAllStringSubmatch(kitDoc(t, "METHODOLOGY.md"), -1)
	scenarios := replayScenarios(t)
	seen, missing := map[string]bool{}, []string{}
	for _, c := range cites {
		pkg, name, sub := c[1], c[2], c[3]
		if seen[c[0]] {
			continue
		}
		seen[c[0]] = true
		replay := pkg == "box" && name == "TestReplayRecordedFrames"
		if sub != "" && (!replay || !scenarios[sub]) {
			missing = append(missing, pkg+"."+name+"/"+sub)
		}
		if !definesTest(t, pkg, name) {
			missing = append(missing, pkg+"."+name)
		}
	}
	if len(seen) <= 30 {
		t.Errorf("METHODOLOGY.md cites %d Go tests: a vacuous gate", len(seen))
	}
	sort.Strings(missing)
	if len(missing) != 0 {
		t.Errorf("METHODOLOGY.md cites Go tests that do not exist: %v", missing)
	}
}

func replayScenarios(t *testing.T) map[string]bool {
	f, err := os.Open(filepath.Join(mcpn4Dir, "box", "testdata", "replay.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	out := map[string]bool{}
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 64<<20)
	for sc.Scan() {
		var line struct{ Name string }
		if err := json.Unmarshal(sc.Bytes(), &line); err != nil {
			t.Fatal(err)
		}
		out[line.Name] = true
	}
	return out
}

func definesTest(t *testing.T, pkg, name string) bool {
	files, _ := filepath.Glob(filepath.Join(mcpn4Dir, pkg, "*_test.go"))
	def := regexp.MustCompile(`(?m)^func ` + name + `\(t \*testing\.T\)`)
	for _, path := range files {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if def.Match(data) {
			return true
		}
	}
	return false
}

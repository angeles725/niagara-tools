package blackbox

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest/boxstation"
)

// shxSpyFixture is a recorded spy page of the spy package's testdata (a laboratory station's
// answer with its station details replaced by neutral values), read in place.
func shxSpyFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "spy", "testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return data
}

// shxSpyRig is a connected servlet rig whose station serves the recorded index, metrics and
// util spy pages.
func shxSpyRig(t *testing.T) (*Rig, *boxstation.Servlets) {
	t.Helper()
	r, sv := servletRig(t, nil)
	shxServeSpy(sv, "", boxstation.SpyPage{Body: shxSpyFixture(t, "index.html")})
	shxServeSpy(sv, "metrics", boxstation.SpyPage{Body: shxSpyFixture(t, "metrics.html")})
	shxServeSpy(sv, "util", boxstation.SpyPage{Body: shxSpyFixture(t, "util.html")})
	return r, sv
}

func shxServeSpy(sv *boxstation.Servlets, path string, page boxstation.SpyPage) {
	sv.Do(func(sv *boxstation.Servlets) { sv.SpyPages[path] = page })
}

func shxSpyRequests(sv *boxstation.Servlets) (out []string) {
	sv.Do(func(sv *boxstation.Servlets) { out = append([]string{}, sv.SpyRequests...) })
	return out
}

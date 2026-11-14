package blackbox

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest/boxstation"
)

// servletRig is a connected rig whose station also serves GET /file/ and /ord/ (the kit's
// FakeStation with files and ord_routes); configure, when not nil, edits the options first.
func servletRig(t *testing.T, configure func(o *Options)) (*Rig, *boxstation.Servlets) {
	t.Helper()
	var sv *boxstation.Servlets
	r := NewRig(t, func(r *Rig, o *Options) {
		sv = boxstation.ServeServlets(t, r.Station)
		o.Stations = map[string]string{StationName: sv.URL}
		if configure != nil {
			configure(o)
		}
	})
	r.Connect(nil)
	return r, sv
}

// fileRig is FileToolCase: a servlet rig serving the synthetic Px view as px/General.px.
func fileRig(t *testing.T, configure func(o *Options)) (*Rig, *boxstation.Servlets) {
	t.Helper()
	r, sv := servletRig(t, configure)
	serveFile(sv, "px/General.px", syntheticPx(t), "text/xml")
	return r, sv
}

// kitFixture is the bytes of mcp-n4-kit/tests/fixtures/<name>, read in place.
func kitFixture(t *testing.T, name ...string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(append([]string{kitDir, "tests", "fixtures"}, name...)...))
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func syntheticPx(t *testing.T) []byte { return kitFixture(t, "synthetic_view.px") }

func serveFile(sv *boxstation.Servlets, path string, body []byte, ctype string) {
	sv.Do(func(sv *boxstation.Servlets) { sv.Files[path] = boxstation.File{Body: body, ContentType: ctype} })
}

func routeOrds(sv *boxstation.Servlets, routes ...boxstation.OrdRoute) {
	sv.Do(func(sv *boxstation.Servlets) { sv.OrdRoutes = routes })
}

func fileRequests(sv *boxstation.Servlets) (out []string) {
	sv.Do(func(sv *boxstation.Servlets) { out = append([]string{}, sv.FileRequests...) })
	return out
}

func ordRequests(sv *boxstation.Servlets) (out []string) {
	sv.Do(func(sv *boxstation.Servlets) { out = append([]string{}, sv.OrdRequests...) })
	return out
}

func getCount(sv *boxstation.Servlets) (n int) {
	sv.Do(func(sv *boxstation.Servlets) { n = sv.GETs })
	return n
}

func wantStrings(t *testing.T, what string, got, want []string) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("%s = %q, want %q", what, got, want)
	}
}

// toolsByName is the server's tools/list, by tool name.
func toolsByName(t *testing.T, s *Server) map[string]map[string]any {
	t.Helper()
	out := map[string]map[string]any{}
	for _, item := range objects(Dig(s.Request("tools/list", nil), "result", "tools")) {
		name, _ := item["name"].(string)
		out[name] = item
	}
	if len(out) == 0 {
		t.Fatal("tools/list listed no tools")
	}
	return out
}

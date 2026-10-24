package files_test

// Go twins of the test_files tool tests that read srv.ctx (go_server.go_twin("files",
// "internals")), plus the output_file rules, on an in-memory station that serves GET /file/.

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/session"
)

func fileServer(t *testing.T, stateDir string) *readtoolstest.Server {
	t.Helper()
	s := readtoolstest.NewServer(readtoolstest.NewStation(), session.Config{StateDir: stateDir})
	s.Connect(t)
	s.Station.Files = map[string]readtoolstest.File{"px/General.px": {Body: "<px media='m'/>", ContentType: "text/xml"}}
	return s
}

func call(t *testing.T, s *readtoolstest.Server, tool string, wantErr bool, kv ...any) (string, *pyjson.Object) {
	t.Helper()
	text, isErr, out := s.Call(t, tool, kv...)
	if isErr != wantErr {
		t.Fatalf("%s: isError %v: %s", tool, isErr, text)
	}
	return text, out
}

func get(o *pyjson.Object, k string) any { v, _ := o.Get(k); return v }

func TestMissingAndForbiddenFilesAreTypedResultsWithoutALatch(t *testing.T) {
	s := fileServer(t, "")
	_, out := call(t, s, "n4_read_file", false, "file_ord", "file:^nav/NavFile.nav")
	if get(out, "status") != "not_found" || get(out, "http_status") != 404 {
		t.Errorf("missing: %s", pyjson.Repr(out))
	}
	s.Station.Files["px/Secret.px"] = readtoolstest.File{Code: 403}
	_, out = call(t, s, "n4_read_file", false, "file_ord", "px/Secret.px")
	if get(out, "status") != "forbidden" || get(out, "http_status") != 403 || get(out, "hint") == nil {
		t.Errorf("forbidden: %s", pyjson.Repr(out))
	}
	if block := s.Ctx.AuthBlock(); block != "" {
		t.Errorf("a 403/404 latched: %s", block)
	}
	if _, out = call(t, s, "n4_read_file", false, "file_ord", "px/General.px"); get(out, "status") != "ok" {
		t.Errorf("after: %s", pyjson.Repr(out))
	}
}

func TestTheReadIsRecordedAsASessionObservation(t *testing.T) {
	s := fileServer(t, "")
	call(t, s, "n4_read_file", false, "file_ord", "file:^px/General.px")
	obs := s.Ctx.Observations[len(s.Ctx.Observations)-1]
	if get(obs, "tool") != "n4_read_file" || get(obs, "ord") != "file:^px/General.px" || get(obs, "rows") != 1 {
		t.Errorf("observation %s", pyjson.Repr(obs))
	}
	call(t, s, "n4_read_file", false, "file_ord", "px/None.px")
	if obs = s.Ctx.Observations[len(s.Ctx.Observations)-1]; get(obs, "rows") != 0 {
		t.Errorf("a 404 observation %s", pyjson.Repr(obs))
	}
	call(t, s, "n4_list_px_views", true) // no BQL route: the query fails after no observation
	s.Station.OrdRoutes = []readtoolstest.OrdRoute{{Body: "Slot Path,Display Name,Px File,Media\nNULL,G,file:^px/G.px,m\n"}}
	call(t, s, "n4_list_px_views", false)
	if obs = s.Ctx.Observations[len(s.Ctx.Observations)-1]; get(obs, "tool") != "n4_list_px_views" || get(obs, "ord") != "" || get(obs, "rows") != 1 {
		t.Errorf("px views observation %s", pyjson.Repr(obs))
	}
}

func TestReadFileOutputIsPrivateAndNeverOverwritten(t *testing.T) {
	state := filepath.Join(t.TempDir(), "state")
	if err := os.Mkdir(state, 0o700); err != nil {
		t.Fatal(err)
	}
	s := fileServer(t, state)
	s.Station.Files["images/logo.png"] = readtoolstest.File{Body: "\x89PNG\r\n\x00", ContentType: "image/png"}
	_, out := call(t, s, "n4_read_file", false, "file_ord", "images/logo.png", "output_file", "logo.png")
	path := filepath.Join(state, "files", "logo.png")
	data, err := os.ReadFile(path)
	if get(out, "output_file") != path || err != nil || string(data) != "\x89PNG\r\n\x00" || get(out, "binary") != true {
		t.Fatalf("saved %s: %q %v", pyjson.Repr(out), data, err)
	}
	for p, want := range map[string]os.FileMode{path: 0o600, filepath.Dir(path): 0o700} {
		if info, _ := os.Stat(p); info.Mode().Perm() != want {
			t.Errorf("%s mode %o, want %o", p, info.Mode().Perm(), want)
		}
	}
	text, _ := call(t, s, "n4_read_file", true, "file_ord", "images/logo.png", "output_file", "logo.png")
	if want := "output_file " + path + " already exists: choose another name (the server never overwrites a result file)"; text != want {
		t.Errorf("overwrite: %s", text)
	}
	os.Remove(path)
	os.Symlink(filepath.Join(state, "elsewhere"), path)
	call(t, s, "n4_read_file", true, "file_ord", "images/logo.png", "output_file", "logo.png")
	if _, err := os.Lstat(filepath.Join(state, "elsewhere")); err == nil {
		t.Error("written through a dangling link")
	}
	before := len(s.Station.FileRequests)
	call(t, s, "n4_read_file", true, "file_ord", "px/General.px", "output_file", "a.b/c")
	call(t, s, "n4_read_file", true, "file_ord", "px/General.px", "output_file", "noext")
	if len(s.Station.FileRequests) != before {
		t.Error("a refused output_file sent a request")
	}
	s.Station.Files["px/Big.px"] = readtoolstest.File{Body: string(make([]byte, 2000)), ContentType: "text/xml"}
	_, out = call(t, s, "n4_read_file", false, "file_ord", "px/Big.px", "max_bytes", pyjson.IntLiteral("1024"), "output_file", "big.px")
	if get(out, "output_skipped") == nil || get(out, "output_file") != nil || get(out, "sha256") != nil {
		t.Errorf("truncated: %s", pyjson.Repr(out))
	}
	if _, err := os.Lstat(filepath.Join(state, "files", "big.px")); err == nil {
		t.Error("a truncated read was saved")
	}
}

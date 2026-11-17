package writeops_test

import (
	"archive/zip"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

const lab = "station:|slot:/Lab"

// tplSource is testdata/template/FanCoil/<entry>, a component template's source.
func tplSource(t *testing.T, entry string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "template", "FanCoil", entry))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// tplFile zips the FanCoil sources into a .ntpl under t.TempDir(), each edit (old, new pairs on
// the manifest, then the bog) applied first.
func tplFile(t *testing.T, manifestEdits, bogEdits []string) string {
	t.Helper()
	manifest, bog := tplSource(t, "template-manifest.xml"), tplSource(t, "template.bog")
	for i := 0; i+1 < len(manifestEdits); i += 2 {
		manifest = strings.Replace(manifest, manifestEdits[i], manifestEdits[i+1], 1)
	}
	for i := 0; i+1 < len(bogEdits); i += 2 {
		bog = strings.Replace(bog, bogEdits[i], bogEdits[i+1], 1)
	}
	return writeZip(t, "FanCoil.ntpl", map[string]string{"template-manifest.xml": manifest, "template.bog": bog})
}

func writeZip(t *testing.T, name string, entries map[string]string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	w := zip.NewWriter(f)
	for _, n := range []string{"template-manifest.xml", "template.bog"} {
		if body, ok := entries[n]; ok {
			e, err := w.Create(n)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := e.Write([]byte(body)); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	return path
}

// tplTypes is the registry reply of a station that has baja and control: every type the
// template adds is described, and BooleanWritable declares its fallback (a frozen
// StatusBoolean, null as the 4.15 lab answers it) so a setting can write it although the bog
// leaves it at its default.
func tplTypes() *pyjson.Object {
	return obj("baja:Folder", obj(), "baja:String", obj(), "baja:Double", obj(), "baja:Marker", obj(),
		"baja:StatusNumeric", obj("c", []any{obj("st", "p", "n", "value", "v", obj("nm", "p", "t", "baja:Double", "v", "0.0"))}),
		"baja:StatusBoolean", obj("c", []any{obj("st", "p", "n", "value", "v", obj("nm", "p", "t", "baja:Boolean", "v", "false"))}),
		"control:NumericWritable", obj("c", []any{}, "p", "baja:Component"),
		"control:BooleanWritable", obj("c", []any{obj("st", "p", "n", "fallback", "v", obj("nm", "p", "t", "baja:StatusBoolean",
			"s", []any{obj("nm", "p", "n", "value", "v", "false"), obj("nm", "p", "n", "status", "v", "40")}))},
			"p", "baja:Component"),
		"baja:Component", obj())
}

// tplClient is the lab folder (handle 10, one child Existing) on a station answering types.
func tplClient(types *pyjson.Object, extra ...any) *typed {
	parent := obj("", obj("h", "10", "t", "baja:Folder"), "Existing", obj("h", "11", "t", "baja:Folder"))
	nodes := obj(lab, parent)
	for i := 0; i+1 < len(extra); i += 2 {
		nodes.Set(extra[i].(string), extra[i+1])
	}
	return &typed{loads: loads{nodes: nodes}, types: types}
}

func deployArgs(path string, kv ...any) *pyjson.Object {
	a := obj("template_file", path, "parent_ord", lab, "name", "FC1",
		"settings", obj("Setpoint", pyjson.IntLiteral("68"), "Location", "Room 101"))
	for i := 0; i+1 < len(kv); i += 2 {
		a.Set(kv[i].(string), kv[i+1])
	}
	return a
}

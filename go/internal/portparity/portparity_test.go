package portparity

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func write(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for rel, text := range files {
		p := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestExtract(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		text       string
		ids, funcs []string
	}{
		{"html id attribute", `<div id="equipmentView"></div><p id='noModel'>`, []string{"equipmentView", "noModel"}, nil},
		{"id inside a js string, escaped quotes", `el.innerHTML = "<div id=\"eqPanel\">";`, []string{"eqPanel"}, nil},
		{"id property assignment", `node.id = 'sp-body';`, []string{"sp-body"}, nil},
		{"function declaration and generator", "function renderEquipment(a) {}\nasync function* gen() {}", nil, []string{"renderEquipment", "gen"}},
		{"const function expression and arrows", "const eqData = function () {};\nlet eqHist = (a, b) => a;\nvar clamp = v => v;\nconst load = async () => 1;",
			nil, []string{"eqData", "eqHist", "clamp", "load"}},
		{"a plain const value is no function", "const limit = 4;\nconst name = other;", nil, nil},
		{"data-id and hidden ids are not ids", `<div data-id="x" hidden-id="y">`, nil, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ids, funcs := extract(tt.text)
			if got := names(ids); !reflect.DeepEqual(got, tt.ids) {
				t.Errorf("ids = %v, want %v", got, tt.ids)
			}
			if got := names(funcs); !reflect.DeepEqual(got, tt.funcs) {
				t.Errorf("funcs = %v, want %v", got, tt.funcs)
			}
		})
	}
}

func names(ms []match) []string {
	var out []string
	for _, m := range ms {
		out = append(out, m.name)
	}
	return out
}

func TestScanSkipsVendorDataAndForeignFiles(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	write(t, root, map[string]string{
		"js/app.js":               "function boot() {}\n\nfunction show() { return '<b id=\"main\">'; }\n",
		"index.html":              `<div id="main"></div>`,
		"css/site.css":            "body {}\n",
		"vendor/lib.js":           "function vendored() {}",
		"node_modules/x/index.js": "function dep() {}",
		"data/sample.js":          "function sample() {}",
		"js/chart.min.js":         "function minified() {}",
		".deploy-baseline/old.js": "function old() {}",
		"README.md":               "function notCode() {}",
	})
	tr, err := Scan(root)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := sortedKeys(tr.Files), []string{"app.js", "index.html", "site.css"}; !reflect.DeepEqual(got, want) {
		t.Errorf("files = %v, want %v", got, want)
	}
	if got, want := sortedKeys(tr.Funcs), []string{"boot", "show"}; !reflect.DeepEqual(got, want) {
		t.Errorf("funcs = %v, want %v", got, want)
	}
	// first occurrence in lexical walk order: index.html sorts before js/app.js
	if m := tr.IDs["main"]; m.Path != "index.html" || m.Line != 1 {
		t.Errorf("id main at %s:%d, want index.html:1", m.Path, m.Line)
	}
	if m := tr.Funcs["show"]; m.Path != "js/app.js" || m.Line != 3 {
		t.Errorf("func show at %s:%d, want js/app.js:3", m.Path, m.Line)
	}
}

func TestScanErrors(t *testing.T) {
	t.Parallel()
	if _, err := Scan(filepath.Join(t.TempDir(), "absent")); err == nil {
		t.Error("a missing root must be an error")
	}
	empty := t.TempDir()
	if _, err := Scan(empty); err == nil {
		t.Error("a root with no .js/.mjs/.html/.css file must be an error (never a vacuous pass)")
	}
}

func TestMissingIsSortedByKindThenName(t *testing.T) {
	t.Parallel()
	ref, port := t.TempDir(), t.TempDir()
	write(t, ref, map[string]string{
		"a.js":    "function zeta() {}\nfunction alpha() {}\nfunction kept() {}",
		"b.html":  `<i id="z1"></i><i id="a1"></i><i id="both"></i>`,
		"gone.js": "function kept() {}",
	})
	write(t, port, map[string]string{
		"a.js":   "function kept() {}",
		"b.html": `<i id="both"></i>`,
	})
	r, _ := Scan(ref)
	p, _ := Scan(port)
	var got []string
	for _, m := range Missing(r, p) {
		got = append(got, string(m.Kind)+" "+m.Name)
	}
	want := []string{"file gone.js", "id a1", "id z1", "function alpha", "function zeta"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("missing = %v, want %v", got, want)
	}
}

package ntplmanifest

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/lint"
)

// entry is one member of a template archive built for a test.
type entry struct{ name, body string }

const bog = `<bajaObjectGraph version="4.0"><p m="b=baja" t="b:Folder"/></bajaObjectGraph>` + "\n"

// manifestXML is a template manifest whose root carries rootAttrs and whose <dependencies> name deps.
func manifestXML(rootAttrs string, deps ...string) string {
	s := `<?xml version="1.0" encoding="UTF-8"?>` + "\n<templateManifest id=\"t1\" version=\"1.0\"" + rootAttrs + ">\n  <dependencies>\n"
	for _, d := range deps {
		s += `    <dependency name="` + d + `" vendor="Acme" vendorVersion="1.0" rel="minimum"/>` + "\n"
	}
	return s + "  </dependencies>\n</templateManifest>\n"
}

// ntpl builds a template archive in memory with archive/zip, entries in the order given.
func ntpl(t *testing.T, entries ...entry) string {
	t.Helper()
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	for _, e := range entries {
		f, err := w.Create(e.name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.Write([]byte(e.body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}

// template is a whole archive: template.bog and a manifest with deps.
func template(t *testing.T, rootAttrs string, deps ...string) string {
	return ntpl(t, entry{bogEntry, bog}, entry{manifestEntry, manifestXML(rootAttrs, deps...)})
}

// partBuild is ModA-rt's build file: it declares nre, baja and control-rt.
const partBuild = "dependencies {\n  nre(\":nre\")\n  api(\":baja\")\n  api(\":control-rt\")\n  // api(\":commented-rt\")\n}\n"

// moduleRoot writes files (path -> content) under a fresh module root that holds ModA-rt's build file and a
// settings file whose call declares nothing (it is no build file).
func moduleRoot(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	write(t, root, "settings.gradle.kts", "include(\":ModA-rt\")\ndependencies { api(\":settings-only-rt\") }\n")
	write(t, root, "ModA-rt/ModA-rt.gradle.kts", partBuild)
	for rel, body := range files {
		write(t, root, rel, body)
	}
	return root
}

func write(t *testing.T, root, rel, body string) {
	t.Helper()
	p := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// run is the lint as its toolbelt wrapper runs it: n4lint lint-ntpl-manifest-deps <args>.
func run(args ...string) (code int, stdout, stderr string) {
	var out, errb bytes.Buffer
	code = lint.Command(Spec())(append([]string{ID}, args...), cli.Stdio{Out: &out, Err: &errb})
	return code, out.String(), errb.String()
}

// expect runs the lint on root and compares exit code, stdout (@ROOT@ stands for root) and an empty stderr.
func expect(t *testing.T, name, root string, code int, rows string) {
	t.Helper()
	want := bytes.ReplaceAll([]byte(rows), []byte("@ROOT@"), []byte(root))
	if c, out, errText := run(root); c != code || out != string(want) || errText != "" {
		t.Errorf("%s: exit %d\nstdout %q\nstderr %q\nwant exit %d\nstdout %q", name, c, out, errText, code, want)
	}
}

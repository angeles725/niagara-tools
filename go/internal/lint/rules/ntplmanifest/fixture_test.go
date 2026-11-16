package ntplmanifest

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/lint"
)

// entry is one member of a template archive built for a test.
type entry struct{ name, body string }

const bog = `<bajaObjectGraph version="4.0"><p m="b=baja" t="b:Folder"/></bajaObjectGraph>` + "\n"

// manifestXML is a template manifest whose root carries rootAttrs and whose <dependencies> name deps.
func manifestXML(rootAttrs string, deps ...string) string {
	s := `<?xml version="1.0" encoding="UTF-8"?>` + "\n<template id=\"t1\" version=\"1.0\"" + rootAttrs + ">\n  <dependencies>\n"
	for _, d := range deps {
		s += `    <dependency name="` + d + `" vendor="Acme" vendorVersion="1.0" rel="minimum"/>` + "\n"
	}
	return s + "  </dependencies>\n</template>\n"
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

// partBuild is ModA-rt's build file: it declares nre, baja and local-rt (a module no installation ships).
const partBuild = "dependencies {\n  nre(\":nre\")\n  api(\":baja\")\n  api(\":local-rt\")\n  // api(\":commented-rt\")\n}\n"

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

// run is the lint as its toolbelt wrapper runs it, n4lint lint-ntpl-manifest-deps <args>, with no
// NIAGARA_HOME in the environment whatever the machine running the test has.
func run(args ...string) (code int, stdout, stderr string) {
	return runEnv(nil, args...)
}

// runEnv is run with env as the whole environment the lint reads.
func runEnv(env map[string]string, args ...string) (code int, stdout, stderr string) {
	var out, errb bytes.Buffer
	spec := specWith(func(k string) string { return env[k] })
	code = lint.Command(spec)(append([]string{ID}, args...), cli.Stdio{Out: &out, Err: &errb})
	return code, out.String(), errb.String()
}

// stock are the modules of the fixture installation, named like some an installation ships.
var stock = []string{"baja", "fox-rt", "jetty-rt", "hx-wb", "web-rt", "control-rt", "bajaui-wb"}

// installation is a fixture Niagara home: modules/ holding one empty <name>.jar per name, next to files that
// are no module (a dot file, a text file).
func installation(t *testing.T, names ...string) string {
	t.Helper()
	dir := t.TempDir()
	write(t, dir, "modules/.hidden.jar", "")
	write(t, dir, "modules/readme.txt", "")
	for _, n := range names {
		write(t, dir, "modules/"+n+".jar", "")
	}
	return dir
}

// expect runs the lint on root with the fixture installation of stock modules and compares exit code,
// stdout (@ROOT@ stands for root, @HOME@ for the installation) and an empty stderr.
func expect(t *testing.T, name, root string, code int, rows string) {
	t.Helper()
	nh := installation(t, stock...)
	want := strings.NewReplacer("@ROOT@", root, "@HOME@", nh).Replace(rows)
	if c, out, errText := run(homeOption, nh, root); c != code || out != want || errText != "" {
		t.Errorf("%s: exit %d\nstdout %q\nstderr %q\nwant exit %d\nstdout %q", name, c, out, errText, code, want)
	}
}

// undeclaredRow is the FAIL ntpl-deps row of rel naming names, as expect spells it.
func undeclaredRow(rel, names string) string {
	return "FAIL  ntpl-deps  @ROOT@/" + rel + ": manifest dependencies neither declared (api/nre/implementation) in any *.gradle.kts under @ROOT@ nor shipped by the installation @HOME@: " + names + "\n"
}

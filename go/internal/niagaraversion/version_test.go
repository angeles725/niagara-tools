package niagaraversion

import (
	"archive/zip"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// writeJar writes a jar holding the given entries (name to content) and returns its path.
func writeJar(t *testing.T, path string, entries map[string]string) string {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	for name, content := range entries {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	return path
}

// installation makes <root>/modules/baja.jar the way a Niagara installation ships it and returns root.
func installation(t *testing.T, version string) string {
	t.Helper()
	root := t.TempDir() + "/Niagara-" + version
	writeJar(t, root+"/modules/baja.jar", map[string]string{
		"META-INF/module.xml": `<?xml version="1.0" encoding="UTF-8"?>
<module name="baja" bajaVersion="0" vendor="Tridium" vendorVersion="` + version + `" nre="true">
 <installation><dependencies><nre name="nre-core-*" version="` + version + `"/></dependencies></installation>
</module>`,
	})
	return root
}

// moduleJar makes a built module jar whose module.xml declares the given dependency elements.
func moduleJar(t *testing.T, dependencies string) string {
	t.Helper()
	return writeJar(t, t.TempDir()+"/Demo-rt.jar", map[string]string{
		"META-INF/module.xml": `<module name="Demo-rt" vendor="Acme" vendorVersion="2.4.0">
 <dependencies>` + dependencies + `</dependencies>
</module>`,
	})
}

func TestInstallVersionReadsBajaOfTheRootOrItsModulesDirectory(t *testing.T) {
	t.Parallel()
	root := installation(t, "4.14.0.162")
	for _, dir := range []string{root, root + "/modules", root + "/"} {
		version, got, err := InstallVersion(dir)
		if err != nil || version != "4.14.0.162" || got != root {
			t.Errorf("InstallVersion(%q) = %q, %q, %v; want 4.14.0.162 of %q", dir, version, got, err, root)
		}
	}
}

func TestInstallVersionSaysWhenThereIsNoInstallationOrItCannotBeRead(t *testing.T) {
	t.Parallel()
	empty := t.TempDir()
	if _, _, err := InstallVersion(empty); !errors.Is(err, ErrNotInstallation) {
		t.Errorf("a directory with no baja.jar: err = %v, want ErrNotInstallation", err)
	}
	bad := t.TempDir()
	if err := os.WriteFile(bad+"/baja.jar", []byte("not a zip"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := InstallVersion(bad); err == nil || errors.Is(err, ErrNotInstallation) {
		t.Errorf("a baja.jar that is not a jar: err = %v, want a read error that is not ErrNotInstallation", err)
	}
	noVersion := writeJar(t, t.TempDir()+"/modules/baja.jar", map[string]string{"META-INF/module.xml": `<module name="baja"/>`})
	if _, _, err := InstallVersion(filepath.Dir(noVersion)); err == nil || errors.Is(err, ErrNotInstallation) {
		t.Errorf("a baja.jar with no vendorVersion: err = %v, want a read error", err)
	}
}

func TestCompiledForIsTheHighestTridiumDependencyFloor(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct{ name, deps, want string }{
		{"one baja floor", `<dependency name="baja" vendor="Tridium" vendorVersion="4.14"/>`, "4.14"},
		{"the highest of several", `<dependency name="baja" vendor="Tridium" vendorVersion="4.14"/><dependency name="alarm-rt" vendor="Tridium" vendorVersion="4.15"/>`, "4.15"},
		{"numeric, not textual", `<dependency name="a" vendor="Tridium" vendorVersion="4.9"/><dependency name="baja" vendor="Tridium" vendorVersion="4.14"/>`, "4.14"},
		{"a four-part floor is cut to major.minor", `<dependency name="baja" vendor="Tridium" vendorVersion="4.14.0.162"/>`, "4.14"},
		{"another vendor's floor is not a Niagara version", `<dependency name="baja" vendor="Tridium" vendorVersion="4.13"/><dependency name="lib" vendor="Acme" vendorVersion="9.2"/>`, "4.13"},
		{"attributes in any order", `<dependency vendorVersion="4.15" vendor="Tridium" name="baja"/>`, "4.15"},
	} {
		got, err := CompiledFor(moduleJar(t, tc.deps))
		if err != nil || got != tc.want {
			t.Errorf("%s: CompiledFor = %q, %v; want %q", tc.name, got, err, tc.want)
		}
	}
}

func TestCompiledForFailsClosedWhenTheJarRecordsNoNiagaraVersion(t *testing.T) {
	t.Parallel()
	for name, jar := range map[string]string{
		"no dependency":          moduleJar(t, ""),
		"only another vendor":    moduleJar(t, `<dependency name="lib" vendor="Acme" vendorVersion="9.2"/>`),
		"a floor with no number": moduleJar(t, `<dependency name="baja" vendor="Tridium" vendorVersion="latest"/>`),
		"no module.xml":          writeJar(t, t.TempDir()+"/x.jar", map[string]string{"a.txt": "x"}),
		"not a zip":              func() string { p := t.TempDir() + "/x.jar"; os.WriteFile(p, []byte("x"), 0o644); return p }(),
		"missing":                t.TempDir() + "/none.jar",
	} {
		if got, err := CompiledFor(jar); err == nil {
			t.Errorf("%s: CompiledFor = %q, want an error", name, got)
		}
	}
}

func TestMatchComparesMajorAndMinor(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		compiled, installed string
		same                bool
	}{
		{"4.14", "4.14.0.162", true},
		{"4.14", "4.14", true},
		{"4.14", "4.15.6.20", false},
		{"4.13", "4.14.0.162", false},
		{"4.9", "4.14.0.162", false},
		{"4.14", "4.140.0.1", false},
		{"4.14", "5.14.0.1", false},
	} {
		same, err := Match(tc.compiled, tc.installed)
		if err != nil || same != tc.same {
			t.Errorf("Match(%q, %q) = %v, %v; want %v", tc.compiled, tc.installed, same, err, tc.same)
		}
	}
	for _, bad := range [][2]string{{"", "4.14.0.1"}, {"4.14", ""}, {"4", "4.14.0.1"}, {"4.14", "x.y"}} {
		if same, err := Match(bad[0], bad[1]); err == nil || same {
			t.Errorf("Match(%q, %q) = %v, %v; want an error and no match", bad[0], bad[1], same, err)
		}
	}
}

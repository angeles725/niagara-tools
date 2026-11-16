package reportmodule

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

// readyFx is a synthetic module tree for report-module --ready: one artifact Mod-rt with a built jar, a deployed
// baseline, a mirror home and a stub toolbelt. No Niagara install, no client data.
type readyFx struct {
	root, art, jar, home, tb string
}

// moduleXML is a META-INF/module.xml shaped like a built jar's: the module part name and its vendorVersion on
// the root tag, the baja floor on a dependency line.
func moduleXML(name, version, bajaFloor string) string {
	return `<?xml version="1.0" encoding="UTF-8"?>` + "\n" +
		`<module name="` + name + `" bajaVersion="0" vendor="V" vendorVersion="` + version + `" moduleName="Mod" runtimeProfile="rt">` + "\n" +
		" <dependencies>\n" + `  <dependency name="baja" vendor="Tridium" vendorVersion="` + bajaFloor + `"/>` + "\n" +
		" </dependencies>\n</module>\n"
}

// writeJar writes a zip at path with the given entries (name -> body).
func writeJar(t *testing.T, path string, entries map[string]string) {
	t.Helper()
	must(t, os.MkdirAll(filepath.Dir(path), 0o755))
	var b bytes.Buffer
	w := zip.NewWriter(&b)
	for _, name := range sortedKeys(entries) {
		f, err := w.Create(name)
		must(t, err)
		_, err = f.Write([]byte(entries[name]))
		must(t, err)
	}
	must(t, w.Close())
	must(t, os.WriteFile(path, b.Bytes(), 0o644))
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	for i := 1; i < len(keys); i++ {
		for j := i; j > 0 && keys[j] < keys[j-1]; j-- {
			keys[j], keys[j-1] = keys[j-1], keys[j]
		}
	}
	return keys
}

// moduleJar is the entries of a module jar signed by signer (its sigfile name), with an extra marker entry so
// two jars of the same version differ in content.
func moduleJar(version, bajaFloor, signer, marker string) map[string]string {
	e := map[string]string{"META-INF/module.xml": moduleXML("Mod-rt", version, bajaFloor), "a/A.class": marker}
	if signer != "" {
		e["META-INF/"+signer+".SF"], e["META-INF/"+signer+".RSA"] = "sf", "rsa"
	}
	return e
}

// newReadyFx builds the tree: a jar 2.8.6 (baja >= 4.14) signed ACMESIGN, a .deploy-baseline, a mirror on
// Niagara 4.15.3.28 whose modules/Mod-rt.jar is the deployed 2.8.5, and stubs that pass.
func newReadyFx(t *testing.T) readyFx {
	t.Helper()
	tmp := t.TempDir()
	fx := readyFx{root: filepath.Join(tmp, "m"), home: filepath.Join(tmp, "home"), tb: filepath.Join(tmp, "tb")}
	fx.art = filepath.Join(fx.root, "Mod-rt")
	fx.jar = filepath.Join(fx.art, "build", "libs", "Mod-rt.jar")
	writeJar(t, fx.jar, moduleJar("2.8.6", "4.14", "ACMESIGN", "new"))
	must(t, os.MkdirAll(filepath.Join(fx.art, ".deploy-baseline"), 0o755))
	must(t, os.MkdirAll(filepath.Join(fx.home, "modules"), 0o755))
	must(t, os.WriteFile(filepath.Join(fx.home, ".niagara-mirror"), nil, 0o644))
	writeJar(t, filepath.Join(fx.home, "modules", "baja.jar"),
		map[string]string{"META-INF/module.xml": `<module name="baja" bajaVersion="0" vendor="Tridium" vendorVersion="4.15.3.28" moduleName="baja">` + "\n</module>\n"})
	writeJar(t, filepath.Join(fx.home, "modules", "Mod-rt.jar"), moduleJar("2.8.5", "4.14", "ACMESIGN", "old"))
	must(t, os.MkdirAll(fx.tb, 0o755))
	fx.stub(t, "verify-module", `printf '%s\n' "$*" > "$(dirname "$0")/verify.args"; printf 'PASS  bytecode  x  ok\n'; exit 0`)
	fx.stub(t, "schema-risk", "exit 0")
	return fx
}

// stub replaces a toolbelt member with a bash body.
func (fx readyFx) stub(t *testing.T, member, body string) {
	t.Helper()
	script(t, fx.tb, member+".sh", "#!/usr/bin/env bash\n"+body+"\n", 0o755)
}

// run is report-module with args (the module root appended) through the stub toolbelt.
func (fx readyFx) run(t *testing.T, args ...string) (code int, out, errOut string) {
	t.Helper()
	var o, e bytes.Buffer
	s := cli.Stdio{Out: &o, Err: &e, Getenv: func(k string) string {
		if k == ToolbeltEnv {
			return fx.tb
		}
		return ""
	}}
	code = mainEnv(append(args, fx.root), s, []string{"PATH=" + os.Getenv("PATH")})
	return code, o.String(), e.String()
}

// sumOf is the full sha256 hex of a file.
func sumOf(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	must(t, err)
	s := sha256.Sum256(data)
	return hex.EncodeToString(s[:])
}

// readyRowOf finds the row of check for the fixture's artifact Mod-rt ("" when none).
func readyRowOf(out, check string) string { return rowOf(out, "Mod-rt", check) }

// rowOf finds the `<artifact>  <ST>  <check>  ...` line of out ("" when none).
func rowOf(out, artifact, check string) string {
	for _, ln := range strings.Split(out, "\n") {
		if f := strings.SplitN(ln, "  ", 4); len(f) >= 3 && f[0] == artifact && f[2] == check {
			return ln
		}
	}
	return ""
}

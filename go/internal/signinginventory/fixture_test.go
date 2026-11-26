package signinginventory

import (
	"archive/zip"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/keytool"
)

// Canary values the tests plant in the fixtures: none of them may ever reach stdout or stderr.
const (
	canaryStore = "St0r3-Canary-Zq9"
	canaryKey   = "K3y-Canary-Xv7"
)

// fx is the throwaway keystores, built once with the real keytool: jceks holds good (code signing, 365 days),
// soon (20 days left), old (expired in 2020) and ca (a CA certificate); p12 holds good only.
var fx struct {
	dir, bin, jceks, p12 string
	skip                 string // why the stores are not there, when they are not
}

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "signing-inventory-test-")
	if err != nil {
		panic(err)
	}
	fx.dir = dir
	buildStores()
	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}

// buildStores makes the fixture stores; with no keytool, or when it fails, fx.skip says why.
func buildStores() {
	bin, err := keytool.Find(os.Getenv("JAVA_HOME"))
	if err != nil {
		fx.skip = "no keytool (set JAVA_HOME to a JDK or put keytool on PATH): the fixture keystores cannot be built"
		return
	}
	fx.bin, fx.jceks, fx.p12 = bin, filepath.Join(fx.dir, "lab.jceks"), filepath.Join(fx.dir, "lab.p12")
	eku := []string{"-ext", "eku=codeSigning", "-ext", "bc=ca:false"}
	for _, k := range []struct {
		store, kind, alias string
		extra              []string
	}{
		{fx.jceks, "JCEKS", "good", append([]string{"-validity", "365"}, eku...)},
		{fx.jceks, "JCEKS", "soon", append([]string{"-validity", "20"}, eku...)},
		{fx.jceks, "JCEKS", "old", append([]string{"-startdate", "2020/01/01 00:00:00", "-validity", "30"}, eku...)},
		{fx.jceks, "JCEKS", "ca", []string{"-validity", "365", "-ext", "bc=ca:true", "-ext", "ku=keyCertSign"}},
		{fx.p12, "PKCS12", "good", append([]string{"-validity", "365"}, eku...)},
	} {
		if err := genkey(k.store, k.kind, k.alias, k.extra...); err != nil {
			fx.skip = "the fixture keystores cannot be built: " + err.Error()
			return
		}
	}
}

// need skips the test when the fixture stores could not be built (no keytool on this host).
func need(t *testing.T) {
	t.Helper()
	if fx.skip != "" {
		t.Skip(fx.skip)
	}
}

func genkey(store, kind, alias string, extra ...string) error {
	args := append([]string{"-genkeypair", "-alias", alias, "-keyalg", "RSA", "-keysize", "2048", "-dname", "CN=" + alias + " signer",
		"-keystore", store, "-storetype", kind, "-storepass", canaryStore, "-keypass", canaryStore}, extra...)
	if out, err := exec.Command(fx.bin, args...).CombinedOutput(); err != nil {
		return fmt.Errorf("keytool %s: %v: %s", alias, err, out)
	}
	return nil
}

// profileXML is a signing profile naming store; pass "" leaves the store password out.
func profileXML(store, kind, pass string) string {
	s := `<?xml version="1.0" encoding="UTF-8" standalone="no"?>` + "\n" +
		`<!DOCTYPE properties SYSTEM "http://java.sun.com/dtd/properties.dtd">` + "\n<properties>\n" +
		"<entry key=\"niagara.signing.storepath\">" + store + "</entry>\n" +
		"<entry key=\"niagara.signing.storetype\">" + kind + "</entry>\n" +
		"<entry key=\"niagara.signing.keypass.good\">" + canaryKey + "</entry>\n"
	if pass != "" {
		s += "<entry key=\"niagara.signing.storepass\">" + pass + "</entry>\n"
	}
	return s + "</properties>\n"
}

// write puts a file named name under dir and returns its path.
func write(t *testing.T, dir, name, text string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

// jar writes a zip of entries (name -> text) at path and returns it.
func jar(t *testing.T, path string, entries map[string]string) string {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, text := range entries {
		w, _ := zw.Create(name)
		w.Write([]byte(text))
	}
	zw.Close()
	return write(t, filepath.Dir(path), filepath.Base(path), buf.String())
}

// inv runs signing-inventory with a fresh HOME and the fixture's JAVA_HOME (env overrides them) and returns
// the exit code, stdout and stderr.
func inv(t *testing.T, env map[string]string, args ...string) (int, string, string) {
	t.Helper()
	var out, errb bytes.Buffer
	e := map[string]string{"HOME": t.TempDir()}
	if fx.bin != "" {
		e["JAVA_HOME"] = filepath.Dir(filepath.Dir(fx.bin))
	}
	for k, v := range env {
		e[k] = v
	}
	code := Main(args, cli.Stdio{Out: &out, Err: &errb, Getenv: func(k string) string { return e[k] }})
	return code, out.String(), errb.String()
}

// has fails unless some row of out holds every part.
func has(t *testing.T, out string, parts ...string) {
	t.Helper()
	for _, line := range strings.Split(out, "\n") {
		ok := true
		for _, p := range parts {
			ok = ok && strings.Contains(line, p)
		}
		if ok {
			return
		}
	}
	t.Errorf("no row with %q in:\n%s", parts, out)
}

// quiet fails when a canary reached the output.
func quiet(t *testing.T, outs ...string) {
	t.Helper()
	for _, o := range outs {
		if strings.Contains(o, canaryStore) || strings.Contains(o, canaryKey) {
			t.Errorf("a canary value was printed:\n%s", o)
		}
	}
}

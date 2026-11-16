package buildmodule

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/fstype"
)

// The tests of this package are serial on purpose: each writes stub scripts and then runs them, and a
// parallel test that forks while a script is still open for writing makes its exec fail with "text file
// busy" (ETXTBSY). Do not add t.Parallel() here.
//
// fixture is a throwaway build: a toolbelt of stub scripts, a module root "Foo" (rt and ux with sources, wb
// a scaffold) with a fake gradlew, a niagara_home and a JDK directory. gradle is never run.
type fixture struct {
	t                    *testing.T
	dir, kit, root, home string
	env                  map[string]string // the environment Main reads
	out, err             string            // streams of the last run
}

// stub is the script every fake command is, with LOGDIR replaced by the fixture directory: it logs its name
// and arguments to calls.log and the environment the chain controls to env.log, then runs the bash text of
// "<script>.do" when a test wrote one.
const stub = `#!/usr/bin/env bash
{ printf '%s' "${0##*/}"; printf ' <%s>' "$@"; echo; } >> "LOGDIR/calls.log"
echo "${0##*/} cwd=$(pwd) JTO=[${JAVA_TOOL_OPTIONS-}] GUH=[${GRADLE_USER_HOME-}] TB=[${BUILD_MODULE_TOOLBELT-unset}]" >> "LOGDIR/env.log"
[ ! -f "$0.do" ] || . "$0.do"
`

func newFixture(t *testing.T) *fixture {
	t.Helper()
	d := t.TempDir()
	f := &fixture{t: t, dir: d, kit: d + "/kit", root: d + "/mod", home: d + "/nh"}
	for _, script := range []string{f.kit + "/" + gateScript, f.kit + "/" + preflightScript, f.kit + "/" + reportScript, f.root + "/gradlew"} {
		f.write(script, strings.ReplaceAll(stub, "LOGDIR", d))
		if err := os.Chmod(script, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	f.write(f.kit+"/"+werrorInit, "")
	f.write(f.kit+"/"+scriptName, fixtureScript)
	for _, p := range []string{"rt", "ux", "wb"} {
		f.write(f.root+"/Foo/Foo-"+p+"/Foo-"+p+".gradle.kts", "")
		f.mkdir(f.root + "/Foo/Foo-" + p + "/src")
	}
	f.write(f.root+"/Foo/Foo-rt/src/com/x/BFoo.java", "")
	f.write(f.root+"/Foo/Foo-ux/src/rc/foo.js", "")
	f.mkdir(f.home + "/" + m2Repository + "/com/tridium/plugin/7.6.17")
	f.mkdir(f.home + "/modules")
	f.mkdir(d + "/j8")
	f.write(d+"/mounts", "rootfs / ext4 rw 0 0\n")
	f.env = map[string]string{
		ToolbeltEnv: f.kit, "JAVA8": d + "/j8", "HOME": d + "/home", fstype.MountsEnv: d + "/mounts",
		"niagara_home": "", "NIAGARA_PLUGIN_VERSION": "", "JAVA_TOOL_OPTIONS": "", "GRADLE_USER_HOME": "",
		gradleEnvPrefix + aliasProperty: "",
	}
	return f
}

// run is Main over args; the streams are kept in f.out and f.err.
func (f *fixture) run(args ...string) int {
	f.t.Helper()
	var out, errs bytes.Buffer
	code := Main(args, cli.Stdio{Out: &out, Err: &errs, Getenv: func(name string) string {
		if v, ok := f.env[name]; ok {
			return v
		}
		return os.Getenv(name)
	}})
	f.out, f.err = out.String(), errs.String()
	return code
}

// signs configures a signing alias, which keeps the no-signer WARN out of a test about other messages.
func (f *fixture) signs() { f.env[gradleEnvPrefix+aliasProperty] = "acmesigner" }

// build runs the chain on the fixture's own module and home, opted in to a marker-less home.
func (f *fixture) build(flags ...string) int {
	f.t.Helper()
	return f.run(append(append([]string{"--install-into-home"}, flags...), f.root, "Foo", f.home)...)
}

func (f *fixture) mkdir(dir string) {
	f.t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		f.t.Fatal(err)
	}
}

func (f *fixture) write(path, content string) {
	f.t.Helper()
	f.mkdir(filepath.Dir(path))
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		f.t.Fatal(err)
	}
}

// read is the content of path, "" when it does not exist.
func (f *fixture) read(path string) string {
	b, _ := os.ReadFile(path)
	return string(b)
}

// does makes a stub script (a toolbelt script name, or "gradlew") run the bash text body when called.
func (f *fixture) does(script, body string) {
	f.t.Helper()
	path := f.kit + "/" + script
	if script == "gradlew" {
		path = f.root + "/gradlew"
	}
	f.write(path+".do", body)
}

// calls is the log of the stub calls, one "<script> <arg>..." line each.
func (f *fixture) calls() string { return f.read(f.dir + "/calls.log") }

// call is the logged line of the first call of script, "" when it never ran.
func (f *fixture) call(script string) string {
	for _, line := range strings.Split(f.calls(), "\n") {
		if strings.HasPrefix(line, script+" ") || line == script {
			return line
		}
	}
	return ""
}

// jar writes a module jar: META-INF/module.xml carrying version on its root tag ("" writes a module with
// only a baja dependency floor), unless payload is "" one payload entry, and the META-INF/<NAME>.SF and .RSA
// files of every signer.
func (f *fixture) jar(path, version, payload string, signers ...string) {
	f.t.Helper()
	xml := `<module name="X"><dependencies><dependency name="baja" vendorVersion="4.13"/></dependencies></module>`
	if version != "" {
		xml = `<module name="X" vendor="A" vendorVersion="` + version + `">` + "\n</module>\n"
	}
	entries := [][2]string{{"META-INF/module.xml", xml}}
	if payload != "" {
		entries = append(entries, [2]string{"com/x/payload.txt", payload})
	}
	for _, s := range signers {
		entries = append(entries, [2]string{"META-INF/" + s + ".SF", "x"}, [2]string{"META-INF/" + s + ".RSA", "x"})
	}
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, e := range entries {
		w, err := zw.Create(e[0])
		if err == nil {
			_, err = w.Write([]byte(e[1]))
		}
		if err != nil {
			f.t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		f.t.Fatal(err)
	}
	f.write(path, buf.String())
}

// deployed and built are the jars of a profile under modules/ and build/libs.
func (f *fixture) deployed(profile string) string { return f.home + "/modules/Foo-" + profile + ".jar" }
func (f *fixture) built(profile string) string {
	return f.root + "/Foo/Foo-" + profile + "/build/libs/Foo-" + profile + ".jar"
}

// wants fails the test unless the last run ended with code and its streams hold every text of has.
func (f *fixture) wants(name string, got, code int, has ...string) {
	f.t.Helper()
	if got != code {
		f.t.Errorf("%s: exit %d, want %d\nstdout: %s\nstderr: %s", name, got, code, f.out, f.err)
	}
	for _, text := range has {
		if !strings.Contains(f.out+f.err, text) {
			f.t.Errorf("%s: output lacks %q\nstdout: %s\nstderr: %s", name, text, f.out, f.err)
		}
	}
}

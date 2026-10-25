package verifymodule

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/lint/rules/golden"
)

// TestGolden replays every run recorded from the bash verify-module.sh (testdata/*.txt).
func TestGolden(t *testing.T) { golden.RunMain(t, Main, 173) }

// write creates every file of tree under dir; a ".jar" key is a zip of its "|"-separated name=text entries.
func write(t *testing.T, dir string, tree map[string]string) {
	t.Helper()
	for name, text := range tree {
		p := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		data := []byte(text)
		if strings.HasSuffix(name, ".jar") {
			var buf bytes.Buffer
			zw := zip.NewWriter(&buf)
			for _, e := range strings.Split(text, "|") {
				n, body, _ := strings.Cut(e, "=")
				w, _ := zw.Create(n)
				w.Write([]byte(body))
			}
			zw.Close()
			data = buf.Bytes()
		}
		if err := os.WriteFile(p, data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func runMain(dir string, args ...string) (int, string, string) {
	for i := range args {
		args[i] = strings.ReplaceAll(args[i], "@DIR@", dir)
	}
	var o, e bytes.Buffer
	code := Main(args, cli.Stdio{Out: &o, Err: &e})
	return code, strings.ReplaceAll(o.String(), dir, "@DIR@"), strings.ReplaceAll(e.String(), dir, "@DIR@")
}

// TestIntegerOperands: the coverage counts are decimal integers (a leading zero is decimal, 08 is 8) and
// one that is not (one past int64) is a usage error, exit 2, naming it. A --plano viewBox is SVG numbers:
// a decimal one is compared in float, and a width or height that is no finite number > 0 (abc, -2) is a
// FAIL row, exit 1. Declared difference: bash arithmetic read 010 as octal 8, and on 08 or 2.5 left the
// coverage or --plano block and parsed the rest of the arguments as the gate's (a non-jar, exit 2 "not a
// jar"); an operand naming an unset variable (abc) ended the run under set -u, exit 1, with no row; a
// negative viewBox passed.
func TestIntegerOperands(t *testing.T) {
	head := append([]byte("\x89PNG\r\n\x1a\n\x00\x00\x00\x0dIHDR"), 0, 0, 0, 2, 0, 0, 0, 3)
	html := func(vb string) string {
		return "<script>IMG_W=2;IMG_H=3</script><svg id=\"zonas\" viewBox=\"" + vb + "\"></svg>" +
			"<img id=\"planoImg\" src=\"data:image/png;base64," + base64.StdEncoding.EncodeToString(head) + "\">\n"
	}
	for _, tc := range []struct {
		name, html  string
		args        []string
		code        int
		out, stderr string
	}{
		{"cov-08", "", []string{"coverage", "08", "1", "0", "0"}, 0, "88.9\n", ""},
		{"cov-010", "", []string{"coverage", "010", "1", "0", "0"}, 0, "90.9\n", ""},
		{"cov-09-unused", "", []string{"coverage", "1", "1", "0", "09"}, 0, "50.0\n", ""},
		{"cov-overflow", "", []string{"coverage", "99999999999999999999", "1", "0", "0"}, 2, "",
			"verify-module: coverage: not an integer: 99999999999999999999\n" + usageCoverage},
		{"plano-08", html("0 0 02 03"), []string{"--plano", "@DIR@/p.html"}, 0,
			"PASS  plano      @DIR@/p.html  Rc=2/3 Ri=2/3 Rv=02/03 — all agree\n", ""},
		{"plano-2.5", html("0 0 2.5 3"), []string{"--plano", "@DIR@/p.html"}, 1,
			"FAIL  plano      @DIR@/p.html  Rv(2.5/3) != Rc(2/3): viewBox vs IMG_W/IMG_H\n", ""},
		{"plano-4.5", html("0 0 4.5 6.75"), []string{"--plano", "@DIR@/p.html"}, 0,
			"PASS  plano      @DIR@/p.html  Rc=2/3 Ri=2/3 Rv=4.5/6.75 — all agree\n", ""},
		{"plano-abc", html("0 0 abc 3"), []string{"--plano", "@DIR@/p.html"}, 1,
			"FAIL  plano      @DIR@/p.html  Rv: viewBox width/height is not a finite number > 0: abc 3\n", ""},
		{"plano-negative", html("0 0 -2 -3"), []string{"--plano", "@DIR@/p.html"}, 1,
			"FAIL  plano      @DIR@/p.html  Rv: viewBox width/height is not a finite number > 0: -2 -3\n", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			write(t, dir, map[string]string{"p.html": tc.html})
			if code, out, stderr := runMain(dir, tc.args...); code != tc.code || out != tc.out || stderr != tc.stderr {
				t.Errorf("exit %d %q %q, want %d %q %q", code, out, stderr, tc.code, tc.out, tc.stderr)
			}
		})
	}
}

// TestStoredByMethod: --stored reads each entry's zip method. A STORED jar whose path or entry name holds
// "Defl:" passes (the bash counted that text in `unzip -v`), and an entry compressed by a method other than
// deflate (bzip2, which unzip -v shows as "BZip2") fails: neither is a STORED entry Workbench can re-sign.
func TestStoredByMethod(t *testing.T) {
	dir := t.TempDir()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, e := range []struct {
		name   string
		method uint16
	}{{"META-INF/module.xml", zip.Store}, {"com/x/A.class", zip.Store}, {"rc/big.txt", 12}} {
		w, err := zw.CreateRaw(&zip.FileHeader{Name: e.name, Method: e.method})
		if err != nil {
			t.Fatal(err)
		}
		w.Write([]byte("x"))
	}
	zw.Close()
	if err := os.WriteFile(filepath.Join(dir, "bz.jar"), buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	_, out, _ := runMain(dir, "--stored", "@DIR@/bz.jar")
	if !strings.Contains(out, "FAIL  stored     @DIR@/bz.jar  1 deflated entries") {
		t.Errorf("a bzip2 entry must fail --stored:\n%s", out)
	}
}

// TestUnreadableInput: an input the gate cannot read gets no verdict. The run stops at the check that
// read it, names it on stderr as "cannot read <path>: <reason>" and exits 3 with no summary line, as an
// unreadable jar does. Declared difference: the bash read a .java through grep/awk 2>/dev/null (empty),
// grepped an unreadable module-include.xml, gradle.kts or --plano file into a verdict, and read an
// unreadable module.palette entry as an empty palette.
func TestUnreadableInput(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root: chmod 000 does not block reads")
	}
	jar := "META-INF/module.xml=<module name=\"Foo\"><types><type name=\"A\" class=\"com.x.A\"/></types></module>|com/x/A.class=\xca\xfe\xba\xbe\x00\x00\x00\x34"
	for _, tc := range []struct{ name, locked, check string }{
		{"java", "mod/Foo-rt/src/com/x/U.java", "facets"},
		{"module-include", "mod/Foo-rt/module-include.xml", "typecount"},
		{"gradle-kts", "mod/Foo-rt/Foo-rt.gradle.kts", "phantom-dep"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			write(t, dir, map[string]string{
				"Foo-rt.jar": jar, "mod/Foo-rt/src/com/x/A.java": "class A {}\n",
				"mod/Foo-rt/src/com/x/U.java": "class U {}\n", "mod/Foo-rt/module-include.xml": "<types/>\n",
				"mod/Foo-rt/Foo-rt.gradle.kts": "dependencies {}\n",
			})
			if err := os.Chmod(filepath.Join(dir, tc.locked), 0); err != nil {
				t.Fatal(err)
			}
			code, out, stderr := runMain(dir, "--src", "@DIR@/mod", "@DIR@/Foo-rt.jar")
			if want := "verify-module: cannot read @DIR@/" + tc.locked + ": Permission denied\n"; code != 3 || stderr != want ||
				strings.Contains(out, "  "+tc.check+" ") || strings.Contains(out, "verify-module:") {
				t.Errorf("exit %d\n%s-- stderr --\n%s", code, out, stderr)
			}
		})
	}
	t.Run("plano", func(t *testing.T) {
		dir := t.TempDir()
		write(t, dir, map[string]string{"p.html": "<script>IMG_W=2;IMG_H=3</script>\n"})
		if err := os.Chmod(filepath.Join(dir, "p.html"), 0); err != nil {
			t.Fatal(err)
		}
		code, out, stderr := runMain(dir, "--plano", "@DIR@/p.html")
		if code != 3 || out != "" || stderr != "verify-module: cannot read @DIR@/p.html: Permission denied\n" {
			t.Errorf("exit %d %q %q", code, out, stderr)
		}
	})
	t.Run("palette-corrupt", func(t *testing.T) {
		dir := t.TempDir()
		var buf bytes.Buffer
		zw := zip.NewWriter(&buf)
		w, _ := zw.Create("META-INF/module.xml")
		w.Write([]byte(`<module name="Foo"><types><type name="A" class="com.x.A"/></types></module>`))
		w, _ = zw.CreateRaw(&zip.FileHeader{Name: "module.palette", Method: zip.Deflate, CRC32: 1, CompressedSize64: 4, UncompressedSize64: 9})
		w.Write([]byte{0xff, 0xff, 0xff, 0xff}) // not a deflate stream
		zw.Close()
		if err := os.WriteFile(filepath.Join(dir, "P-wb.jar"), buf.Bytes(), 0o644); err != nil {
			t.Fatal(err)
		}
		code, out, stderr := runMain(dir, "@DIR@/P-wb.jar")
		if code != 3 || !strings.HasPrefix(stderr, "verify-module: cannot read module.palette in @DIR@/P-wb.jar: ") ||
			strings.Contains(out, "  palette ") || strings.Contains(out, "verify-module:") {
			t.Errorf("exit %d\n%s-- stderr --\n%s", code, out, stderr)
		}
	})
}

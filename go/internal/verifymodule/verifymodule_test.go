package verifymodule

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

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

// TestUnreadableSource pins VM-walk-locked (a chmod 000 cannot be spelled in a golden): a --src
// sub-directory the walk cannot enter is named once on stderr however many checks walk it, the rows of
// the files it did list still print, and the verdict is INCOMPLETE, exit 3. An unreadable .java reads as
// empty, as the bash greps with 2>/dev/null did. Declared difference: the walk line quotes the path in the
// C locale ('...', the shared lint.Walk wording) where GNU find under C.UTF-8 used curly quotes, and the
// bash also printed the awk and test errors for the unreadable file.
func TestUnreadableSource(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root: chmod 000 does not block reads")
	}
	dir := t.TempDir()
	write(t, dir, map[string]string{
		"Foo-rt.jar":                   "META-INF/module.xml=<type name=\"A\" class=\"com.x.A\"/>|com/x/A.class=\xca\xfe\xba\xbe\x00\x00\x00\x34",
		"mod/Foo-rt/src/com/x/A.java":  "String s = \"station:|x\";\n",
		"mod/Foo-rt/src/com/x/U.java":  "@NiagaraProperty(name = \"t\", flags = OPERATOR)\n",
		"mod/Foo-rt/src/locked/L.java": "x\n",
	})
	locked := filepath.Join(dir, "mod/Foo-rt/src/locked")
	for _, p := range []string{locked, filepath.Join(dir, "mod/Foo-rt/src/com/x/U.java")} {
		if err := os.Chmod(p, 0); err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() { _ = os.Chmod(locked, 0o755) })
	code, out, stderr := runMain(dir, "--src", "@DIR@/mod", "@DIR@/Foo-rt.jar")
	want := "verify-module: cannot list every file under @DIR@/mod/Foo-rt/src: find: '@DIR@/mod/Foo-rt/src/locked': Permission denied\n"
	if code != 3 || stderr != want || !strings.Contains(out, "WARN  ord-literal  @DIR@/Foo-rt.jar  @DIR@/mod/Foo-rt/src/com/x/A.java:1:") ||
		!strings.Contains(out, "PASS  facets ") || !strings.HasSuffix(out, "-> INCOMPLETE (a --src walk failed)\n") {
		t.Errorf("exit %d\n%s-- stderr --\n%s", code, out, stderr)
	}
}

// TestArithmeticFallThrough pins what bash did when its arithmetic failed on an operand inside the
// coverage block: it left the block and parsed the rest of the arguments as the gate's (a non-jar is then
// exit 2). stdout and the exit are the bash's; declared difference: bash prefixed its error with
// "<script path>: line <n>: ".
func TestArithmeticFallThrough(t *testing.T) {
	for _, tc := range []struct {
		name, html  string
		args        []string
		code        int
		out, stderr string
	}{
		{"cov-08", "", []string{"coverage", "08", "1", "0", "0"}, 2, "",
			"verify-module: 08: value too great for base (error token is \"08\")\nverify-module: not a jar: 08\n"},
		{"cov-09-unused", "", []string{"coverage", "1", "1", "0", "09"}, 0, "50.0\n", ""},
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

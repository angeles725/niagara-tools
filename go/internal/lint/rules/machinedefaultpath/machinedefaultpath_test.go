package machinedefaultpath

import (
	"bytes"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/lint"
)

// rows runs the rule alone on one file A.java holding src and returns its stdout with the path cut to "A.java".
func rows(t *testing.T, src string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "A.java"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	var o, e bytes.Buffer
	if code := lint.Command(Spec())([]string{ID, dir}, cli.Stdio{Out: &o, Err: &e}); code != 0 || e.Len() != 0 {
		t.Fatalf("exit %d stderr %q, want 0 and none (advisory)", code, e.String())
	}
	return strings.ReplaceAll(o.String(), dir+"/", "")
}

func row(line int, path, where string) string {
	return "ADVISORY  " + ID + "  A.java:" + strconv.Itoa(line) + "  machine-specific default path \"" + path +
		"\" in " + where + " -- take it from a slot, a station-relative ORD, an environment variable or the site table (METHODOLOGY.md § rt)\n"
}

const (
	field = "a field initializer"
	slot  = "an @NiagaraProperty defaultValue"
	fall  = "a fallback argument"
)

// TestFindings: each default context and each machine-path shape the doctrine names.
func TestFindings(t *testing.T) {
	for _, tc := range []struct{ name, src, want string }{
		{"static final field", "class A {\n  static final String DIR = \"/home/cristian/x\";\n}\n", row(2, "/home/cristian/x", field)},
		{"instance field", "class A {\n  private String d = \"/Users/me/x\";\n}\n", row(2, "/Users/me/x", field)},
		{"field on the next line", "class A {\n  static final String D =\n      \"/tmp/run\";\n}\n", row(3, "/tmp/run", field)},
		{"another niagara-prefixed home", "class A {\n  String d = \"/home/niagara2/x\";\n}\n", row(2, "/home/niagara2/x", field)},
		{"bare tmp", "class A {\n  String d = \"/tmp\";\n}\n", row(2, "/tmp", field)},
		{"var tmp", "class A {\n  String d = \"/var/tmp/x\";\n}\n", row(2, "/var/tmp/x", field)},
		{"wsl windows home", "class A {\n  String d = \"/mnt/c/Users/equipo/x\";\n}\n", row(2, "/mnt/c/Users/equipo/x", field)},
		{"windows backslash", "class A {\n  String d = \"C:\\\\Users\\\\me\\\\x\";\n}\n", row(2, `C:\Users\me\x`, field)},
		{"windows slash any drive any case", "class A {\n  String d = \"d:/users/me\";\n}\n", row(2, "d:/users/me", field)},
		{"scratchpad", "class A {\n  String d = \"/srv/claude/Scratchpad/out\";\n}\n", row(2, "/srv/claude/Scratchpad/out", field)},
		{"file uri", "class A {\n  String d = \"file:///home/me/x\";\n}\n", row(2, "file:///home/me/x", field)},
		{"concatenation head", "class A {\n  String d = \"/home/\" + USER + \"/x\";\n}\n", row(2, "/home/", field)},
		{"crlf", "class A {\r\n  String d =\r\n    \"/home/x\";\r\n}\r\n", row(3, "/home/x", field)},
		{"defaultValue", "@NiagaraProperty(name = \"dir\", type = \"baja:String\", defaultValue = \"BString.make(\\\"/home/me/x\\\")\")\nclass A {}\n",
			row(1, "/home/me/x", slot)},
		{"defaultValue ord", "@NiagaraProperty(\n  name = \"f\",\n  type = \"baja:Ord\",\n  defaultValue =\n    \"BOrd.make(\\\"file:/C:/Users/me/f\\\")\"\n)\nclass A {}\n",
			row(5, "file:/C:/Users/me/f", slot)},
		{"getProperty fallback", "class A {\n  void f() { String d = System.getProperty(\"x.dir\", \"/home/me\"); }\n}\n", row(2, "/home/me", fall)},
		{"getProperty key constant", "class A {\n  String f() { return System.getProperty(KEY,\n      \"/tmp/x\"); }\n}\n", row(3, "/tmp/x", fall)},
		{"orElse", "class A {\n  String f() { return Optional.ofNullable(System.getenv(\"D\")).orElse(\"/home/me\"); }\n}\n", row(2, "/home/me", fall)},
		{"getOrDefault", "class A {\n  String f() { return System.getenv().getOrDefault(\"D\", \"/Users/me\"); }\n}\n", row(2, "/Users/me", fall)},
		{"requireNonNullElse", "class A {\n  String f() { return Objects.requireNonNullElse(System.getenv(\"D\"), \"/tmp/d\"); }\n}\n", row(2, "/tmp/d", fall)},
		// The two Java cases of the deferred G43 spec (ledger.d/2026-10-22-modular-line-efficient-quality.md).
		{"ledger defaultValue windows", "@NiagaraProperty(name = \"e\", type = \"baja:String\", defaultValue = \"BString.make(\\\"C:\\\\\\\\Users\\\\\\\\ops\\\\\\\\export\\\")\")\nclass A {}\n",
			row(1, `C:\Users\ops\export`, slot)},
		{"ledger getProperty scratchpad", "class A {\n  String f() { return System.getProperty(\"pw.file\", \"/tmp/scratchpad/pw\"); }\n}\n",
			row(2, "/tmp/scratchpad/pw", fall)},
		{"one row per line", "class A {\n  String a = \"/home/a\", b = \"/home/b\";\n}\n", row(2, "/home/a", field)},
	} {
		if got := rows(t, tc.src); got != tc.want {
			t.Errorf("%s: got\n%s\nwant\n%s", tc.name, got, tc.want)
		}
	}
}

// TestClean: comments, non-default contexts, install roots and relative or station-relative paths.
func TestClean(t *testing.T) {
	for _, tc := range []struct{ name, src string }{
		{"line comment", "class A {\n  // String D = \"/home/x\";\n}\n"},
		{"javadoc", "class A {\n  /** Default \"/home/x\".\n   * String D = \"/tmp/x\"; */\n  String d = \"data/x\";\n}\n"},
		{"block comment defaultValue", "/* @NiagaraProperty(defaultValue = \"BString.make(\\\"/home/x\\\")\") */\nclass A {}\n"},
		{"install root unix", "class A {\n  String d = \"/opt/Niagara/Niagara-4.15\";\n}\n"},
		{"install root windows", "class A {\n  String d = \"C:\\\\Niagara\\\\Niagara-4.15\";\n}\n"},
		{"relative", "class A {\n  String d = \"shared/home/x\";\n  String e = \"tmp/x\";\n}\n"},
		{"station-relative ord", "class A {\n  String d = \"file:^data/x\";\n}\n"},
		{"local in a method", "class A {\n  void f() {\n    String d = \"/home/me\";\n  }\n}\n"},
		{"split concatenation", "class A {\n  String d = \"/ho\" + \"me/x\";\n}\n"},
		{"property key", "class A {\n  void f() { System.getProperty(\"/home/key\"); }\n}\n"},
		{"call argument", "class A {\n  void f() { log(\"/home/me\"); open(x, \"/tmp/y\"); }\n}\n"},
		{"comparison", "class A {\n  boolean b = d == \"/home/x\";\n}\n"},
		{"other annotation element", "@Foo(path = \"/home/x\")\nclass A {}\n"},
		{"defaultValue identifier", "@NiagaraProperty(name = \"d\", type = \"baja:String\", defaultValue = \"BString.make(Scratchpad.DIR)\")\nclass A {}\n"},
		// The JACE platform's own account home, the same on every unit (decompiled niagarad-ext
		// QnxUsbBackupServlet USB_COOKIE_PATH, the one real-tree row of 2026-10-24).
		{"jace platform home", "class A {\n  static final String P = \"/home/niagara/daemon/fp-usb\";\n}\n"},
		{"homepage word", "class A {\n  String d = \"/homepage\";\n}\n"},
		{"text block", "class A {\n  String d = \"\"\"\n    /home/x\n    \"\"\";\n}\n"},
	} {
		if got := rows(t, tc.src); got != "" {
			t.Errorf("%s: got\n%s\nwant no row", tc.name, got)
		}
	}
}

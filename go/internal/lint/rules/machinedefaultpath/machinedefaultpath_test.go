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
func rows(t *testing.T, src string) string { t.Helper(); return rowsIn(t, "A.java", src) }

// rowsIn runs the rule alone on one file at rel (under the root) holding src; the root is cut from the rows.
func rowsIn(t *testing.T, rel, src string) string {
	t.Helper()
	dir := t.TempDir()
	full := filepath.Join(dir, rel)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	var o, e bytes.Buffer
	if code := lint.Command(Spec())([]string{ID, dir}, cli.Stdio{Out: &o, Err: &e}); code != 0 || e.Len() != 0 {
		t.Fatalf("exit %d stderr %q, want 0 and none (WARN without --strict)", code, e.String())
	}
	return strings.ReplaceAll(o.String(), dir+"/", "")
}

func row(line int, path, where string) string { return rowAt("A.java", line, path, where) }

// rowAt is the row the deferred G43 spec names (its text a prefix of the message), plus the remediation hint.
func rowAt(file string, line int, path, where string) string {
	return "WARN  " + ID + "  " + file + ":" + strconv.Itoa(line) + "  machine-specific default \"" + path +
		"\" in " + where + " -- take it from a slot, a station-relative ORD, an environment variable or the site table (METHODOLOGY.md § rt)\n"
}

const (
	field = "a field initializer"
	slot  = "an @NiagaraProperty defaultValue"
	fall  = "a fallback argument"
	decl  = "a const/let/var initializer"
	prop  = "an object-literal property default"
	value = "a .properties value"
)

// TestFindings: each default context and each machine-path shape the doctrine names.
func TestFindings(t *testing.T) {
	for _, tc := range []struct{ name, src, want string }{
		// The deferred G43 spec's first case, verbatim.
		{"ledger static final field", "class A {\n  private static final String PW_FILE = \"/home/dev/.n4/pw\";\n}\n", row(2, "/home/dev/.n4/pw", field)},
		// The spec's prefixes are case-insensitive.
		{"case-insensitive prefix", "class A {\n  String d = \"/TMP/x\";\n  String e = \"/Home/me\";\n}\n", row(2, "/TMP/x", field) + row(3, "/Home/me", field)},
		{"static final field", "class A {\n  static final String DIR = \"/home/someuser/x\";\n}\n", row(2, "/home/someuser/x", field)},
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
		{"ledger station-relative ord", "class A {\n  static final String PW = \"file:^config/pw\";\n}\n"},
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

// TestJavaScript: rc/**/*.js const/let/var initializers and object-literal property defaults.
func TestJavaScript(t *testing.T) {
	const js = "m/src/rc/js/app.js"
	at := func(line int, p, where string) string { return rowAt(js, line, p, where) }
	for _, tc := range []struct{ name, src, want string }{
		{"const", "const DIR = \"/home/me/x\";\n", at(1, "/home/me/x", decl)},
		{"let and var in a function", "(function () {\n  'use strict';\n  let a = '/tmp/a';\n  var b = `C:\\\\Users\\\\me`;\n})();\n",
			at(3, "/tmp/a", decl) + at(4, `C:\Users\me`, decl)},
		{"object property", "var cfg = {\n  url: \"/api\",\n  dir: \"/Users/me/out\",\n  'pw': \"/home/me/.pw\"\n};\n",
			at(3, "/Users/me/out", prop) + at(4, "/home/me/.pw", prop)},
		{"scratchpad", "export const OUT = \"/srv/scratchpad/o\";\n", at(1, "/srv/scratchpad/o", decl)},
		{"comment", "// const D = \"/home/me\";\n/* var e = '/tmp/x'; */\nconst ok = 1;\n", ""},
		{"call argument and assignment", "fetch(\"/home/me\");\nx = \"/tmp/x\";\n", ""},
		{"template with a substitution", "const d = `/home/${user}/x`;\n", ""},
		{"ternary", "var d = a ? b : \"/tmp/x\";\n", ""},
		{"regex holding quotes", "var r = /\"/g; var s = \"ok\";\n", ""},
		{"jace platform home", "const P = \"/home/niagara/x\";\n", ""},
	} {
		if got := rowsIn(t, js, tc.src); got != tc.want {
			t.Errorf("%s: got\n%s\nwant\n%s", tc.name, got, tc.want)
		}
	}
	if got := rowsIn(t, "m/src/tools/gen.js", "const D = \"/home/me\";\n"); got != "" {
		t.Errorf("a .js outside rc/: got\n%s\nwant no row", got)
	}
}

// TestProperties: every .properties value, read as java.util.Properties.load reads it.
func TestProperties(t *testing.T) {
	const pf = "m/gradle.properties"
	for _, tc := range []struct{ name, src, want string }{
		// As the real gradle.properties write it: a properties backslash is doubled.
		// R10-D8: the four host build-location keys are lint-structure L10's; G43 prints no row for them.
		{"windows user home", "niagara_home=C:\\\\Niagara\\\\Niagara-4.14\nniagara_user_home=C:\\\\Users\\\\equipo\\\\Niagara4.14\\\\X\n", ""},
		{"other build-location keys", "niagara_home=/home/me/N4\nnodeHome=C:\\\\Users\\\\me\\\\node\norg.gradle.java.installations.paths=/home/me/jdk8\n", ""},
		{"another key in gradle.properties", "some_other=C:\\\\Users\\\\equipo\\\\x\n", rowAt(pf, 1, `C:\Users\equipo\x`, value)},
		{"colon separator and continuation", "# out=/home/x\nout : /home/\\\n    me/out\nnext=1\n", rowAt(pf, 2, "/home/me/out", value)},
		{"relative and install root", "a=build/out\nb=/opt/Niagara\n", ""},
		{"single backslash loads as C:Users", "u=C:\\Users\\me\n", ""},
	} {
		if got := rowsIn(t, pf, tc.src); got != tc.want {
			t.Errorf("%s: got\n%s\nwant\n%s", tc.name, got, tc.want)
		}
	}
}

// TestBuildKeysOwnedByL10 (R10-D8): the four keys print no row in any gradle.properties, a kit fixture's
// too (lint-structure L10 FAILs those, K16), and exit 0 under --strict; another .properties file still WARNs.
func TestBuildKeysOwnedByL10(t *testing.T) {
	const line = "niagara_user_home=C:\\\\Users\\\\equipo\\\\Niagara4.14\\\\X\n"
	const p = `C:\Users\equipo\Niagara4.14\X`
	for _, tc := range []struct{ name, rel, want string }{
		{"other properties file", "m/local.properties", rowAt("m/local.properties", 1, p, value)},
		{"kit template fixture", "kit/build-n4-module-kit/fixtures/MinimalPan/gradle.properties", ""},
	} {
		if got := rowsIn(t, tc.rel, line); got != tc.want {
			t.Errorf("%s: got\n%s\nwant\n%s", tc.name, got, tc.want)
		}
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "gradle.properties"), []byte(line), 0o644); err != nil {
		t.Fatal(err)
	}
	var o, e bytes.Buffer
	if code := lint.Command(Spec())([]string{ID, "--strict", dir}, cli.Stdio{Out: &o, Err: &e}); code != 0 || o.String() != "" {
		t.Errorf("--strict: exit %d out %q, want 0 and no row", code, o.String())
	}
}

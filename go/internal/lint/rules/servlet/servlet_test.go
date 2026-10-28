package servlet

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/javasrc"
	"github.com/angeles725/niagara-tools/go/internal/lint"
	"github.com/angeles725/niagara-tools/go/internal/lint/rules/golden"
)

// TestGolden: every tests/lint-servlet.bats run and every edge tree, recorded from the bash lint. A case
// whose text starts with a note before "-- args --" is a declared difference from the bash lint.
func TestGolden(t *testing.T) { golden.Run(t, Spec(), 52) }

// TestUnreadable: the bash lint walked with scan_files_nonempty_or_exit, which stops at a sub-directory
// find cannot enter (env line, no row, exit 3), so ContinueOnWalkError stays unset.
func TestUnreadable(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root: chmod 000 does not block reads")
	}
	dir := t.TempDir()
	locked := filepath.Join(dir, "locked")
	if err := os.WriteFile(filepath.Join(dir, "A.java"), []byte("class A extends BWebServlet {\n  void doPost(WebOp op) { c.set(\"a\", 1); }\n}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(locked, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(locked, 0o755) })
	var o, e bytes.Buffer
	code := lint.Command(Spec())([]string{id, dir}, cli.Stdio{Out: &o, Err: &e})
	want := id + ": cannot list every file under " + dir + ": find: '" + locked + "': Permission denied\n"
	if code != 3 || o.Len() != 0 || e.String() != want {
		t.Errorf("exit %d out %q err %q, want 3, no row and %q", code, o.String(), e.String(), want)
	}
}

// check runs the rule on src and returns its rows as "<SEV> <check> <line>".
func check(t *testing.T, src string) []string {
	t.Helper()
	r := Spec().Rule
	fds, err := r.Check(javasrc.NewFile("M/src/A.java", strings.Split(strings.TrimSuffix(src, "\n"), "\n")))
	if err != nil {
		t.Fatalf("Check error: %v", err)
	}
	var got []string
	for _, f := range fds {
		sev := f.Severity
		if sev == lint.Unset {
			sev = r.Severity()
		}
		got = append(got, fmt.Sprintf("%s %s %d", sev, f.Check, f.Line))
	}
	return got
}

// truthCases are the truth fixes made after byte identity, each seeded RED first.
var truthCases = []struct {
	name, src string
	want      []string
}{
	// T8 (truth audit 2026-10-22): the chihuahua BChiServlet.java:41 row anchored on a Javadoc line that
	// documents a guard living in another file.
	{"T8 an X-Requested-With in a Javadoc is not a guard", `/**
 * Security: /api/* requires X-Requested-With: XMLHttpRequest
 */
class A extends BWebServlet {
  void x() { }
}`, nil},
	{"T8 csrf-xrw-only anchors on the code line, not an earlier comment", `class A extends BWebServlet {
  /* checks X-Requested-With */
  void x(WebOp op) { String h = op.getHeader("X-Requested-With"); }
}`, []string{"WARN csrf-xrw-only 3"}},
	{"T8 a csrfToken in a block comment is no CSRF token", `class A extends BWebServlet {
  /* csrfToken
     CsrfUtil */
  void x(WebOp op) { String h = op.getHeader("X-Requested-With"); }
}`, []string{"WARN csrf-xrw-only 4"}},
	{"T8 an X-Content-Type-Options in a block comment sets no header", `class A extends BWebServlet {
  /* setHeader("X-Content-Type-Options", "nosniff") */
  void doGet(WebOp op) { op.getResponse().getWriter(); }
}`, []string{"INFO api-response-headers 3"}},
	{"T8 the header name in a string after an http:// literal still counts", `class A extends BWebServlet {
  void x(WebOp op) { String u = "http://h/"; String h = op.getHeader("X-Requested-With"); }
}`, []string{"WARN csrf-xrw-only 2"}},
	// T2: every other check read raw lines, so a comment or a string literal counted as code.
	{"T2 a parse call in a comment or a literal is no parse", `class A extends BWebServlet {
  // Double.parseDouble(x)
  /* Integer.parseInt(y) */
  String s = "Integer.parseInt(";
}`, nil},
	{"T2 a handler named in a comment is no handler", `// see void doGet( below
class A extends HttpServlet {
  /** void doPost(WebOp op) writes c.set(v) */
  void x() { c.set("a", 1); LOG.info("x"); }
}`, nil},
	{"T2 a servlet declared only in a comment is no servlet", `// class A extends BWebServlet
class A { void x() { Double.parseDouble(s); } }`, nil},
	{"T2 a write, a log call or a response write in a comment is none", `class A extends BWebServlet {
  void doPost(WebOp op) {
    /* c.setDouble("a", 1); */ // c.set("b", 2);
    // LOG.info("x"); op.getResponse().getWriter();
  }
}`, nil},
	{"T2 a brace in a literal does not close the handler", `class A extends BWebServlet {
  void doPost(WebOp op) {
    String b = "}";
    c.setDouble("a", 1);
  }
}`, []string{"FAIL auth 2", "WARN unbounded-set 2"}},
	{"T2 an auth call or a clamp in a comment is no gate", `class A extends BWebServlet {
  void doPost(WebOp op) { c.setDouble("a", 1); } // getRemoteUser() MAX
}`, []string{"FAIL auth 2", "WARN unbounded-set 2"}},
	{"T2 a Cache-Control in a comment is no header", `class A extends BWebServlet {
  // rsp.setHeader("Cache-Control", "max-age=60");
  /* rsp.setHeader("Cache-Control", "max-age=60"); */
}`, nil},
	{"T2 a try in a comment encloses no parse", `class A extends BWebServlet {
  void x(String s) {
    // try {
    Double.parseDouble(s);
  }
}`, []string{"FAIL input-400 4"}},
	{"T2 a 400 in a literal is no 400 response", `class A extends BWebServlet {
  void x(String s) {
    try { Double.parseDouble(s); }
    catch (Exception e) { log("400"); }
  }
}`, []string{"WARN catch-no-400 4"}},
	// T3: the catch was searched from the line after the try's closing line, so the common
	// "} catch (E e) {" on that very line was never read and a silent default passed.
	{"T3 a catch on the try's closing line is read", `class A extends BWebServlet {
  void x(String s) {
    try {
      v = Double.parseDouble(s);
    } catch (NumberFormatException e) {
      v = 0;
    }
  }
}`, []string{"WARN catch-no-400 5"}},
	{"T3 a one-line try/catch is read", `class A extends BWebServlet {
  void x(String s) {
    try { v = Double.parseDouble(s); } catch (Exception e) { v = 0; }
    try { v = Double.parseDouble(s); } catch (Exception e) { rsp.sendError(400); }
  }
}`, []string{"WARN catch-no-400 3"}},
	{"T3 a same-line catch that answers 400 is clean", `class A extends BWebServlet {
  void x(String s) {
    try {
      v = Integer.parseInt(s);
    } catch (NumberFormatException e) {
      op.getResponse().setStatus(SC_BAD_REQUEST);
    }
  }
}`, nil},
}

func TestTruth(t *testing.T) {
	for _, tc := range truthCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := check(t, tc.src); !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got %q, want %q\nsrc:\n%s", got, tc.want, tc.src)
			}
		})
	}
}

// TestHandlerSize (modularity-gaps Δ6): a servlet method spanning more than 80 lines (its name line to its
// closing brace) is one handler-size row: WARN with no map or a legacy map (FAIL under --strict), FAIL in a
// strict module; a method of exactly 80 lines and a long method of a class that is no servlet give none.
func TestHandlerSize(t *testing.T) {
	t.Parallel()
	body := func(n int) string { return strings.Repeat("    x();\n", n) }
	servlet := "package com.x;\npublic class BS extends BWebServlet {\n  public void doPost(WebOp op) {\n" + body(79) +
		"  }\n  private void handleRead(WebOp op) {\n" + body(78) + "  }\n}\n"
	plain := "package com.x;\npublic class Plain {\n  void run() {\n" + body(90) + "  }\n}\n"
	const row = "  handler-size  com/x/BS.java:3  method doPost() spans 81 lines (> 80) inside the servlet -- move its" +
		" route group into a ux/route/<Route>Handler class (types/dashboard.md § DUX1)\n"
	tests := []struct {
		name, mapBody string
		strict        bool
		want          string
		code          int
	}{
		{"no map", "", false, "WARN" + row, 0},
		{"no map, --strict", "", true, "FAIL" + row, 1},
		{"legacy", "modularity: legacy\nlegacy-reason: deployed before the map\n", false, "WARN" + row, 0},
		{"strict module", "modularity: strict\n", false, "FAIL" + row, 1},
	}
	for _, tt := range tests {
		root := t.TempDir()
		src := filepath.Join(root, "M-ux", "src")
		if err := os.MkdirAll(filepath.Join(src, "com", "x"), 0o755); err != nil {
			t.Fatal(err)
		}
		files := map[string]string{"M-ux/src/com/x/BS.java": servlet, "M-ux/src/com/x/Plain.java": plain}
		if tt.mapBody != "" {
			files[lint.MapFile] = tt.mapBody
		}
		for rel, b := range files {
			if err := os.WriteFile(filepath.Join(root, rel), []byte(b), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		args := []string{id, src}
		if tt.strict {
			args = append(args, "--strict")
		}
		var o, e bytes.Buffer
		code := lint.Command(Spec())(args, cli.Stdio{Out: &o, Err: &e})
		if o.String() != tt.want || code != tt.code || e.Len() != 0 {
			t.Errorf("%s: exit %d %q (stderr %q), want exit %d %q", tt.name, code, o.String(), e.String(), tt.code, tt.want)
		}
	}
}

package servlet

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
	"github.com/angeles725/niagara-tools/go/internal/lint"
	"github.com/angeles725/niagara-tools/go/internal/lint/rules/golden"
)

// TestGolden: every tests/lint-servlet.bats run and every edge tree, recorded from the bash lint. A case
// whose text starts with a note before "-- args --" is a declared difference from the bash lint.
func TestGolden(t *testing.T) { golden.Run(t, Spec(), 45) }

// TestUnreadable: the bash find printed its error and the loop scanned every file it listed, so the
// spec sets ContinueOnWalkError (the engine adds the env lines and exit 3).
func TestUnreadable(t *testing.T) {
	golden.Unreadable(t, Spec(), "class A extends BWebServlet {\n  void doPost(WebOp op) { c.set(\"a\", 1); }\n}\n")
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

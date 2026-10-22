package timers

import (
	"bytes"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/lint"
)

// Each testdata/<case>.txt is one tests/lint-timers.bats case (or an edge case) recorded from the bash
// lint-timers.sh it replaces: "-- root --" (the root, relative to the tree), "-- want --" ("exit <n>"
// then the exact stdout of the bash lint, with the tree's absolute path written @DIR@) and one
// "-- <path> --" section per file of the tree.
type golden struct {
	root, want string
	files      map[string]string
}

func readGolden(t *testing.T, path string) golden {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	g := golden{files: map[string]string{}}
	name := ""
	var body strings.Builder
	flush := func() {
		switch name {
		case "":
		case "root":
			g.root = strings.TrimSuffix(body.String(), "\n")
		case "want":
			g.want = body.String()
		default:
			g.files[name] = body.String()
		}
		body.Reset()
	}
	for _, ln := range strings.SplitAfter(string(data), "\n") {
		if strings.HasPrefix(ln, "-- ") && strings.HasSuffix(ln, " --\n") {
			flush()
			name = strings.TrimSuffix(strings.TrimPrefix(ln, "-- "), " --\n")
			continue
		}
		body.WriteString(ln)
	}
	flush()
	return g
}

// runLint runs lint-timers through the engine as the wrapper does and returns "exit <n>\n" + stdout and stderr.
func runLint(t *testing.T, args ...string) (string, string) {
	t.Helper()
	var o, e bytes.Buffer
	code := lint.Command(Spec())(append([]string{ID}, args...), cli.Stdio{Out: &o, Err: &e})
	return "exit " + string(rune('0'+code)) + "\n" + o.String(), e.String()
}

func writeTree(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for p, text := range files {
		full := filepath.Join(dir, p)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// TestGolden: every recorded case prints the bash lint's exact stdout and exit code, and nothing on stderr.
func TestGolden(t *testing.T) {
	paths, _ := filepath.Glob("testdata/*.txt")
	if len(paths) < 60 {
		t.Fatalf("found %d golden cases, want every bats case (>= 60)", len(paths))
	}
	for _, p := range paths {
		t.Run(strings.TrimSuffix(filepath.Base(p), ".txt"), func(t *testing.T) {
			g := readGolden(t, p)
			dir := writeTree(t, g.files)
			got, stderr := runLint(t, filepath.Join(dir, g.root))
			if got = strings.ReplaceAll(got, dir, "@DIR@"); got != g.want || stderr != "" {
				t.Errorf("stderr %q\ngot\n%s\nwant\n%s", stderr, got, g.want)
			}
		})
	}
}

// TestDiscardedUnterminatedLastLine: the bash loop read lines with `read`, which skips a last line with
// no "\n", so a discarded schedule there is not reported (recorded from lint-timers.sh).
func TestDiscardedUnterminatedLastLine(t *testing.T) {
	const head = "class A {\n  void a() { Clock.schedule(this, null, x, null); }\n"
	for name, tc := range map[string]struct{ tail, want string }{
		"terminated":   {"  void b() { Clock.schedule(this, null, x, null); } }\n", "exit 1\n" + strings.Repeat("FAIL  discarded-ticket  @DIR@/A.java: Clock.schedule result is discarded — assign it to a Clock.Ticket field\n", 2)},
		"unterminated": {"  void b() { Clock.schedule(this, null, x, null); } }", "exit 1\nFAIL  discarded-ticket  @DIR@/A.java: Clock.schedule result is discarded — assign it to a Clock.Ticket field\n"},
	} {
		t.Run(name, func(t *testing.T) {
			dir := writeTree(t, map[string]string{"A.java": head + tc.tail})
			got, _ := runLint(t, dir)
			got = strings.ReplaceAll(got, dir, "@DIR@")
			want := tc.want + "FAIL  timer-ticket  @DIR@/A.java: schedules a Clock ticket but has no stopped() override to cancel it\n"
			if got != want {
				t.Errorf("got\n%s\nwant\n%s", got, want)
			}
		})
	}
}

// TestArgs: the argument contract of lint-timers.sh (exactly one root, no flags, usage exits 2).
func TestArgs(t *testing.T) {
	dir := writeTree(t, map[string]string{"x.txt": ""})
	file := filepath.Join(dir, "x.txt")
	for _, tc := range []struct {
		args     []string
		out, err string
	}{
		{nil, "exit 2\n", "usage: lint-timers.sh <java-root>\n"},
		{[]string{dir, dir}, "exit 2\n", "usage: lint-timers.sh <java-root>\n"},
		{[]string{"-h"}, "exit 3\n", "lint-timers: not a directory: -h\n"},
		{[]string{"--strict"}, "exit 3\n", "lint-timers: not a directory: --strict\n"},
		{[]string{file}, "exit 3\n", "lint-timers: not a directory: " + file + "\n"},
		{[]string{dir}, "exit 0\n", ""},
	} {
		if out, err := runLint(t, tc.args...); out != tc.out || err != tc.err {
			t.Errorf("args %q: got %q %q, want %q %q", tc.args, out, err, tc.out, tc.err)
		}
	}
}

// TestUnreadablePermissions: a module-permissions.xml that cannot be read prints the bash message and stops
// the run at that file with exit 3, as the lint-timers.sh header documents. The bash `exit 3` ran inside the
// $(perms_grant) subshell and failed open (the run went on and could exit 0); this port keeps the contract.
func TestUnreadablePermissions(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root")
	}
	dir := writeTree(t, map[string]string{
		"m/module-permissions.xml": "<permissions/>\n",
		"m/src/BX.java":            "public class BX extends BComponent {\n  ExecutorService e = Executors.newSingleThreadExecutor();\n}\n",
	})
	perms := filepath.Join(dir, "m/module-permissions.xml")
	if err := os.Chmod(perms, 0); err != nil {
		t.Fatal(err)
	}
	out, err := runLint(t, filepath.Join(dir, "m/src"))
	if want := "lint-timers: cannot read " + perms + ": awk: fatal: cannot open file `" + perms + "' for reading: Permission denied\n"; out != "exit 3\n" || err != want {
		t.Errorf("got %q %q, want exit 3 and %q", out, err, want)
	}
}

func TestSplitTwo(t *testing.T) {
	for in, want := range map[string][2]string{
		"/m/module-permissions.xml yes":     {"/m/module-permissions.xml", "yes"},
		"- no":                              {"-", "no"},
		"/my mod/module-permissions.xml no": {"/my", "mod/module-permissions.xml no"},
	} {
		if a, b := splitTwo(in); a != want[0] || b != want[1] {
			t.Errorf("splitTwo(%q) = %q, %q; want %q", in, a, b, want)
		}
	}
}

func TestBlankStr(t *testing.T) {
	for in, want := range map[string]string{
		`log("a.cancel()"); b.cancel();`: `log(""); b.cancel();`,
		`c = '\''; d = "x\"y";`:          `c = ''; d = "";`,
		`no literals`:                    `no literals`,
	} {
		if got := blankStr(in); got != want {
			t.Errorf("blankStr(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestGawkOrderDifferential checks gawkOrder against gawk's own `for (k in a)` order, table growth included.
func TestGawkOrderDifferential(t *testing.T) {
	gawk, err := exec.LookPath("gawk")
	if err != nil {
		t.Skip("gawk not installed")
	}
	rnd := rand.New(rand.NewSource(1))
	const first, rest = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ_", "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ_0123456789"
	for _, n := range []int{1, 2, 13, 38, 39, 40, 127, 381, 382, 1200} {
		seen := map[string]bool{}
		var keys []string
		for len(keys) < n {
			k := string(first[rnd.Intn(len(first))])
			for i := rnd.Intn(16); i > 0; i-- {
				k += string(rest[rnd.Intn(len(rest))])
			}
			if !seen[k] {
				seen[k] = true
				keys = append(keys, k)
			}
		}
		cmd := exec.Command(gawk, `{ a[$0] = a[$0] "x" } END { for (k in a) print k }`)
		cmd.Stdin = strings.NewReader(strings.Join(keys, "\n") + "\n")
		out, err := cmd.Output()
		if err != nil {
			t.Fatal(err)
		}
		if want := strings.Fields(string(out)); !reflect.DeepEqual(gawkOrder(keys), want) {
			t.Errorf("%d keys: gawkOrder differs from gawk", n)
		}
	}
}

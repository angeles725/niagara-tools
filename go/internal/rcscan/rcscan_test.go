package rcscan

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/lint/rules/golden"
)

// TestGolden: every tests/rc-scan.bats and tests/report-module.bats run of rc-scan.sh, and every edge tree,
// recorded from the awk script (go/README.md, Adding a lint rule, step 3): exit code, stdout and stderr byte
// for byte. truth-*.txt cases are hand-written after the port, one per truth fix.
func TestGolden(t *testing.T) {
	paths, _ := filepath.Glob("testdata/*.txt")
	if len(paths) < 40 {
		t.Fatalf("found %d golden cases, want at least 40", len(paths))
	}
	for _, p := range paths {
		t.Run(strings.TrimSuffix(filepath.Base(p), ".txt"), func(t *testing.T) {
			t.Parallel()
			c, err := golden.Read(p)
			if err != nil {
				t.Fatal(err)
			}
			dir := t.TempDir()
			for f, text := range c.Files {
				full := filepath.Join(dir, f)
				if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(full, []byte(text), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			var args []string
			for _, a := range c.Args {
				args = append(args, strings.ReplaceAll(a, "@DIR@", dir))
			}
			var o, e bytes.Buffer
			code := Main(args, cli.Stdio{Out: &o, Err: &e})
			got := strings.ReplaceAll(fmt.Sprintf("exit %d\n%s", code, o.String()), dir, "@DIR@")
			if stderr := strings.ReplaceAll(e.String(), dir, "@DIR@"); got != c.Want || stderr != c.Stderr {
				t.Errorf("got\n%s-- stderr --\n%s\nwant\n%s-- stderr --\n%s", got, stderr, c.Want, c.Stderr)
			}
		})
	}
}

// TestFailClosed: an unreadable sub-directory stops the scan before any row (RS-locked), and an unreadable
// file, where gawk printed its fatal error and the script went on, is an env error too.
func TestFailClosed(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root: chmod 000 does not block reads")
	}
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "rc"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "rc", "a.js"), []byte("var a = 'station:|x';\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	locked := filepath.Join(dir, "locked")
	if err := os.Mkdir(locked, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(locked, 0o755) })
	run := func() (int, string, string) {
		var o, e bytes.Buffer
		code := Main([]string{dir}, cli.Stdio{Out: &o, Err: &e})
		return code, o.String(), e.String()
	}
	if code, out, err := run(); code != 3 || out != "" || err != "rc-scan: cannot list every file under "+dir+": find: '"+locked+"': Permission denied\n" {
		t.Errorf("locked dir: exit %d out %q err %q", code, out, err)
	}
	if err := os.Chmod(locked, 0o755); err != nil {
		t.Fatal(err)
	}
	u := filepath.Join(dir, "rc", "u.js")
	if err := os.WriteFile(u, nil, 0); err != nil {
		t.Fatal(err)
	}
	if code, out, err := run(); code != 3 || out != "" || err != "rc-scan: cannot scan "+u+": Permission denied\n" {
		t.Errorf("unreadable file: exit %d out %q err %q", code, out, err)
	}
}

// TestHandMatchers: ordLiteral, ipv4 and the login test are the awk EREs matched by hand (perf); they must
// agree with the regexp on every shape, including line starts, segment quotes and broken dot runs.
func TestHandMatchers(t *testing.T) {
	t.Parallel()
	ord := regexp.MustCompile("(station:|local:|slot:/)|(^|[|\"'`])h:[0-9a-fA-F]+")
	ip := regexp.MustCompile(`[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+`)
	login := regexp.MustCompile(`login|step-?up`)
	for _, s := range []string{"", "h:1", "xh:1", "|h:f", "\"h:G", "'h:a0", "`h:", "width:100px;h:9", "a|h:z h:1 'h:F",
		"station:", "local:|", "slot:x", "slot:/", "1.2.3.4", "1.2.3", "1..2.3.4", "1.2.3.4.5", ".1.2.3.4", "a1.2.3.4b",
		"1.2.3.x.4", "12.34.56.78", "1.2. 3.4", "v1.2.3", "1.2.3.4.", "x.1.2..3.4.5.6", "LOGIN", "step-up", "stepup",
		"step up", "Step-Up", "logi n"} {
		if got, want := ordLiteral(s), ord.MatchString(s); got != want {
			t.Errorf("ordLiteral(%q) = %v, the ERE says %v", s, got, want)
		}
		if got, want := ipv4(s), ip.MatchString(s); got != want {
			t.Errorf("ipv4(%q) = %v, the ERE says %v", s, got, want)
		}
		if got, want := mentionsGate(s) && !strings.Contains(asciiLower(s), "auth"), login.MatchString(asciiLower(s)); got != want {
			t.Errorf("mentionsGate(%q) login half = %v, the ERE says %v", s, got, want)
		}
	}
}

// TestJSFileSize (modularity-gaps Δ3): under a module map a .js file over 400 lines is one row at its line 1
// (WARN legacy, FAIL strict); exactly 400 lines, an allow marker on line 1 and a map-less module give none.
func TestJSFileSize(t *testing.T) {
	t.Parallel()
	lines := func(n int) string { return strings.Repeat("var v = 1;\n", n) }
	const row = "  rc-scan  src/rc/js/big.js:1  js-file-size: 401 lines (> 400) -- split by concern into services/, store," +
		" components/ and pages/ files (types/frontend-standard.md § Component layout)\n"
	tests := []struct {
		name, mapBody, want string
		code                int
	}{
		{"no map", "", "", 0},
		{"legacy", "modularity: legacy\nlegacy-reason: deployed before the map\n", "WARN" + row, 0},
		{"strict", "modularity: strict\n", "FAIL" + row, 1},
	}
	for _, tt := range tests {
		root := t.TempDir()
		js := filepath.Join(root, "M-ux", "src", "rc", "js")
		if err := os.MkdirAll(js, 0o755); err != nil {
			t.Fatal(err)
		}
		files := map[string]string{"big.js": lines(401), "at-budget.js": lines(400),
			"allowed.js": "// rc-scan: allow js-file-size (a generated table)\n" + lines(400)}
		if tt.mapBody != "" {
			files["../../../../MODULE-MAP.md"] = tt.mapBody
		}
		for name, body := range files {
			if err := os.WriteFile(filepath.Join(js, name), []byte(body), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		var o, e bytes.Buffer
		code := Main([]string{filepath.Join(root, "M-ux")}, cli.Stdio{Out: &o, Err: &e})
		if o.String() != tt.want || code != tt.code || e.Len() != 0 {
			t.Errorf("%s: exit %d %q (stderr %q), want exit %d %q", tt.name, code, o.String(), e.String(), tt.code, tt.want)
		}
	}
}

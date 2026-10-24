package citeresolve

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

func run(t *testing.T, args ...string) (int, string, string) {
	t.Helper()
	var out, errb bytes.Buffer
	code := Main(args, cli.Stdio{Out: &out, Err: &errb, Getenv: func(string) string { return "" }})
	return code, out.String(), errb.String()
}

// rootArgs points every root at the fixtures, plus extra arguments.
func rootArgs(t *testing.T, extra ...string) []string {
	r := fixtureRoots(t)
	return append([]string{"--kit", r.Kit, "--research", r.Research, "--modules", r.Modules, "--memory", r.Memory}, extra...)
}

func TestMainExitCodes(t *testing.T) {
	t.Parallel()
	ok, mixed := filepath.Join("testdata", "kit", "types", "ok.md"), filepath.Join("testdata", "kit", "types", "mixed.md")
	cases := []struct {
		name string
		args []string
		code int
		out  string // a substring of stdout
		err  string // a substring of stderr
	}{
		{"all resolved", rootArgs(t, ok), 0, "\tRESOLVED\t", "4 pointers: RESOLVED 4"},
		{"all resolved under --strict", rootArgs(t, "--strict", ok), 0, "", ""},
		{"findings without --strict", rootArgs(t, mixed), 0, "\tBARE_ID\t", ""},
		{"findings under --strict", rootArgs(t, "--strict", mixed), 1, "\tMISSING_DELTA\t", "MISSING_SECTION 1"},
		{"row columns", rootArgs(t, mixed), 0, mixed + "\t3\tcorpus\tB1023\t§ND4\tMISSING_SECTION\tB1023 has no §ND4\n", ""},
		{"no card", rootArgs(t), 3, "", "usage:"},
		{"unknown format", rootArgs(t, "--format", "xml", ok), 3, "", "--format"},
		{"unknown flag", rootArgs(t, "--nope", ok), 3, "", "usage:"},
		{"missing research root", append(rootArgs(t), "--research", "testdata/none", ok), 3, "", "research root"},
		{"unreadable card", rootArgs(t, "testdata/kit/types/none.md"), 3, "", "cannot read card"},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			code, out, errs := run(t, tt.args...)
			if code != tt.code || !strings.Contains(out, tt.out) || !strings.Contains(errs, tt.err) {
				t.Errorf("exit %d, stdout %q, stderr %q; want exit %d, stdout ~%q, stderr ~%q", code, out, errs, tt.code, tt.out, tt.err)
			}
		})
	}
}

func TestMainUnreadableRoot(t *testing.T) {
	t.Parallel()
	if os.Geteuid() == 0 {
		t.Skip("root reads a mode-000 directory")
	}
	dir := filepath.Join(t.TempDir(), "research")
	if err := os.Mkdir(dir, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })
	code, out, errs := run(t, append(rootArgs(t), "--research", dir, filepath.Join("testdata", "kit", "types", "ok.md"))...)
	if code != 3 || out != "" || !strings.Contains(errs, "cannot read research root") {
		t.Errorf("exit %d, stdout %q, stderr %q; want exit 3, no rows, a research-root error", code, out, errs)
	}
}

func TestMainJSON(t *testing.T) {
	t.Parallel()
	code, out, _ := run(t, rootArgs(t, "--format", "json", filepath.Join("testdata", "kit", "types", "mixed.md"))...)
	var rows []map[string]any
	if err := json.Unmarshal([]byte(out), &rows); code != 0 || err != nil || len(rows) != 4 {
		t.Fatalf("exit %d, %d rows, err %v:\n%s", code, len(rows), err, out)
	}
	if rows[0]["status"] != "MISSING_DELTA" || rows[0]["line"] != float64(1) || rows[0]["kind"] != "retro" {
		t.Errorf("row 0 = %v", rows[0])
	}
}

func TestDefaultKitFromCard(t *testing.T) {
	t.Parallel()
	r := fixtureRoots(t)
	args := []string{"--research", r.Research, "--modules", r.Modules, "--memory", r.Memory, filepath.Join("testdata", "kit", "types", "ok.md")}
	if code, _, errs := run(t, args...); code != 0 {
		t.Errorf("exit %d, stderr %q; want the kit found above the card", code, errs)
	}
}

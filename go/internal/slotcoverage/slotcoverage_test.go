package slotcoverage

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"

	"github.com/angeles725/niagara-tools/go/internal/lint/rules/golden"
)

// TestGolden replays every run recorded from the bash slot-coverage.sh (testdata/*.txt).
func TestGolden(t *testing.T) { golden.RunMain(t, Main, 60) }

// unreadableTree writes the tree TestUnreadable runs on: one type, two lexicon keys and three sources.
func unreadableTree(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for name, text := range map[string]string{
		"module-include.xml":    "<types>\n<type name=\"Thing\" class=\"com.x.BThing\"/>\n</types>\n",
		"module.lexicon":        "fanMode=Fan Mode\ntagZ=Z\n",
		"src/com/x/BA.java":     "@NiagaraProperty(name = \"aSlot\", flags = Flags.OPERATOR)\n",
		"src/com/x/BThing.java": "@NiagaraProperty(name = \"setpoint\", flags = Flags.OPERATOR)\n",
		"src/com/x/BZ.java":     "@NiagaraProperty(name = \"fanMode\", flags = Flags.OPERATOR)\n@Range(\"tagZ\")\n",
	} {
		p := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// TestUnreadable pins what the bash printed (recorded by hand from origin/main) for an input it cannot
// read: a grep error line per grep and a verdict on what was read; an unreadable source ends the awk scan
// (BZ's fanMode slot is never read, so its key is STALE) while the @Range grep reads on.
func TestUnreadable(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root: chmod 000 does not block reads")
	}
	for _, tc := range []struct {
		name, locked string
		perSlot      bool
		code         int
		out, err     string
	}{
		{"parse-xml", "module-include.xml", false, 0, "pct=N/A (type-set)\nmissing=\nextra=\n",
			"grep: @DIR@/module-include.xml: Permission denied\n"},
		{"perslot-xml", "module-include.xml", true, 1, "pct=33.3 (per-slot)\nMISSING aSlot\nMISSING setpoint\n",
			"grep: @DIR@/module-include.xml: Permission denied\n"},
		{"parse-lex", "module.lexicon", false, 1,
			"slot-coverage: FAIL empty lexicon with 1 declared type(s)\npct=0.0 (type-set)\nmissing=Thing\nextra=\n",
			"grep: @DIR@/module.lexicon: Permission denied\ngrep: @DIR@/module.lexicon: Permission denied\n"},
		{"perslot-lex", "module.lexicon", true, 1, "pct=0.0 (per-slot)\nMISSING aSlot\nMISSING fanMode\nMISSING setpoint\n",
			"grep: @DIR@/module.lexicon: Permission denied\n"},
		{"perslot-java", "src/com/x/BThing.java", true, 1, "pct=0.0 (per-slot)\nMISSING aSlot\nSTALE fanMode\n", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := unreadableTree(t)
			if err := os.Chmod(filepath.Join(dir, tc.locked), 0); err != nil {
				t.Fatal(err)
			}
			args := []string{dir + "/module-include.xml", dir + "/module.lexicon"}
			if tc.perSlot {
				args = append(append([]string{"per-slot"}, args...), dir+"/src")
			}
			var o, e bytes.Buffer
			code := Main(args, cli.Stdio{Out: &o, Err: &e})
			out, stderr := strings.ReplaceAll(o.String(), dir, "@DIR@"), strings.ReplaceAll(e.String(), dir, "@DIR@")
			if code != tc.code || out != tc.out || stderr != tc.err {
				t.Errorf("exit %d\n%s-- stderr --\n%s\nwant exit %d\n%s-- stderr --\n%s", code, out, stderr, tc.code, tc.out, tc.err)
			}
		})
	}
}

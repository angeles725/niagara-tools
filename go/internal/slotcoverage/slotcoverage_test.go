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

// TestUnreadable pins the fail-closed read: a module-include.xml, lexicon or source that cannot be read is
// exit 3, its path and the reason on stderr, nothing on stdout. Declared difference: the bash printed
// grep's error and a verdict on the rest (an unreadable source even ended the slot scan: pct=N/A, exit 0).
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
		{"parse-xml", "module-include.xml", false, 3, "", "slot-coverage: cannot read @DIR@/module-include.xml: Permission denied\n"},
		{"perslot-xml", "module-include.xml", true, 3, "", "slot-coverage: cannot read @DIR@/module-include.xml: Permission denied\n"},
		{"parse-lex", "module.lexicon", false, 3, "", "slot-coverage: cannot read @DIR@/module.lexicon: Permission denied\n"},
		{"perslot-lex", "module.lexicon", true, 3, "", "slot-coverage: cannot read @DIR@/module.lexicon: Permission denied\n"},
		{"perslot-java", "src/com/x/BThing.java", true, 3, "", "slot-coverage: cannot read @DIR@/src/com/x/BThing.java: Permission denied\n"},
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

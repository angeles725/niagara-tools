package portparity

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

func run(args ...string) (code int, out, errs string) {
	var o, e bytes.Buffer
	code = Main(args, cli.Stdio{Out: &o, Err: &e})
	return code, o.String(), e.String()
}

// module lays out <root>/ref (the reference) and <root>/mod/src/rc (the port) with a .port-parity in mod/.
func module(t *testing.T, waivers string) (root, mod string) {
	t.Helper()
	root = t.TempDir()
	write(t, root, map[string]string{
		"ref/index.html":        `<div id="levels"></div><div id="equipmentView"></div>`,
		"ref/ui.js":             "function renderLevels() {}\nfunction renderEquipment() {}\n",
		"mod/src/rc/index.html": `<div id="levels"></div>`,
		"mod/src/rc/ui.js":      "function renderLevels() {}\n",
	})
	if waivers != "" {
		write(t, root, map[string]string{"mod/.port-parity": waivers})
	}
	return root, filepath.Join(root, "mod")
}

const header = "ref: ../ref\nport: src/rc\n"

func TestMainModuleMode(t *testing.T) {
	t.Parallel()
	_, mod := module(t, header+"id equipmentView -- own page [decision: W5]\nfunction renderEquipment -- own page [decision: W5]\n")
	code, out, errs := run(mod)
	if code != ExitOK || errs != "" {
		t.Fatalf("exit %d, stderr %q", code, errs)
	}
	if want := "port-parity: 2 files, 2 ids, 2 functions in the reference; 2 missing, 2 waived, 0 FAIL, 0 WARN\n"; out != want {
		t.Errorf("out = %q, want %q", out, want)
	}
}

func TestMainModuleModeFail(t *testing.T) {
	t.Parallel()
	_, mod := module(t, header+"id equipmentView -- own page [decision: W5]\n")
	code, out, _ := run(mod)
	if code != ExitFail {
		t.Fatalf("exit %d, want %d", code, ExitFail)
	}
	if !strings.HasPrefix(out, "FAIL  lint-port-parity  ui.js:2  missing function renderEquipment (in the reference, not in the port, no waiver)\n") {
		t.Errorf("out = %q", out)
	}
}

func TestMainFlagsOverrideHeadersAndJSON(t *testing.T) {
	t.Parallel()
	root, mod := module(t, "")
	write(t, root, map[string]string{"w.txt": "ref: nowhere\nfunction renderEquipment -- gone [decision: W5]\n"})
	code, out, errs := run("--ref", filepath.Join(root, "ref"), "--port", filepath.Join(mod, "src", "rc"),
		"--waivers", filepath.Join(root, "w.txt"), "--format", "json")
	if code != ExitFail || errs != "" {
		t.Fatalf("exit %d, stderr %q", code, errs)
	}
	var doc struct {
		Counts   Counts    `json:"counts"`
		Findings []Finding `json:"findings"`
	}
	if err := json.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatalf("not JSON: %v\n%s", err, out)
	}
	if doc.Counts.Missing != 2 || doc.Counts.Waived != 1 || doc.Counts.Fail != 1 || len(doc.Findings) != 1 ||
		doc.Findings[0].Kind != KindID || doc.Findings[0].Name != "equipmentView" {
		t.Errorf("doc = %+v", doc)
	}
}

func TestMainCleanJSONHasEmptyFindings(t *testing.T) {
	t.Parallel()
	root, _ := module(t, "")
	code, out, _ := run("--ref", filepath.Join(root, "ref"), "--port", filepath.Join(root, "ref"), "--format", "json")
	if code != ExitOK || !strings.Contains(out, `"findings": []`) {
		t.Errorf("exit %d out %s", code, out)
	}
}

func TestMainEnvErrors(t *testing.T) {
	t.Parallel()
	root, mod := module(t, "")
	_, withBadRef := module(t, "ref: ../absent\n")
	_, noRef := module(t, "port: src/rc\n")
	tests := []struct {
		name string
		args []string
		errs string
	}{
		{"no arguments", nil, usage},
		{"unknown flag", []string{"--strict", mod}, usage},
		{"flag without value", []string{"--ref"}, usage},
		{"ref without port", []string{"--ref", root}, usage},
		{"bad format", []string{"--format", "xml", mod}, "port-parity: --format must be text or json, not xml\n"},
		{"module without a parity file", []string{mod}, "port-parity: no .port-parity in " + mod + " (the module declares no reference port)\n"},
		{"parity file without ref", []string{noRef}, "port-parity: " + filepath.Join(noRef, ".port-parity") + ": no ref: header\n"},
		{"unreadable reference", []string{withBadRef}, "port-parity: cannot read the reference: "},
		{"missing waiver file", []string{"--ref", root, "--port", root, "--waivers", filepath.Join(root, "absent")}, "port-parity: cannot read "},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			code, out, errs := run(tt.args...)
			if code != ExitEnv || out != "" || !strings.HasPrefix(errs, tt.errs) {
				t.Errorf("exit %d out %q errs %q, want exit 3 and errs prefix %q", code, out, errs, tt.errs)
			}
		})
	}
}

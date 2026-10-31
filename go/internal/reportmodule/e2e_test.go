package reportmodule

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

// realMembers runs report-module --lint-only on root with only the named members real (every other toolbelt
// member a no-op stub, the golden cases' toolbelt/.stub-rest) and returns its exit code and stdout.
func realMembers(t *testing.T, root string, real ...string) (int, string) {
	t.Helper()
	repo, err := filepath.Abs("../../..")
	must(t, err)
	dir := t.TempDir()
	tb := filepath.Join(dir, "toolbelt")
	must(t, os.MkdirAll(tb, 0o755))
	must(t, os.WriteFile(filepath.Join(tb, ".stub-rest"), nil, 0o644))
	for _, m := range real {
		must(t, os.Symlink(filepath.Join(repo, "build-n4-module-kit", "toolbelt", m+".sh"), filepath.Join(tb, m+".sh")))
	}
	toolbeltCopy(t, repo, dir)
	var o, e bytes.Buffer
	s := cli.Stdio{Out: &o, Err: &e, Getenv: func(k string) string {
		if k == ToolbeltEnv {
			return tb
		}
		return os.Getenv(k)
	}}
	code := mainEnv([]string{"--lint-only", root}, s, os.Environ())
	return code, o.String() + e.String()
}

// moduleTree writes a module root m (files relative to it) under a temp dir.
func moduleTree(t *testing.T, files map[string]string) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), "m")
	for rel, body := range files {
		p := filepath.Join(root, rel)
		must(t, os.MkdirAll(filepath.Dir(p), 0o755))
		must(t, os.WriteFile(p, []byte(body), 0o644))
	}
	return root
}

const e2eMap = "modularity: strict\n\n| Concern | Part | Files | Tests | Change here when |\n|---|---|---|---|---|\n" +
	"| control | rt | A-rt/src/a/BBig.java | A-rt/srcTest/a/BBigTest.java | the decision |\n"

// TestStrictSizeRelayEndToEnd (strict-new Δ4): a `modularity: strict` module whose B-class has an 81-line method
// gets lint-size's FAIL row relayed as FAIL, no PASS lint-size row, the ISSUES verdict and exit 1.
func TestStrictSizeRelayEndToEnd(t *testing.T) {
	if testing.Short() {
		t.Skip("runs toolbelt members as subprocesses")
	}
	body := "package a;\nimport javax.baja.sys.BComponent;\npublic final class BBig extends BComponent {\n  void run() {\n" +
		strings.Repeat("    int x = 1;\n", 79) + "  }\n}\n"
	root := moduleTree(t, map[string]string{"MODULE-MAP.md": e2eMap, "A-rt/src/a/BBig.java": body, "A-rt/srcTest/a/BBigTest.java": "class BBigTest {}\n"})
	code, out := realMembers(t, root, "lint-size", "lint-module-map")
	if code != ExitFail || !strings.Contains(out, "A-rt  FAIL  lint-size  ") || strings.Contains(out, "PASS  lint-size") ||
		!strings.Contains(out, "(module)  PASS  lint-module-map  clean") || !strings.Contains(out, " · 1 FAIL · ") || !strings.Contains(out, "->  ISSUES") {
		t.Errorf("exit %d, want %d with a relayed lint-size FAIL row and the ISSUES verdict:\n%s", code, ExitFail, out)
	}
}

// TestInvalidMapRelayEndToEnd: a strict header with no valid table is lint-module-map's MM2 FAIL row, relayed (not
// only the verdict), and the modularity row keeps the header's mode with no counts.
func TestInvalidMapRelayEndToEnd(t *testing.T) {
	if testing.Short() {
		t.Skip("runs toolbelt members as subprocesses")
	}
	root := moduleTree(t, map[string]string{"MODULE-MAP.md": "modularity: strict\n\nno table yet\n", "A-rt/src/a/A.java": "class A {}\n"})
	code, out := realMembers(t, root, "lint-module-map")
	if code != ExitFail || !strings.Contains(out, "(module)  FAIL  lint-module-map  MODULE-MAP.md  MM2: no | Concern") ||
		!strings.Contains(out, "(module)  INFO  modularity  mode=strict concerns=- unlisted=- largest=A-rt/src/a/A.java") {
		t.Errorf("exit %d, want %d with the MM2 row relayed and mode=strict:\n%s", code, ExitFail, out)
	}
}

// TestPortParityRelayEndToEnd (retro reference-port-parity Δ2): a module that declares a reference port in
// .port-parity gets lint-port-parity's FAIL row for an unwaived absence, the ISSUES verdict and exit 1; a module
// without the file gets no lint-port-parity row at all (most modules port nothing).
func TestPortParityRelayEndToEnd(t *testing.T) {
	if testing.Short() {
		t.Skip("runs toolbelt members as subprocesses")
	}
	t.Parallel()
	root := moduleTree(t, map[string]string{
		"../ref/ui.js":      "function renderLevels() {}\nfunction renderEquipment() {}\n",
		"A-ux/src/rc/ui.js": "function renderLevels() {}\n",
		".port-parity":      "ref: ../ref\nport: A-ux/src/rc\n",
	})
	code, out := realMembers(t, root, "lint-port-parity")
	if code != ExitFail || !strings.Contains(out, "(module)  FAIL  lint-port-parity  ui.js:2  missing function renderEquipment") ||
		!strings.Contains(out, "->  ISSUES") {
		t.Errorf("exit %d, output:\n%s", code, out)
	}
	must(t, os.Remove(filepath.Join(root, ".port-parity")))
	code, out = realMembers(t, root, "lint-port-parity")
	if code != 0 || strings.Contains(out, "lint-port-parity") {
		t.Errorf("no .port-parity: exit %d, output:\n%s", code, out)
	}
}

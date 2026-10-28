package verifymodule

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// slotWallRow runs the gate with --src on the driver fixture copy in dir and returns its exit code and its
// slot-wall row.
func slotWallRow(t *testing.T, dir string) (int, string) {
	t.Helper()
	write(t, dir, map[string]string{strings.SplitN(fxJar, "=", 2)[0]: strings.SplitN(fxJar, "=", 2)[1]})
	code, out, _ := runMain(dir, "--src", "@DIR@/mod", "@DIR@/MinimalDriver-rt.jar")
	for _, ln := range strings.Split(out, "\n") {
		if f := strings.Fields(ln); len(f) >= 2 && f[1] == "slot-wall" {
			return code, ln
		}
	}
	t.Fatalf("no slot-wall row in\n%s", out)
	return 0, ""
}

// TestSlotWallStrict: the scaffolded (strict) driver passes slot-wall; a 16-slot class FAILs it in the strict
// module and fails the gate, and only WARNs once the module map is gone (a legacy, map-less module).
func TestSlotWallStrict(t *testing.T) {
	dir := copyFixture(t)
	if _, row := slotWallRow(t, dir); !strings.HasPrefix(row, "PASS") {
		t.Fatalf("fixture slot-wall row %q, want PASS", row)
	}
	var b strings.Builder
	b.WriteString("package com.angeles.MinimalDriver;\n@NiagaraType\n")
	for i := 0; i < 16; i++ {
		b.WriteString("@NiagaraProperty(name = \"p" + string(rune('a'+i)) + "\", type = \"int\", defaultValue = \"0\")\n")
	}
	b.WriteString("public class BWide extends BComponent {}\n")
	write(t, dir, map[string]string{fxRt + "BWide.java": b.String()})
	code, row := slotWallRow(t, dir)
	if code != 1 || !strings.HasPrefix(row, "FAIL") || !strings.Contains(row, "16 properties in class BWide") {
		t.Errorf("strict: exit %d row %q, want exit 1 and a FAIL row for BWide", code, row)
	}
	if err := os.Remove(filepath.Join(dir, "mod", "MODULE-MAP.md")); err != nil {
		t.Fatal(err)
	}
	if _, row = slotWallRow(t, dir); !strings.HasPrefix(row, "WARN") {
		t.Errorf("no map: row %q, want the legacy WARN", row)
	}
}
